package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	ms "github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/datawire/teleproxy/pkg/k8s"
)

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			fmt.Printf("%v: %v\n", err, args)
		} else {
			fmt.Println(err)
		}
		panic(err)
	}
}

var watch = &cobra.Command{
	Use:   "rls-watch",
	Short: "Watch RateLimit CRD files",
	Run:   doWatch,
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

func doWatch(cmd *cobra.Command, args []string) {
	w := k8s.NewClient(nil).Watcher()
	count := 0

	matches, err := filepath.Glob(fmt.Sprintf("%s-*", output))
	if err != nil {
		log.Printf("warning: %v", err)
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

	log.Printf("initial count %d", count)

	w.Watch("ratelimits", func(w *k8s.Watcher) {
		config := &Config{Domains: make(map[string]*Domain)}

		for _, r := range w.List("ratelimits") {
			spec, err := decode(r.QName(), r.Spec())
			if err != nil {
				log.Printf("%s: %v", r.QName(), err)
			} else {
				config.add(spec)
			}
		}

		count += 1
		realout := fmt.Sprintf("%s-%d/config", output, count)
		err = os.MkdirAll(realout, 0775)
		die(err)

		for _, domain := range config.Domains {
			bytes, err := yaml.Marshal(domain)
			die(err)
			fname := filepath.Join(realout, fmt.Sprintf("config.%s.yaml", domain.Name))
			err = ioutil.WriteFile(fname, bytes, 0644)
			die(err)
		}

		err = os.Remove(output)
		if err != nil {
			log.Println(err)
		}
		err = os.Symlink(filepath.Dir(realout), output)
		die(err)
	})
	w.Wait()
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

type Spec struct {
	Domain string
	Limits []Limit
	source string
}

func (s *Spec) SetSource(source string) {
	s.source = source
	for i, _ := range s.Limits {
		s.Limits[i].source = source
	}
}

type Limit struct {
	Pattern []map[string]string
	Rate    uint64
	Unit    string
	source  string
}

func (s Spec) validate(errs *Errors) {
	for _, l := range s.Limits {
		l.validate(errs)
	}
}

func (l Limit) validate(errs *Errors) {
	for _, entry := range l.Pattern {
		if len(entry) == 0 {
			errs.add(l.source, fmt.Errorf("empty entry: %v", l))
		}
	}
	switch l.Unit {
	case "second":
	case "minute":
	case "hour":
	case "day":
	default:
		errs.add(l.source, fmt.Errorf("unrecognized unit: %v", l))
	}
}

func decode(source string, input interface{}) (Spec, error) {
	var result Spec
	d, err := ms.NewDecoder(&ms.DecoderConfig{
		ErrorUnused: true,
		Result:      &result,
	})
	if err != nil {
		return result, err
	}
	err = d.Decode(input)
	if err == nil {
		result.SetSource(source)
	}
	return result, err
}

func load(path string) []k8s.Resource {
	var result []k8s.Resource
	file, err := os.Open(path)
	die(err)
	d := yaml.NewDecoder(file)
	for {
		var uns map[interface{}]interface{}
		err = d.Decode(&uns)
		if err == io.EOF {
			break
		} else {
			die(err)
		}
		md := uns["metadata"].(map[interface{}]interface{})
		_, ok := md["namespace"]
		if !ok {
			md["namespace"] = "default"
		}
		result = append(result, k8s.NewResourceFromYaml(uns))
	}
	return result
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
		log.Printf("warning: unrecognized unit: %s", unit)
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

func (l *NodeSlice) add(pattern []map[string]string, limit Limit) {
	for k, v := range pattern[0] {
		child := l.child(k, v)
		child.add(pattern[1:], limit)
	}
}

func (c *Config) add(spec Spec) {
	domain, ok := c.Domains[spec.Domain]
	if !ok {
		domain = &Domain{spec.Domain, nil}
		c.Domains[spec.Domain] = domain
	}
	for _, limit := range spec.Limits {
		domain.add(limit)
	}
}

func (d *Domain) add(limit Limit) {
	if len(limit.Pattern) == 0 {
		log.Printf("%s: empty pattern", limit.source)
	} else {
		d.Descriptors.add(limit.Pattern, limit)
	}
}

func (n *Node) add(pattern []map[string]string, limit Limit) {
	if len(pattern) == 0 {
		newRate := Rate{limit.Rate, limit.Unit}
		if n.Rate.Rate == 0 {
			n.Rate = newRate
		} else {
			log.Printf("warning: %s: multiple limits for pattern %v, smaller limit enforced\n",
				limit.source, limit.Pattern)
			if newRate.rps() < n.Rate.rps() {
				n.Rate = newRate
			}
		}
	} else {
		n.Descriptors.add(pattern, limit)
	}
}
