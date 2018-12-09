package controller

import (
	"sync/atomic"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/teleproxy/pkg/k8s"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"

	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	ms "github.com/mitchellh/mapstructure"
)

// Controller is a custom Kubernetes controller that monitor k8s
// cluster and load all the rules used by this app for authorizing
// api calls.
type Controller struct {
	Logger *logrus.Logger
	Config *config.Config
	Rules  atomic.Value
}

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch() {
	c.Rules.Store(make([]Rule, 0))
	w := k8s.NewClient(nil).Watcher()

	w.Watch("policies", func(w *k8s.Watcher) {
		rules := make([]Rule, 0)
		c.Logger.Info("loading rules:")
		for _, p := range w.List("policies") {
			spec, err := decode(p.QName(), p.Spec())
			if err != nil {
				c.Logger.Debugf("malformed object, bad spec: %v", spec)
				continue
			}
			for _, r := range spec.Rules {
				rule := Rule{}
				err := ms.Decode(r, &rule)
				if err != nil {
					c.Logger.Error(err)
				} else {
					c.Logger.Infof("host=%s, path=%s, public=%v, scopes=%s",
						rule.Host, rule.Path, rule.Public, rule.Scopes)
					rules = append(rules, rule)
				}
			}
		}

		c.Rules.Store(rules)
	})

	w.Wait()
}

// Spec ..
type Spec struct {
	Rules []Rule
}

// Rule ..
type Rule struct {
	Host   string
	Path   string
	Public bool
	Scopes string
}

// Match ..
func (r Rule) Match(host, path string) bool {
	return match(r.Host, host) && match(r.Path, path)
}

func match(pattern, input string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		return false
	}

	return g.Match(input)
}

func decode(source string, input interface{}) (*Spec, error) {
	var s Spec
	d, err := ms.NewDecoder(&ms.DecoderConfig{
		ErrorUnused: true,
		Result:      &s,
	})
	if err != nil {
		return nil, err
	}

	err = d.Decode(input)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
