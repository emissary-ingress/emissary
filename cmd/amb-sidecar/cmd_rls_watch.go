package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/lib/mapstructure"
	"github.com/datawire/teleproxy/pkg/k8s"
)

var rlslog *logrus.Logger

func rlsdie(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			rlslog.Fatalf("%v: %v\n", err, args)
		} else {
			rlslog.Fatalln(err)
		}
	}
}

var watch = &cobra.Command{
	Use:   "rls-watch",
	Short: "Watch RateLimit CRD files",
	RunE:  doWatch,
}

func init() {
	argparser.AddCommand(watch)
	watch.Flags().StringVarP(&output, "output", "o", "", "output directory")
	watch.MarkFlagRequired("output")
}

var output string

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

type ControllerConfig struct {
	AmbassadorID              string
	AmbassadorNamespace       string
	AmbassadorSingleNamespace bool
}

func doWatch(cmd *cobra.Command, args []string) error {
	rlslog = logrus.New()

	w := k8s.NewClient(nil).Watcher()

	controllerConfig := ControllerConfig{
		AmbassadorID:              os.Getenv("AMBASSADOR_ID"),
		AmbassadorNamespace:       os.Getenv("AMBASSADOR_NAMESPACE"),
		AmbassadorSingleNamespace: os.Getenv("AMBASSADOR_SINGLE_NAMESPACE") != "",
	}
	if controllerConfig.AmbassadorID == "" {
		controllerConfig.AmbassadorID = "default"
	}
	if controllerConfig.AmbassadorNamespace == "" {
		controllerConfig.AmbassadorNamespace = "default"
	}

	count := 0

	matches, err := filepath.Glob(fmt.Sprintf("%s-*", output))
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

	w.Watch("ratelimits", func(w *k8s.Watcher) {
		config := &Config{Domains: make(map[string]*Domain)}
		for _, r := range w.List("ratelimits") {
			var spec crd.RateLimitSpec
			err := mapstructure.Convert(r.Spec(), &spec)
			if err != nil {
				rlslog.Errorln(errors.Wrap(err, "malformed ratelimit resource spec"))
				continue
			}
			if controllerConfig.AmbassadorSingleNamespace && r.Namespace() != controllerConfig.AmbassadorNamespace {
				continue
			}
			if !spec.AmbassadorID.Matches(controllerConfig.AmbassadorID) {
				continue
			}

			SetSource(&spec, r.QName())
			config.add(spec)
		}

		count += 1
		realout := fmt.Sprintf("%s-%d/config", output, count)
		err = os.MkdirAll(realout, 0775)
		rlsdie(err)

		for _, domain := range config.Domains {
			bytes, err := yaml.Marshal(domain)
			rlsdie(err)
			fname := filepath.Join(realout, fmt.Sprintf("config.%s.yaml", domain.Name))
			err = ioutil.WriteFile(fname, bytes, 0644)
			rlsdie(err)
		}

		err = os.Remove(output)
		if err != nil {
			rlslog.Println(err)
		}
		err = os.Symlink(filepath.Dir(realout), output)
		rlsdie(err)
	})
	w.Wait()
	return nil
}

type Errors struct {
	errors map[string][]string
}

func (e *Errors) add(key string, err error) {
	e.errors[key] = append(e.errors[key], err.Error())
}

func (e *Errors) empty() bool {
	return len(e.errors) != 0
}

func SetSource(s *crd.RateLimitSpec, source string) {
	for i := range s.Limits {
		s.Limits[i].Source = source
	}
}

func validateSpec(s crd.RateLimitSpec, errs *Errors) {
	for _, l := range s.Limits {
		validateLimit(l, errs)
	}
}

func validateLimit(l crd.Limit, errs *Errors) {
	for _, entry := range l.Pattern {
		if len(entry) == 0 {
			errs.add(l.Source, fmt.Errorf("empty entry: %v", l))
		}
	}
	switch l.Unit {
	case "second":
	case "minute":
	case "hour":
	case "day":
	default:
		errs.add(l.Source, fmt.Errorf("unrecognized unit: %v", l))
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
