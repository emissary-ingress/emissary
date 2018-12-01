package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	ms "github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/datawire/teleproxy/pkg/k8s"
)

var rls = &cobra.Command{
	Use:   "rls [subcommand]",
	Short: "Work with ratelimit crds",
}

func init() {
	apictl.AddCommand(rls)
}

var validate = &cobra.Command{
	Use:   "validate [files]",
	Short: "Validate ratelimit crd files.",
	Run:   doValidate,
}

func init() {
	rls.AddCommand(validate)
	validate.Flags().BoolVar(&offline, "offline", false, "perform offline validation only")
	validate.Flags().StringVarP(&output, "output", "o", "", "output directory")
}

var offline bool
var output string

func doValidate(cmd *cobra.Command, args []string) {
	var err error
	var local_resources []k8s.Resource
	var remote_resources []k8s.Resource

	for _, arg := range args {
		local_resources = append(local_resources, load(arg)...)
	}

	fmt.Printf("Found %d local resources.\n", len(local_resources))

	if !offline {
		c := k8s.NewClient(nil)
		remote_resources, err = c.List("ratelimits")
		die(err)

		fmt.Printf("Found %d remote resources in cluster.\n", len(remote_resources))
	}

	fmt.Printf("Validating...\n")

	resources := make(map[string]k8s.Resource)

	for _, r := range remote_resources {
		resources[r.QName()] = r
	}

	for _, r := range local_resources {
		resources[r.QName()] = r
	}

	config := &Config{Domains: make(map[string]*Domain)}

	errs := &Errors{make(map[string][]string)}

	for _, r := range resources {
		spec, err := decode(r.QName(), r.Spec())
		if err != nil {
			fmt.Println(err)
		} else {
			spec.validate(errs)
			config.add(spec)
		}
	}

	code := 0
	for k, v := range errs.errors {
		fmt.Printf("%s: %s\n", k, strings.Join(v, "\n  "+strings.Repeat(" ", len(k))))
		code = 1
	}

	if code > 0 {
		os.Exit(code)
	}

	if output != "" {
		for _, domain := range config.Domains {
			bytes, err := yaml.Marshal(domain)
			die(err)
			fname := filepath.Join(output, fmt.Sprintf("config.%s.yaml", domain.Name))
			err = ioutil.WriteFile(fname, bytes, 0644)
			die(err)
		}
	}
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

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			panic(fmt.Errorf("%v: %v", err, args))
		} else {
			panic(err)
		}
	}
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
