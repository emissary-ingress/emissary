package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/datawire/teleproxy/pkg/k8s"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/mapstructure"
)

var rls = &cobra.Command{
	Use:   "rls [subcommand]",
	Short: "Work with Rate Limits",
}

func init() {
	apictl.AddCommand(rls)
}

var validate = &cobra.Command{
	Use:   "Validate [files]",
	Short: "Validate RateLimit CRD files",
	Run:   doValidate,
}

func init() {
	rls.AddCommand(validate)
	validate.Flags().BoolVar(&offline, "offline", false, "perform offline validation only")
}

var offline bool

func doValidate(cmd *cobra.Command, args []string) {
	var err error
	var local_resources []k8s.Resource
	var remote_resources []k8s.Resource

	if err := licenseClaims.RequireFeature(licensekeys.FeatureRateLimit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

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
		var spec crd.RateLimitSpec
		err := mapstructure.Convert(r.Spec(), &spec)
		if err != nil {
			log.Printf("%s: %v", r.QName(), err)
			continue
		}
		SetSource(&spec, r.QName())
		validateSpec(spec, errs)
		config.add(spec)
	}

	count := 0
	for k, v := range errs.errors {
		fmt.Printf("%s: %s\n", k, strings.Join(v, "\n  "+strings.Repeat(" ", len(k))))
		count += 1
	}

	fmt.Printf("Found %d errors.\n", count)

	if count > 0 {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

type Errors struct {
	errors map[string][]string
}

func (e *Errors) add(key string, err error) {
	e.errors[key] = append(e.errors[key], err.Error())
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
		log.Printf("%s: empty pattern", limit.Source)
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
			log.Printf("warning: %s: multiple limits for pattern %v, smaller limit enforced\n",
				limit.Source, limit.Pattern)
			if newRate.rps() < n.Rate.rps() {
				n.Rate = newRate
			}
		}
	} else {
		n.Descriptors.add(pattern, limit)
	}
}
