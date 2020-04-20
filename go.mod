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
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f
	github.com/hashicorp/consul/api v1.1.0
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/jpillora/backoff v1.0.0
	github.com/lyft/goruntime v0.1.8
	github.com/lyft/gostats v0.2.6
	github.com/lyft/ratelimit v1.3.0-local-vendored-copy
	github.com/mediocregopher/radix.v2 v0.0.0-20180603022615-94360be26253
	github.com/mholt/certmagic v0.8.3
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/o1egl/paseto v1.0.0
	github.com/onsi/gomega v1.7.0
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/sirupsen/logrus v1.4.3-0.20200306102446-7ea96a3284ed
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tsenart/vegeta v12.2.1+incompatible
	golang.org/x/crypto v0.0.0-20191028145041-f83a4685e152
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/grpc v1.24.0
	gopkg.in/src-d/go-billy.v4 v4.3.2
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.7
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
	k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/klog v1.0.0
	sigs.k8s.io/yaml v1.1.0
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
	github.com/docker/docker v0.0.0-00010101000000-000000000000 => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/docker/docker v1.4.2-0.20181221150755-2cb26cfe9cbf => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
)
