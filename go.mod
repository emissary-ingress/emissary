module github.com/datawire/apro

go 1.12

require (
	github.com/Jeffail/gabs v1.2.0
	github.com/aclements/go-moremath v0.0.0-20180329182055-b1aff36309c7
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05 // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/datawire/kat-backend v1.1.0
	github.com/datawire/liboauth2 v0.0.0-20190528220730-c19497f31033
	github.com/datawire/teleproxy v0.5.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dgryski/go-gk v0.0.0-20140819190930-201884a44051 // indirect
	github.com/die-net/lrucache v0.0.0-20181227122439-19a39ef22a11
	github.com/gobwas/glob v0.2.3
	github.com/gogo/googleapis v1.1.0
	github.com/gogo/protobuf v1.2.0
	github.com/google/uuid v1.1.0
	github.com/gorilla/mux v1.6.1
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f
	github.com/hashicorp/consul v1.4.4
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/lyft/ratelimit v1.3.0
	github.com/mailru/easyjson v0.0.0-20190312143242-1de009706dbe // indirect
	github.com/myzhan/boomer v0.0.0-20190321085146-9f3c9f575895
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/onsi/gomega v1.4.3
	github.com/pkg/errors v0.8.1
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/tsenart/vegeta v12.2.1+incompatible
	github.com/zeromq/goczmq v4.1.0+incompatible // indirect
	github.com/zeromq/gomq v0.0.0-20181008000130-95dc37dee5c4 // indirect
	golang.org/x/net v0.0.0-20190322120337-addf6b3196f6
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/tools v0.0.0-20190226205152-f727befe758c // indirect
	google.golang.org/grpc v1.18.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190111032252-67edc246be36
	k8s.io/apimachinery v0.0.0-20190119020841-d41becfba9ee
	k8s.io/client-go v10.0.0+incompatible
)

replace github.com/lyft/ratelimit v1.3.0 => ./vendor-ratelimit

replace github.com/tsenart/vegeta => github.com/datawire/vegeta v12.2.2-0.20190408190644-d94b721fc676+incompatible

// Lock the k8s.io dependencies together to the same version.
//
// The "v0.0.0-2019…" versions are the tag "kubernetes-1.13.4", but
// `go build` (in its infinite wisdom) wants to edit the file to not
// be useful to humans.  <https://github.com/golang/go/issues/27271>
// <https://github.com/golang/go/issues/25898>
//
// client-go v10 is the version corresponding to Kubernetes 1.13.
// These 4 packages should all be upgraded together (for example,
// client-go v10 won't build with the other packages using
// v1.14.0-alpha versions
// <https://github.com/kubernetes/client-go/issues/551>)
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190228180923-a9e421a79326
	k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
)

// Stupid hack for dependencies that both (1) erroneously include
// golint in their go.sum, and (2) erroneously refer to it as
// github.com/golang/lint instead of golang.org/x/lint
replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190227174305-5b3e6a55c961
