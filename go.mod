module github.com/datawire/apro

go 1.13

require (
	cloud.google.com/go v0.34.0 // indirect
	github.com/Jeffail/gabs v1.2.0
	github.com/aclements/go-moremath v0.0.0-20180329182055-b1aff36309c7
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05 // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/datawire/ambassador v0.72.0
	github.com/datawire/liboauth2 v0.0.0-20190930180758-8a034a1de052
	github.com/datawire/teleproxy v0.7.3-0.20190913173620-2f688be1248e
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dgryski/go-gk v0.0.0-20140819190930-201884a44051 // indirect
	github.com/die-net/lrucache v0.0.0-20181227122439-19a39ef22a11
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.6.1
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7
	github.com/hashicorp/consul/api v1.1.0
	github.com/hashicorp/go-msgpack v0.5.4 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/lyft/goruntime v0.1.8
	github.com/lyft/gostats v0.2.6
	github.com/lyft/ratelimit v1.3.0
	github.com/mediocregopher/radix.v2 v0.0.0-20180603022615-94360be26253
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/myzhan/boomer v0.0.0-20190321085146-9f3c9f575895
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/onsi/gomega v1.7.0
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/pascaldekloe/goe v0.1.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/russross/blackfriday v2.0.0+incompatible // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.4.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/tsenart/vegeta v12.2.1+incompatible
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/zeromq/goczmq v4.1.0+incompatible // indirect
	github.com/zeromq/gomq v0.0.0-20181008000130-95dc37dee5c4 // indirect
	gitlab.com/golang-commonmark/html v0.0.0-20180917080848-cfaf75183c4a // indirect
	gitlab.com/golang-commonmark/linkify v0.0.0-20180917065525-c22b7bdb1179 // indirect
	gitlab.com/golang-commonmark/markdown v0.0.0-20181102083822-772775880e1f
	gitlab.com/golang-commonmark/mdurl v0.0.0-20180912090424-e5bce34c34f2 // indirect
	gitlab.com/golang-commonmark/puny v0.0.0-20180912090636-2cd490539afe // indirect
	gitlab.com/opennota/wd v0.0.0-20180912061657-c5d65f63c638 // indirect
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/oauth2 v0.0.0-20181203162652-d668ce993890 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/genproto v0.0.0-20190123001331-8819c946db44 // indirect
	google.golang.org/grpc v1.23.0
	gopkg.in/russross/blackfriday.v2 v2.0.0-00010101000000-000000000000
	gopkg.in/src-d/go-billy.v4 v4.2.1
	gopkg.in/src-d/go-git.v4 v4.8.1
	gopkg.in/yaml.v2 v2.2.2
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
	k8s.io/api v0.0.0-20190816221958-51687e09eeea
	k8s.io/apimachinery v0.0.0-20190816201409-1714e684133b
	k8s.io/client-go v11.0.1-0.20190816222228-6d55c1b1f1ca+incompatible
)

replace github.com/lyft/ratelimit v1.3.0 => ./vendor-ratelimit

replace github.com/datawire/ambassador => ./ambassador

replace github.com/tsenart/vegeta => github.com/datawire/vegeta v12.2.2-0.20190408190644-d94b721fc676+incompatible

// Stupid hack for dependencies that both (1) erroneously include
// golint in their go.sum, and (2) erroneously refer to it as
// github.com/golang/lint instead of golang.org/x/lint
replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190227174305-5b3e6a55c961

// https://github.com/russross/blackfriday/issues/500
replace gopkg.in/russross/blackfriday.v2 => github.com/russross/blackfriday/v2 v2.0.1

// Fix invalid pseudo-version that Go 1.13 complains about.
replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
