module github.com/datawire/ambassador/v2

go 1.13

// If you're editing this file, there's a few things you should know:
//
//  1. Avoid the `replace` command as much as possible.  Go only pays
//     attention to the `replace` command when it appears in the main
//     module, which means that if the `replace` command is required
//     for the compile to work, then anything using ambassador.git as
//     a library needs to duplicate that `replace` in their go.mod.
//     We don't want to burden our users with that if we can avoid it
//     (since we encourage them to use the gRPC Go libraries when
//     implementing plugin services), and we don't want to deal with
//     that ourselves in apro.git.
//
//     The biggest reason we wouldn't be able to avoid it is if we
//     need to pull in a library that has a `replace` in its
//     go.mod--just as us adding a `replace` to our go.mod would
//     require our dependents to do the same, our dependencies adding
//     a `replace` requires us to do the same.  And even then, if
//     we're careful we might be able to avoid it.
//
//  2. If you do add a `replace` command to this file, always include
//     a version number to the left of the "=>" (even if you're
//     copying the command from a dependnecy and the dependency's
//     go.mod doesn't have a version on the left side).  This way we
//     don't accidentally keep pinning to an older version after our
//     dependency's `replace` goes away.  Sometimes it can be tricky
//     to figure out what version to put on the left side; often the
//     easiest way to figure it out is to get it working without a
//     version, run `go mod vendor`, then grep for "=>" in
//     `./vendor/modules.txt`.  If you don't see a "=>" line for that
//     replacement in modules.txt, then that's an indicator that we
//     don't really need that `replace`, maybe replacing it with a
//     `require` (or not; see the notes on go-autorest below).
//
//  3. If you do add a `replace` command to this file, you must also
//     add it to the go.mod in apro.git (see above for explanation).
//
//  4. Use `make go-mod-tidy` instead of `go mod tidy`.  Normal `go
//     mod tidy` will try to remove `github.com/cncf/udpa`--don't let
//     it, that would break `make generate`; the github.com/cncf/udpa
//     version needs to be kept in-sync with the
//     github.com/cncf/udpa/go version (`make go-mod-tidy` will do
//     this).

require (
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/aokoli/goutils v1.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/cncf/udpa/go v0.0.0-20210322005330-6414d713912e
	github.com/datawire/dlib v1.2.4-0.20210629021142-e221f3b9c3b8
	github.com/datawire/dtest v0.0.0-20210927191556-2cccf1a938b0
	github.com/datawire/go-mkopensource v0.0.0-20211026180000-06f43d9d3384
	github.com/envoyproxy/protoc-gen-validate v0.3.0-java.0.20200609174644-bd816e4522c1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/getkin/kin-openapi v0.66.0
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.2
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/consul/api v1.3.0
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/miekg/dns v1.1.35 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	google.golang.org/genproto v0.0.0-20201110150050-8816d57aaa9a
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog/v2 v2.10.0
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/kubectl v0.20.2
	k8s.io/kubernetes v1.20.2
	k8s.io/metrics v0.20.2
	sigs.k8s.io/controller-runtime v0.8.0
	sigs.k8s.io/controller-tools v0.5.0
	sigs.k8s.io/gateway-api v0.2.0
	sigs.k8s.io/yaml v1.2.0
)

// Because we (unfortunately) need to require k8s.io/kubernetes, which
// is (unfortunately) managed in a way that makes it hostile to being
// used as a library (see
// https://news.ycombinator.com/item?id=27827389) we need to provide
// replacements for a bunch of k8s.io modules that it refers to by
// bogus/broken v0.0.0 versions.
replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	k8s.io/api v0.0.0 => k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.20.2
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.20.2
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.20.2
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.20.2
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.20.2
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.20.2
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.20.2
	k8s.io/component-helpers v0.0.0 => k8s.io/component-helpers v0.20.2
	k8s.io/controller-manager v0.0.0 => k8s.io/controller-manager v0.20.2
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.20.2
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.20.2
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.20.2
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.20.2
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.20.2
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.20.2
	k8s.io/kubectl v0.0.0 => k8s.io/kubectl v0.20.2
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.20.2
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.20.2
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.20.2
	k8s.io/mount-utils v0.0.0 => k8s.io/mount-utils v0.20.2
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.20.2
)

// The issue doesn't trigger with the current versions of our
// dependencies, but if you ever get an error message like:
//
//     /path/to/whatever.go.go:7:2: ambiguous import: found package github.com/Azure/go-autorest/autorest in multiple modules:
//            github.com/Azure/go-autorest v10.8.1+incompatible (/home/lukeshu/go/pkg/mod/github.com/!azure/go-autorest@v10.8.1+incompatible/autorest)
//            github.com/Azure/go-autorest/autorest v0.9.0 (/home/lukeshu/go/pkg/mod/github.com/!azure/go-autorest/autorest@v0.9.0)
//
// then uncomment the `replace` line below, adjusting the version
// number to the left of the "=>" to match the error message.
//
// The go-autorest one is a little funny; we don't actually use that
// module; we use the nested "github.com/Azure/go-autorest/autorest"
// module, which split off from it some time between v11 and v13; and
// we just need to tell it to consider v13 instead of v11 so that it
// knows to use the nested module (instead of complaining about
// "ambiguous import: found package in multiple modules").  We could
// do this with
//
//     require github.com/Azure/go-autorest v13.3.2+incompatible
//
// but `go mod tidy` would remove it and break the build.  We could
// inhibit `go mod tidy` from removing it by importing a package from
// it in `./pkg/ignore/pin.go`, but there are actually no packages in
// it to import; it's entirely nested modules.
//
//replace github.com/Azure/go-autorest v10.8.1+incompatible => github.com/Azure/go-autorest v13.3.2+incompatible
