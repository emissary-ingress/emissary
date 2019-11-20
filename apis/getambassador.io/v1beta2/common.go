package v1

import (
	"encoding/json"
	"net/http"
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
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (selector HeaderFieldSelector) Validate() error {
	if selector.Name == "" && selector.Value != "" {
		return errors.New("has .value but not a .name")
	}
	return nil
}

func (selector HeaderFieldSelector) Matches(header http.Header) bool {
	if selector.Name == "" {
		return true
	}
	value := header.Get(selector.Name)
	if selector.Value == "" {
		return value != ""
	}
	return value == selector.Value
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
