module github.com/datawire/ambassador

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
//  2. If you do add a `replace` command to this file, you must also
//     add it to the go.mod in apro.git (see above for explanation).
//
//  3. We used to use https://github.com/datawire/libk8s to manage the
//     Kubernetes library versions (since the Kubernetes folks de it
//     such a nightmare), but recent versions of Kubernetes 1.x.y now
//     have useful "v0.x.y" git tags that Go understands, so it's
//     actually quite reasonable now.  If you find yourself having to
//     do any hacks with k8s.io library versions (like doing a
//     `replace` for a dozen different k8s.io/ packages), stop, and
//     ask someone for advice.

require (
	git.lukeshu.com/go/libsystemd v0.5.3
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/datawire/pf v0.0.0-20180510150411-31a823f9495a
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ecodia/golang-awaitility v0.0.0-20180710094957-fb55e59708c7
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.1.1
	github.com/gookit/color v1.2.3
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/consul/api v1.3.0
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/lyft/protoc-gen-star v0.4.4
	github.com/mholt/archiver/v3 v3.3.0
	github.com/miekg/dns v1.1.22
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mitchellh/protoc-gen-go-json v0.0.0-20190813154521-ece073100ced
	github.com/operator-framework/operator-sdk v0.16.1-0.20200331173026-a00c1b716663
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sys v0.0.0-20200122134326-e047566fdf82
	google.golang.org/grpc v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm/v3 v3.1.2
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/cli-runtime v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/helm v2.16.5+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/yaml v1.1.0
)

// From the go.mod files in some of our dependencies...
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // from helm.sh/helm/v3@v3.1.0 and github.com/operator-framework/operator-sdk@v0.16.1-0.20200331173026-a00c1b716663
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d // from helm.sh/helm/v3@v3.1.0
	k8s.io/client-go => k8s.io/client-go v0.17.4 // from github.com/operator-framework/operator-sdk@v0.16.1-0.20200331173026-a00c1b716663
)
