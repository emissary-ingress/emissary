// +build linux,amd64,cgo

package aes_plugin_runner

import (
	"net/http"
	"plugin"

	"github.com/pkg/errors"

	_ "github.com/datawire/apro/cmd/amb-sidecar/runner"
)

func _mainNative(socketName, pluginFilepath string) error {
	pluginHandle, err := plugin.Open(pluginFilepath)
	if err != nil {
		return errors.Wrap(err, "load plugin file")
	}

	pluginInterface, err := pluginHandle.Lookup("PluginMain")
	if err != nil {
		return errors.Wrap(err, "invalid plugin file")
	}

	pluginMain, ok := pluginInterface.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return errors.New("invalid plugin file: PluginMain has the wrong type signature")
	}

	return http.ListenAndServe(socketName, http.HandlerFunc(pluginMain))
}

func init() {
	mainNative = _mainNative
}
