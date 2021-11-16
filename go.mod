module github.com/datawire/ambassador/v2

go 1.17

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
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/aokoli/goutils v1.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/cncf/udpa/go v0.0.0-20210322005330-6414d713912e
	github.com/datawire/dlib v1.2.5-0.20211116212847-0316f8d7af2b
	github.com/datawire/dtest v0.0.0-20210927191556-2cccf1a938b0
	github.com/datawire/go-mkopensource v0.0.0-20211110205306-9a3f29b7c373
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
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_model v0.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
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
	k8s.io/code-generator v0.20.2
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

require (
	cloud.google.com/go v0.54.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.1 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.5 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.0 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/armon/go-metrics v0.0.0-20180917152333-f0300d1749da // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/gobuffalo/flect v0.2.2 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/serf v0.8.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/moby/term v0.0.0-20200312100748-672ec06f55cd // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	golang.org/x/tools v0.0.0-20200904185747-39188db58858 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/apiserver v0.20.2 // indirect
	k8s.io/component-base v0.20.2 // indirect
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027 // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
	sigs.k8s.io/kustomize v2.0.3+incompatible // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2 // indirect
)

// We've got some bug-fixes that we need for conversion-gen.
replace k8s.io/code-generator v0.20.2 => github.com/datawire/code-generator v0.20.5-rc.0.0.20211127183116-16d402be64a9

// Because we (unfortunately) need to require k8s.io/kubernetes, which
// is (unfortunately) managed in a way that makes it hostile to being
// used as a library (see
// https://news.ycombinator.com/item?id=27827389) we need to provide
// replacements for a bunch of k8s.io modules that it refers to by
// bogus/broken v0.0.0 versions.
replace (
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
