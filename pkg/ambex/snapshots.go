package ambex

import (
	// standard library

	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	// third-party libraries

	// envoy control plane
	ecp_cache_types "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	ecp_v2_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	ecp_v3_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	// Envoy API v2
	// Be sure to import the package of any types that're referenced with "@type" in our
	// generated Envoy config, even if that package is otherwise not used by ambex.

	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/auth"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/accesslog/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/buffer/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/ext_authz/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/gzip/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/lua/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/rate_limit/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/rbac/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/router/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/tcp_proxy/v2"

	// Envoy API v3
	// Be sure to import the package of any types that're referenced with "@type" in our
	// generated Envoy config, even if that package is otherwise not used by ambex.
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/accesslog/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/access_loggers/file/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/access_loggers/grpc/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/compression/gzip/compressor/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/buffer/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/compressor/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/ext_authz/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/gzip/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/lua/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/ratelimit/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/response_map/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/router/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/tcp_proxy/v3"

	// first-party libraries

	"github.com/datawire/dlib/dlog"
)

// Snapshot handling.
//
// The Envoy configurations that we work with are called "snapshots", since they're
// an internally-consistent representation of a single point in time. As a debugging
// aid, we log the snapshots to disk before sending them on to Envoy. The logs go into
// /ambassador/snapshots/ambex-<version>.json.
//
// Actually making these snapshots is harder than you might think, because the core of
// the Envoy configuration is protobuf messages rather than Go structs, and json.Marshal
// doesn't properly handle Envoy's "typed_config" fields, where a protobuf message is
// included with an explicit "@type" attribute to tell what kind of message it is so that
// Envoy can unmarshal it.
//
// Hence the marshaledSnapshot type, which handles the extra work needed to make it
// possible to use protojson.Marshal rather than json.Marshal. The way this works is that
// we take a bunch of V2 elements and a bunch of V3 elements, and we marshal them down to
// json.RawMessages using protojson. Then we can serialize the whole marshaledSnapshot
// easily with the usual json.Marshal function.
//
// Everything in the marshaledSnapshot is versioned, too, since that's how the ADS
// protocol works. If you manage to find that the versions don't match up, that's likely
// not a good thing.

type marshaledSnapshot struct {
	errors  []error                      `json:"-"`
	Version string                       `json:"version"`
	V2      map[string]marshaledElements `json:"v2"`
	V3      map[string]marshaledElements `json:"v3"`
}

type marshaledElements struct {
	Version  string            `json:"version"`
	Elements []json.RawMessage `json:"elements"`
}

// NewMarshaledSnapshot takes a v2snapshot and a v3snapshot and returns a
// marshaledSnapshot, ready to be serialized.
func NewMarshaledSnapshot(version string, v2snap *ecp_v2_cache.Snapshot, v3snap *ecp_v3_cache.Snapshot) marshaledSnapshot {
	ms := marshaledSnapshot{
		errors:  make([]error, 0),
		Version: version,
		V2:      make(map[string]marshaledElements),
		V3:      make(map[string]marshaledElements),
	}

	ms.marshalV2Resources("Endpoints", v2snap.Resources[ecp_cache_types.Endpoint])
	ms.marshalV2Resources("Clusters", v2snap.Resources[ecp_cache_types.Cluster])
	ms.marshalV2Resources("Routes", v2snap.Resources[ecp_cache_types.Route])
	ms.marshalV2Resources("Listeners", v2snap.Resources[ecp_cache_types.Listener])
	ms.marshalV2Resources("Runtimes", v2snap.Resources[ecp_cache_types.Runtime])

	ms.marshalV3Resources("Endpoints", v3snap.Resources[ecp_cache_types.Endpoint])
	ms.marshalV3Resources("Clusters", v3snap.Resources[ecp_cache_types.Cluster])
	ms.marshalV3Resources("Routes", v3snap.Resources[ecp_cache_types.Route])
	ms.marshalV3Resources("Listeners", v3snap.Resources[ecp_cache_types.Listener])
	ms.marshalV3Resources("Runtimes", v3snap.Resources[ecp_cache_types.Runtime])

	return ms
}

