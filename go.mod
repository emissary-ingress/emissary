module github.com/datawire/ambassador

go 1.13

require (
	git.lukeshu.com/go/libsystemd v0.5.3
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/datawire/libk8s v0.0.0-20191023073802-9add2eb01af2
	github.com/datawire/pf v0.0.0-20180510150411-31a823f9495a
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ecodia/golang-awaitility v0.0.0-20180710094957-fb55e59708c7
	github.com/envoyproxy/protoc-gen-validate v0.0.15-0.20190405222122-d6164de49109
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.1.1
	github.com/gookit/color v1.2.3
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/consul/api v1.1.0
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/lyft/protoc-gen-star v0.4.4
	github.com/mholt/archiver/v3 v3.3.0
	github.com/miekg/dns v1.1.6
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mitchellh/protoc-gen-go-json v0.0.0-20190813154521-ece073100ced
	github.com/operator-framework/operator-sdk v0.16.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20191028145041-f83a4685e152
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271
	golang.org/x/sys v0.0.0-20191028164358-195ce5e7f934
	google.golang.org/grpc v1.24.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.7
	helm.sh/helm/v3 v3.0.2
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/cli-runtime v0.0.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/helm v2.16.5+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm

// Pinned to kubernetes-1.16.2
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191004123735-6bff60de4370
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191004120905-f06fe3961ca9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
