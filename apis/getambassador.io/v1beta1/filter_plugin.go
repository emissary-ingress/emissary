package v1

import (
	"net/http"
	"plugin"
	"strings"

	"github.com/pkg/errors"
)

type FilterPlugin struct {
	Name    string       `json:"name"`
	Handler http.Handler `json:"-"`
}

func (m *FilterPlugin) Validate() error {
	if strings.Contains(m.Name, "/") {
		return errors.Errorf("invalid .spec.Plugin.name: contains a /: %q", m.Name)
	}
	pluginHandle, err := plugin.Open("/etc/ambassador-plugins/" + m.Name + ".so")
	if err != nil {
		return errors.Wrap(err, "could not open plugin file")
	}
	pluginInterface, err := pluginHandle.Lookup("PluginMain")
	if err != nil {
		return errors.Wrap(err, "invalid plugin file")
	}
	pluginMain, ok := pluginInterface.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return errors.Errorf("invalid plugin file: PluginMain has the wrong type")
	}
	m.Handler = http.HandlerFunc(pluginMain)

	return nil
}
