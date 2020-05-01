// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/gogo/protobuf/jsonpb"

	discovery "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/log"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/resource/v2"
)

// HTTPGateway is a custom implementation of [gRPC gateway](https://github.com/grpc-ecosystem/grpc-gateway)
// specialized to Envoy xDS API.
type HTTPGateway struct {
	// Log is an optional log for errors in response write
	Log log.Logger

	// Server is the underlying gRPC server
	Server Server
}

func (h *HTTPGateway) ServeHTTP(resp http.ResponseWriter, req *http.Request) error {
	p := path.Clean(req.URL.Path)

	typeURL := ""
	switch p {
	case resource.FetchEndpoints:
		typeURL = resource.EndpointType
	case resource.FetchClusters:
		typeURL = resource.ClusterType
	case resource.FetchListeners:
		typeURL = resource.ListenerType
	case resource.FetchRoutes:
		typeURL = resource.RouteType
	case resource.FetchSecrets:
		typeURL = resource.SecretType
	case resource.FetchRuntimes:
		typeURL = resource.RuntimeType
	default:
		http.Error(resp, "no endpoint", http.StatusNotFound)
		return fmt.Errorf("no endpoint")
	}

	if req.Body == nil {
		http.Error(resp, "empty body", http.StatusBadRequest)
		return nil
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "cannot read body", http.StatusBadRequest)
		return nil
	}

	// parse as JSON
	out := &discovery.DiscoveryRequest{}
	err = jsonpb.UnmarshalString(string(body), out)
	if err != nil {
		http.Error(resp, "cannot parse JSON body: "+err.Error(), http.StatusBadRequest)
		return nil
	}
	out.TypeUrl = typeURL

	// fetch results
	res, err := h.Server.Fetch(req.Context(), out)
	if err != nil {
		// SkipFetchErrors will return a 304 which will signify to the envoy client that
		// it is already at the latest version; all other errors will 500 with a message.
		if _, ok := err.(*types.SkipFetchError); ok {
			resp.WriteHeader(http.StatusNotModified)
		} else {
			http.Error(resp, "fetch error: "+err.Error(), http.StatusInternalServerError)
		}
		return nil
	}

	buf := &bytes.Buffer{}
	if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, res); err != nil {
		http.Error(resp, "marshal error: "+err.Error(), http.StatusInternalServerError)
	}

	if _, err = resp.Write(buf.Bytes()); err != nil && h.Log != nil {
		h.Log.Errorf("gateway error: %v", err)
	}

	return nil
}
