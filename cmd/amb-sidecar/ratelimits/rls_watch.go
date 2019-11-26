package rls

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/mapstructure"
)

var rlslog dlog.Logger

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

type RateLimitController struct {
	lock   sync.RWMutex // everything after this is guarded by the lock
	limits []k8s.Resource
}

func (r *RateLimitController) withLock(doit func()) {
	r.lock.Lock()
	defer r.lock.Unlock()
	doit()
}

func (r *RateLimitController) setLimits(limits []k8s.Resource) {
	r.withLock(func() {
		r.limits = limits
	})
}

func (r *RateLimitController) GetLimits() (result []k8s.Resource) {
	r.withLock(func() {
		result = r.limits
	})
	return
}

func New() *RateLimitController {
	return &RateLimitController{}
}

func (r *RateLimitController) DoWatch(ctx context.Context, cfg types.Config, kubeinfo *k8s.KubeInfo, _rlslog dlog.Logger) error {
	rlslog = _rlslog

	client, err := k8s.NewClient(kubeinfo)
	if err != nil {
		// this is non fatal (mostly just to facilitate local dev); don't `return err`
		rlslog.Errorln("not watching RateLimit resources:", errors.Wrap(err, "k8s.NewClient"))
		return nil
	}
	w := client.Watcher()

	count := 0

	matches, err := filepath.Glob(fmt.Sprintf("%s-*", cfg.RLSRuntimeDir))
	if err != nil {
		rlslog.Printf("warning: %v", err)
	} else {
		for _, m := range matches {
			parts := strings.Split(m, "-")
			end := parts[len(parts)-1]
			n, err := strconv.Atoi(end)
			if err == nil {
				count = max(count, n)
			}
		}
	}

	rlslog.Printf("initial count %d", count)

	fatal := make(chan error)
	w.Watch("ratelimits", func(w *k8s.Watcher) {
		config := &Config{Domains: make(map[string]*Domain)}
		var limits []k8s.Resource
		for _, r := range w.List("ratelimits") {
			var spec crd.RateLimitSpec
			err := mapstructure.Convert(r.Spec(), &spec)
			if err != nil {
				rlslog.Errorln(errors.Wrap(err, "malformed ratelimit resource spec"))
				continue
			}
			if cfg.AmbassadorSingleNamespace && r.Namespace() != cfg.AmbassadorNamespace {
				continue
			}
			if !spec.AmbassadorID.Matches(cfg.AmbassadorID) {
				continue
			}

			limits = append(limits, r)

			SetSource(&spec, r.QName())
			config.add(spec)
		}

		r.setLimits(limits)

		count += 1
		realout := fmt.Sprintf("%s-%d/%s", cfg.RLSRuntimeDir, count, cfg.RLSRuntimeSubdir)
		err = os.MkdirAll(realout, 0775)
		if err != nil {
			fatal <- err
			return
		}

		for _, domain := range config.Domains {
			bytes, err := yaml.Marshal(domain)
			if err != nil {
				fatal <- err
				return
			}
			fname := filepath.Join(realout, fmt.Sprintf("config.%s.yaml", domain.Name))
			err = ioutil.WriteFile(fname, bytes, 0644)
			if err != nil {
				fatal <- err
				return
			}
		}

		err = os.Remove(cfg.RLSRuntimeDir)
		if err != nil {
			rlslog.Println(err)
		}
		err = os.Symlink(filepath.Dir(realout), cfg.RLSRuntimeDir)
		if err != nil {
			fatal <- err
			return
		}
	})

	var reterr error
	go func() {
		select {
		case <-ctx.Done():
		case reterr = <-fatal:
			rlslog.Errorln(reterr)
		}
		w.Stop()
	}()
	w.Wait()
	return reterr
}

func SetSource(s *crd.RateLimitSpec, source string) {
	for i := range s.Limits {
		s.Limits[i].Source = source
	}
}

type Config struct {
	Domains map[string]*Domain
}
type Domain struct {
	Name        string    `yaml:"domain"`
	Descriptors NodeSlice `yaml:"descriptors,omitempty"`
}

type Node struct {
	Key         string
	Value       string    `yaml:"value,omitempty"`
	Rate        Rate      `yaml:"rate_limit,omitempty"`
	Descriptors NodeSlice `yaml:"descriptors,omitempty"`
}

type Rate struct {
	Rate uint64 `yaml:"requests_per_unit,omitempty"`
	Unit string `yaml:"unit,omitempty"`
}

func normalize(unit string) uint64 {
	switch unit {
	case "second":
		return 1
	case "minute":
		return 60
	case "hour":
		return 60 * 60
	case "day":
		return 24 * 60 * 60
	default:
		rlslog.Printf("warning: unrecognized unit: %s", unit)
		return 0
	}
}

func (r Rate) rps() uint64 {
	return r.Rate * normalize(r.Unit)
}

type NodeSlice []*Node

func (l *NodeSlice) child(key, value string) *Node {
	if value == "*" {
		value = ""
	}

	for _, nd := range *l {
		if nd.Key == key && nd.Value == value {
			return nd
		}
	}
	nd := &Node{Key: key, Value: value}
	*l = append(*l, nd)
	return nd
}

func (l *NodeSlice) add(pattern []map[string]string, limit crd.Limit) {
	for k, v := range pattern[0] {
		child := l.child(k, v)
		child.add(pattern[1:], limit)
	}
}

func (c *Config) add(spec crd.RateLimitSpec) {
	domain, ok := c.Domains[spec.Domain]
	if !ok {
		domain = &Domain{spec.Domain, nil}
		c.Domains[spec.Domain] = domain
	}
	for _, limit := range spec.Limits {
		domain.add(limit)
	}
}

func (d *Domain) add(limit crd.Limit) {
	if len(limit.Pattern) == 0 {
		rlslog.Printf("%s: empty pattern", limit.Source)
	} else {
		d.Descriptors.add(limit.Pattern, limit)
	}
}

func (n *Node) add(pattern []map[string]string, limit crd.Limit) {
	if len(pattern) == 0 {
		newRate := Rate{limit.Rate, limit.Unit}
		if n.Rate.Rate == 0 {
			n.Rate = newRate
		} else {
			rlslog.Printf("warning: %s: multiple limits for pattern %v, smaller limit enforced\n",
				limit.Source, limit.Pattern)
			if newRate.rps() < n.Rate.rps() {
				n.Rate = newRate
			}
		}
	} else {
		n.Descriptors.add(pattern, limit)
	}
}
