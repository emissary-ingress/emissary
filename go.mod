module github.com/datawire/apro

go 1.13

require (
	github.com/Jeffail/gabs v1.2.0
	github.com/aclements/go-moremath v0.0.0-20180329182055-b1aff36309c7
	github.com/datawire/ambassador v0.83.1-ea10
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/die-net/lrucache v0.0.0-20181227122439-19a39ef22a11
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-acme/lego/v3 v3.1.0
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7
	github.com/hashicorp/consul/api v1.1.0
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/jpillora/backoff v1.0.0
	github.com/lyft/goruntime v0.1.8
	github.com/lyft/gostats v0.2.6
	github.com/lyft/ratelimit v1.3.0
	github.com/mediocregopher/radix.v2 v0.0.0-20180603022615-94360be26253
	github.com/o1egl/paseto v1.0.0
	github.com/onsi/gomega v1.7.0
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/pkg/errors v0.8.1
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/tsenart/vegeta v12.2.1+incompatible
	gitlab.com/golang-commonmark/html v0.0.0-20180917080848-cfaf75183c4a // indirect
	gitlab.com/golang-commonmark/linkify v0.0.0-20191026162114-a0c2df6c8f82 // indirect
	gitlab.com/golang-commonmark/markdown v0.0.0-20181102083822-772775880e1f
	gitlab.com/golang-commonmark/mdurl v0.0.0-20180912090424-e5bce34c34f2 // indirect
	gitlab.com/golang-commonmark/puny v0.0.0-20180912090636-2cd490539afe // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190930134127-c5a3c61f89f3
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/grpc v1.23.0
	gopkg.in/src-d/go-billy.v4 v4.2.1
	gopkg.in/src-d/go-git.v4 v4.8.1
	gopkg.in/yaml.v2 v2.2.4
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
	k8s.io/api v0.0.0-20191004120104-195af9ec3521
	k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/client-go v0.0.0-20191004120905-f06fe3961ca9
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

// Allow pre-Go-modules blackfriday import paths to keep working
//   v1.y.z: github.com/russross/blackfriday (pre-modules) (used by gitlab.com/golang-commonmark/markdown.test)
//   v2.0.0: gopkg.in/russross/blackfriday.v2 (pre-modules) (also used by gitlab.com/golang-commonmark/markdown.test)
//   v2.0.1: github.com/russross/blackfriday/v2 (post-modules) (used by everything else)
replace gopkg.in/russross/blackfriday.v2 => github.com/russross/blackfriday v2.0.0+incompatible
