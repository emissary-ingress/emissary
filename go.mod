module github.com/datawire/ambassador-ratelimit

require (
	cloud.google.com/go v0.0.0-20160913182117-3b1ae45394a2
	github.com/Azure/go-autorest v11.1.0+incompatible
	github.com/Masterminds/semver v1.2.2
	github.com/Masterminds/sprig v0.0.0-20181119200632-3af02e5631fd
	github.com/aokoli/goutils v0.0.0-20181203091226-41ac8693c5c1
	github.com/datawire/teleproxy v0.0.0-20181207190820-b9379890f5d0
	github.com/davecgh/go-spew v0.0.0-20170626231645-782f4967f2dc
	github.com/dgrijalva/jwt-go v0.0.0-20160705203006-01aeca54ebda
	github.com/gogo/protobuf v0.0.0-20171007142547-342cbe0a0415
	github.com/golang/protobuf v1.2.0
	github.com/google/btree v0.0.0-20160524151835-7d79101e329e
	github.com/google/gofuzz v0.0.0-20161122191042-44d81051d367
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/google/uuid v0.0.0-20161128191214-064e2069ce9c
	github.com/googleapis/gnostic v0.0.0-20170729233727-0c5108395e2d
	github.com/gophercloud/gophercloud v0.0.0-20180330165814-781450b3c4fc
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7
	github.com/hashicorp/golang-lru v0.0.0-20160207214719-a0d98a5f2880
	github.com/huandu/xstrings v0.0.0-20180906151751-8bbcf2f9ccb5
	github.com/imdario/mergo v0.0.0-20171009183408-7fe0c75c13ab
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/json-iterator/go v0.0.0-20180701071628-ab8a2e0c74be
	github.com/lyft/ratelimit v1.3.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	golang.org/x/crypto v0.0.0-20180808211826-de0752318171
	golang.org/x/net v0.0.0-20180724234803-3673e40ba225
	golang.org/x/oauth2 v0.0.0-20170412232759-a6bd8cefa181
	golang.org/x/sys v0.0.0-20180224232135-f6cff0780e54
	golang.org/x/text v0.3.0
	golang.org/x/time v0.0.0-20161028155119-f51c12702a4d
	google.golang.org/appengine v1.3.0
	gopkg.in/inf.v0 v0.9.0
	gopkg.in/yaml.v2 v2.2.1
	k8s.io/api v0.0.0-20181121071145-b7bd5f2d334c
	k8s.io/apimachinery v0.0.0-20181121071008-d4f83ca2e260
	k8s.io/client-go v0.0.0-20181121071415-8abb21031259
	k8s.io/klog v0.0.0-20181108234604-8139d8cb77af
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/lyft/ratelimit v1.3.0 => ./vendor-ratelimit
