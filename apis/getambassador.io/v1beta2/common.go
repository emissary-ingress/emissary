package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"text/template"

	"github.com/pkg/errors"
)

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
	Name     string             `json:"name"`
	Value    string             `json:"value"`
	Template *template.Template `json:"-"`
}

func (hf *HeaderFieldTemplate) Validate() error {
	tmpl, err := template.New(hf.Name).Parse(hf.Value)
	if err != nil {
		return errors.Wrapf(err, "parsing template for header %q", hf.Name)
	}
	hf.Template = tmpl
	return nil
}

type ErrorResponse struct {
	Realm string `json:"realm"`

	ContentType string                `json:"contentType"`
	Headers     []HeaderFieldTemplate `json:"headers"`

	RawBodyTemplate string             `json:"bodyTemplate"`
	BodyTemplate    *template.Template `json:"-"`
}

func (er *ErrorResponse) Validate(qname string) error {
	// Handle deprecated .ContentType
	if er.ContentType != "" {
		er.Headers = append(er.Headers, HeaderFieldTemplate{
			Name:  "Content-Type",
			Value: er.ContentType,
		})
	}

	// Fill defaults
	if er.Realm == "" {
		er.Realm = qname
	}
	if len(er.Headers) == 0 {
		er.Headers = append(er.Headers, HeaderFieldTemplate{
			Name:  "Content-Type",
			Value: "application/json",
		})
	}
	if er.RawBodyTemplate == "" {
		er.RawBodyTemplate = `{{ . | json "" }}`
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
