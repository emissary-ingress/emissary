module github.com/datawire/apro

go 1.13

require (
	github.com/Jeffail/gabs v1.2.0
	github.com/aclements/go-moremath v0.0.0-20180329182055-b1aff36309c7
	github.com/aws/aws-sdk-go v1.23.0
	github.com/datawire/ambassador v1.0.0-local-vendored-copy
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/die-net/lrucache v0.0.0-20181227122439-19a39ef22a11
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-acme/lego/v3 v3.1.0
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.1
	github.com/golang-migrate/migrate/v4 v4.8.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/gregjones/httpcache v0.0.0-20190203031600-7a902570cb17
	github.com/hashicorp/consul/api v1.1.0
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/jpillora/backoff v1.0.0
	github.com/lyft/goruntime v0.1.8
	github.com/lyft/gostats v0.2.6
	github.com/lyft/ratelimit v1.3.0
	github.com/mediocregopher/radix.v2 v0.0.0-20180603022615-94360be26253
	github.com/mholt/certmagic v0.8.3
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/o1egl/paseto v1.0.0
	github.com/onsi/gomega v1.8.1
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tsenart/vegeta v12.2.1+incompatible
	golang.org/x/crypto v0.0.0-20191028145041-f83a4685e152
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.24.0
	gopkg.in/src-d/go-billy.v4 v4.2.1
	gopkg.in/src-d/go-git.v4 v4.8.1
	gopkg.in/yaml.v2 v2.2.7
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
)

// Point at the local checkouts
replace (
	github.com/datawire/ambassador => ../ambassador
	github.com/datawire/apro => ./
	github.com/lyft/ratelimit => ./vendor-ratelimit
)

replace github.com/tsenart/vegeta => github.com/datawire/vegeta v12.2.2-0.20190408190644-d94b721fc676+incompatible

// Stupid hack for dependencies that both (1) erroneously include
// golint in their go.sum, and (2) erroneously refer to it as
// github.com/golang/lint instead of golang.org/x/lint
replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190227174305-5b3e6a55c961

// We need inherit these from ambassador.git's go.mod
replace (
	github.com/Azure/go-autorest v11.1.2+incompatible => github.com/Azure/go-autorest v13.3.2+incompatible
	github.com/docker/docker v0.0.0-00010101000000-000000000000 => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/docker/docker v1.4.2-0.20181221150755-2cb26cfe9cbf => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	k8s.io/api v0.0.0 => k8s.io/api v0.0.0-20191004120104-195af9ec3521
	k8s.io/api v0.17.2 => k8s.io/api v0.0.0-20191004120104-195af9ec3521
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apiextensions-apiserver v0.17.2 => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apimachinery v0.17.2 => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.0.0-20191004123735-6bff60de4370
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.0.0-20191004120905-f06fe3961ca9
	k8s.io/client-go v12.0.0+incompatible => k8s.io/client-go v0.0.0-20191004120905-f06fe3961ca9
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl v0.0.0 => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
