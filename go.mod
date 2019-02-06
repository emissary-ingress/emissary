module github.com/datawire/apro

require (
	contrib.go.opencensus.io/exporter/ocagent v0.4.1 // indirect
	github.com/Azure/go-autorest v11.3.0+incompatible // indirect
	github.com/Masterminds/sprig v2.17.1+incompatible // indirect
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/datawire/teleproxy v0.3.12
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gobwas/glob v0.2.3
	github.com/google/uuid v1.1.0
	github.com/gophercloud/gophercloud v0.0.0-20190115030418-a9f90060ebd9 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/hashicorp/consul v1.4.0
	github.com/hashicorp/go-cleanhttp v0.5.0 // indirect
	github.com/hashicorp/go-rootcerts v0.0.0-20160503143440-6bb64b370b90 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/hashicorp/memberlist v0.1.3 // indirect
	github.com/hashicorp/serf v0.8.1 // indirect
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lyft/ratelimit v1.3.0
	github.com/mitchellh/go-homedir v1.0.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/pkg/errors v0.8.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/urfave/negroni v1.0.0
	golang.org/x/crypto v0.0.0-20190103213133-ff983b9c42bc // indirect
	golang.org/x/net v0.0.0-20190110200230-915654e7eabc // indirect
	golang.org/x/oauth2 v0.0.0-20190111185915-36a7019397c4 // indirect
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4
	golang.org/x/sys v0.0.0-20190114130336-2be517255631 // indirect
	google.golang.org/api v0.1.0 // indirect
	google.golang.org/genproto v0.0.0-20190111180523-db91494dd46c // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/apimachinery v0.0.0-20190111195121-fa6ddc151d63
)

replace github.com/lyft/ratelimit v1.3.0 => ./vendor-ratelimit
