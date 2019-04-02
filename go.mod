module github.com/datawire/apro

go 1.12

require (
	github.com/Jeffail/gabs v1.2.0
	github.com/PuerkitoBio/goquery v1.5.0
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/datawire/consul-x v0.0.0-20190305163622-7683365ac879
	github.com/datawire/kat-backend v1.1.0
	github.com/datawire/teleproxy v0.3.16
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/die-net/lrucache v0.0.0-20181227122439-19a39ef22a11
	github.com/gobwas/glob v0.2.3
	github.com/gogo/googleapis v1.1.0
	github.com/gogo/protobuf v1.2.0
	github.com/google/uuid v1.1.0
	github.com/gorilla/mux v1.6.1
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f
	github.com/hashicorp/consul v1.4.2
	github.com/hashicorp/go-sockaddr v1.0.1 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/lyft/ratelimit v1.3.0
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.6 // indirect
	github.com/miekg/dns v1.1.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/pkg/errors v0.8.1
	github.com/posener/complete v1.2.1 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/urfave/negroni v1.0.0 // indirect
	golang.org/x/crypto v0.0.0-20190228161510-8dd112bcdc25 // indirect
	golang.org/x/net v0.0.0-20190301231341-16b79f2e4e95
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/sys v0.0.0-20190305064518-30e92a19ae4a // indirect
	golang.org/x/tools v0.0.0-20190308174544-00c44ba9c14f // indirect
	google.golang.org/grpc v1.18.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/apimachinery v0.0.0-20190119020841-d41becfba9ee
	k8s.io/client-go v10.0.0+incompatible
)

replace github.com/lyft/ratelimit v1.3.0 => ./vendor-ratelimit

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