// marshalV2Resources is just a helper: it marshals a bunch of V2 resources
// and updates the marshaledSnapshot correctly. marshaledV2Elements does the heavy
// lifting.
func (ms *marshaledSnapshot) marshalV2Resources(name string, resources ecp_v2_cache.Resources) {
	mel, err := marshaledV2Elements(resources)

	if err != nil {
		ms.errors = append(ms.errors, err)
		return
	}

	ms.V2[name] = *mel
}

// marshalV3Resources is just a helper: it marshals a bunch of V3 resources
// and updates the marshaledSnapshot correctly. marshaledV3Elements does the heavy
// lifting.
func (ms *marshaledSnapshot) marshalV3Resources(name string, resources ecp_v3_cache.Resources) {
	mel, err := marshaledV3Elements(resources)

	if err != nil {
		ms.errors = append(ms.errors, err)
		return
	}

	ms.V3[name] = *mel
}

// marshaledV2Elements builds a marshaledElements data structure from a bunch of V2
// resources. Note that the version comes from the resource set, not from the caller.
func marshaledV2Elements(resources ecp_v2_cache.Resources) (*marshaledElements, error) {
	mel := marshaledElements{
		Version:  resources.Version,
		Elements: make([]json.RawMessage, 0, len(resources.Items)),
	}

	for _, e := range resources.Items {
		j, err := protojson.Marshal(e.Resource.(proto.Message))

		if err != nil {
			return nil, err
		}
		mel.Elements = append(mel.Elements, json.RawMessage(j))
	}

	return &mel, nil
}

// marshaledV3Elements builds a marshaledElements data structure from a bunch of V3
// resources. Note that the version comes from the resource set, not from the caller.
func marshaledV3Elements(resources ecp_v3_cache.Resources) (*marshaledElements, error) {
	mel := marshaledElements{
		Version:  resources.Version,
		Elements: make([]json.RawMessage, 0, len(resources.Items)),
	}

	for _, e := range resources.Items {
		j, err := protojson.Marshal(e.Resource.(proto.Message))

		if err != nil {
			return nil, err
		}
		mel.Elements = append(mel.Elements, json.RawMessage(j))
	}

	return &mel, nil
}

// dumpSnapshot creates a marshaledSnapshot from a V2 snapshot and a V3 snapshot, then
// dumps the marshaledSnapshot to disk. Only numsnaps snapshots are kept: ambex-1.json
// is the newest, then ambex-2.json, etc., so ambex-$numsnaps.json is the oldest.
// Every time we write a new one, we rename all the older ones, ditching the oldest
// after we've written numsnaps snapshots.
func dumpSnapshot(ctx context.Context, snapdirPath string, numsnaps int, generation int, v2snap *ecp_v2_cache.Snapshot, v3snap *ecp_v3_cache.Snapshot) {
	if numsnaps <= 0 {
		// Don't do snapshotting at all.
		return
	}

	// OK, they want snapshots. Make a proper version string...
	version := fmt.Sprintf("v%d", generation)

	// ...and a marshaledSnapshot.
	ms := NewMarshaledSnapshot(version, v2snap, v3snap)

	// Next up, marshal as JSON and write to ambex-0.json. Note that we
	// didn't say anything about a -0 file; that's because it's about to
	// be renamed.
	bs, err := json.Marshal(ms)

	if err != nil {
		dlog.Errorf(ctx, "csDump: marshal failure: %s", err)
		return
	}

	snapPath := path.Join(snapdirPath, "ambex-0.json")

	err = ioutil.WriteFile(snapPath, bs, 0644)

	if err != nil {
		dlog.Errorf(ctx, "csDump: write failure: %s", err)
	} else {
		dlog.Infof(ctx, "Saved snapshot %s", version)
	}

	// Rotate everything one file down. This includes renaming the just-written
	// ambex-0 to ambex-1.
	for i := numsnaps; i > 0; i-- {
		previous := i - 1

		fromPath := path.Join(snapdirPath, fmt.Sprintf("ambex-%d.json", previous))
		toPath := path.Join(snapdirPath, fmt.Sprintf("ambex-%d.json", i))

		err := os.Rename(fromPath, toPath)

		if (err != nil) && !os.IsNotExist(err) {
			dlog.Infof(ctx, "csDump: could not rename %s -> %s: %#v", fromPath, toPath, err)
		}
	}
}
