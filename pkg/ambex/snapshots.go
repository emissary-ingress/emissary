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
// the Envoy configuration is protobuf messages rather than Go structs. So we need to
// do a certain amount of work to make sure we can marshal to JSON using jsonpb.Marshaler
// rather than just json.Marshal, so that the typed fields in the Envoy configuration are
// properly serialized.
//
// These "expanded" snapshots make the snapshots we log easier to read: basically,
// instead of just indexing by Golang types, make the JSON marshal with real names.
type v2ExpandedSnapshot struct {
	Endpoints ecp_v2_cache.Resources `json:"endpoints"`
	Clusters  ecp_v2_cache.Resources `json:"clusters"`
	Routes    ecp_v2_cache.Resources `json:"routes"`
	Listeners ecp_v2_cache.Resources `json:"listeners"`
	Runtimes  ecp_v2_cache.Resources `json:"runtimes"`
}

func NewV2ExpandedSnapshot(v2snap *ecp_v2_cache.Snapshot) v2ExpandedSnapshot {
	return v2ExpandedSnapshot{
		Endpoints: v2snap.Resources[ecp_cache_types.Endpoint],
		Clusters:  v2snap.Resources[ecp_cache_types.Cluster],
		Routes:    v2snap.Resources[ecp_cache_types.Route],
		Listeners: v2snap.Resources[ecp_cache_types.Listener],
		Runtimes:  v2snap.Resources[ecp_cache_types.Runtime],
	}
}

type v3ExpandedSnapshot struct {
	Endpoints ecp_v3_cache.Resources `json:"endpoints"`
	Clusters  ecp_v3_cache.Resources `json:"clusters"`
	Routes    ecp_v3_cache.Resources `json:"routes"`
	Listeners ecp_v3_cache.Resources `json:"listeners"`
	Runtimes  ecp_v3_cache.Resources `json:"runtimes"`
}

func NewV3ExpandedSnapshot(v3snap *ecp_v3_cache.Snapshot) v3ExpandedSnapshot {
	return v3ExpandedSnapshot{
		Endpoints: v3snap.Resources[ecp_cache_types.Endpoint],
		Clusters:  v3snap.Resources[ecp_cache_types.Cluster],
		Routes:    v3snap.Resources[ecp_cache_types.Route],
		Listeners: v3snap.Resources[ecp_cache_types.Listener],
		Runtimes:  v3snap.Resources[ecp_cache_types.Runtime],
	}
}

// A combinedSnapshot has both a V2 and V3 snapshot, for logging.
type combinedSnapshot struct {
	Version string             `json:"version"`
	V2      v2ExpandedSnapshot `json:"v2"`
	V3      v3ExpandedSnapshot `json:"v3"`
}

// csDump creates a combinedSnapshot from a V2 snapshot and a V3 snapshot, then
// dumps the combinedSnapshot to disk. Only numsnaps snapshots are kept: ambex-1.json
// is the newest, then ambex-2.json, etc., so ambex-$numsnaps.json is the oldest.
// Every time we write a new one, we rename all the older ones, ditching the oldest
// after we've written numsnaps snapshots.
func csDump(ctx context.Context, snapdirPath string, numsnaps int, generation int, v2snap *ecp_v2_cache.Snapshot, v3snap *ecp_v3_cache.Snapshot) {
	if numsnaps <= 0 {
		// Don't do snapshotting at all.
		return
	}

	// OK, they want snapshots. Make a proper version string...
	version := fmt.Sprintf("v%d", generation)

	// ...and a combinedSnapshot.
	cs := combinedSnapshot{
		Version: version,
		V2:      NewV2ExpandedSnapshot(v2snap),
		V3:      NewV3ExpandedSnapshot(v3snap),
	}

	// Next up, marshal as JSON and write to ambex-0.json. Note that we
	// didn't say anything about a -0 file; that's because it's about to
	// be renamed.
	bs, err := json.Marshal(cs)

	if err != nil {
		dlog.Errorf(ctx, "CSNAP: marshal failure: %s", err)
		return
	}

	csPath := path.Join(snapdirPath, "ambex-0.json")

	err = ioutil.WriteFile(csPath, bs, 0644)

	if err != nil {
		dlog.Errorf(ctx, "CSNAP: write failure: %s", err)
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
			dlog.Infof(ctx, "CSNAP: could not rename %s -> %s: %#v", fromPath, toPath, err)
		}
	}
}
