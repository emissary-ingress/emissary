package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
)

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

type AmbassadorID []string

func (aid *AmbassadorID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*aid = nil
		return nil
	}

	var err error

	var list []string
	var single string

	if err = json.Unmarshal(data, &single); err == nil {
		*aid = AmbassadorID([]string{single})
		return nil
	}

	if err = json.Unmarshal(data, &list); err == nil {
		*aid = AmbassadorID(list)
		return nil
	}

	return err
}

func (aid AmbassadorID) Matches(envVar string) bool {
	if aid == nil {
		aid = []string{"default"}
	}
	for _, item := range aid {
		if item == envVar {
			return true
		}
	}
	return false
}

type HeaderFieldSelector struct {
	Name          string         `json:"name"`
	Value         string         `json:"value"`
	ValueRegex    *regexp.Regexp `json:"-"`
	RawValueRegex string         `json:"valueRegex"`
	Negate        bool           `json:"negate"`
}

func (selector *HeaderFieldSelector) Validate() error {
	if selector.Name == "" && selector.Value != "" {
		return errors.New("has .value but not a .name")
	}
	if selector.Name == "" && selector.RawValueRegex != "" {
		return errors.New("has .value but not a .name")
	}
	if selector.Value != "" && selector.RawValueRegex != "" {
		return errors.New("has both .value and .valueRegex")
	}
	if selector.RawValueRegex != "" {
		re, err := regexp.Compile(selector.RawValueRegex)
		if err != nil {
			return errors.Wrap(err, ".valueRegex")
		}
		selector.ValueRegex = re
	}
	return nil
}

func (selector HeaderFieldSelector) Matches(header http.Header) bool {
	if selector.Name == "" {
		return true
	}
	value := header.Get(selector.Name)
	var ret bool
	switch {
	case selector.ValueRegex != nil:
		ret = selector.ValueRegex.MatchString(value)
	case selector.Value != "":
		ret = value == selector.Value
	default:
		ret = value != ""
	}
	if selector.Negate {
		ret = !ret
	}
	return ret
}

type HeaderFieldTemplate struct {
	Name  string `json:"name"`
	Value string `json:"value"`

	tmpl *template.Template
}

type HeaderFieldAction int

func (hf *HeaderFieldTemplate) Execute(data interface{}) (*string, error) {
	tmpl, err := hf.tmpl.Clone()
	if err != nil {
		return nil, err
	}

	action := "set"
	tmpl.Funcs(template.FuncMap{
		"doNotSet": func() string { action = "none"; return "" },
	})

	w := new(strings.Builder)
	if err := tmpl.Execute(w, data); err != nil {
		return nil, err
	}

	if action == "none" {
		return nil, nil
	}
	value := w.String()
	return &value, nil
}

var hasKey = sprig.GenericFuncMap()["hasKey"]

func (hf *HeaderFieldTemplate) Validate() error {
	tmpl, err := template.
		New(hf.Name).
		Funcs(template.FuncMap{
			"hasKey":   hasKey,
			"doNotSet": (func() string)(nil),
		}).
		Parse(hf.Value)
	if err != nil {
		return errors.Wrapf(err, "parsing template for header %q", hf.Name)
	}
	hf.tmpl = tmpl
	return nil
}

type ErrorResponse struct {
	Headers []HeaderFieldTemplate `json:"headers"`

	RawBodyTemplate string             `json:"bodyTemplate"`
	BodyTemplate    *template.Template `json:"-"`
}

func (er *ErrorResponse) ValidateWithoutDefaults() error {
	return er.validate(false)
}

func (er *ErrorResponse) Validate() error {
	return er.validate(true)
}

func (er *ErrorResponse) validate(fillDefaults bool) error {
	// Fill defaults
	if fillDefaults {
		if len(er.Headers) == 0 {
			er.Headers = append(er.Headers, HeaderFieldTemplate{
				Name:  "Content-Type",
				Value: "application/json",
			})
		}
		if er.RawBodyTemplate == "" {
			er.RawBodyTemplate = `{{ . | json "" }}`
		}
	}

	// Parse+validate the header-field templates
	for i := range er.Headers {
		hf := &(er.Headers[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrap(err, "headers")
		}
	}
	// Parse+validate the bodyTemplate
	tmpl, err := template.
		New("bodyTemplate").
		Funcs(template.FuncMap{
			"json": func(prefix string, data interface{}) (string, error) {
				nonIdentedJSON, err := json.Marshal(data)
				if err != nil {
					return "", err
				}
				var indentedJSON bytes.Buffer
				if err := json.Indent(&indentedJSON, nonIdentedJSON, prefix, "\t"); err != nil {
					return "", err
				}
				return indentedJSON.String(), nil
			},
		}).
		Parse(er.RawBodyTemplate)
	if err != nil {
		return errors.Wrap(err, "parsing template for bodyTemplate")
	}
	er.BodyTemplate = tmpl

	return nil
}
