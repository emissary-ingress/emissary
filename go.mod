module github.com/datawire/apro

require (
	github.com/datawire/teleproxy v0.3.16
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gobwas/glob v0.2.3
	github.com/google/uuid v1.1.0
	github.com/hashicorp/consul v1.4.2
	github.com/hashicorp/go-cleanhttp v0.5.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.0 // indirect
	github.com/hashicorp/serf v0.8.2 // indirect
	github.com/jcuga/golongpoll v0.0.0-20180711123949-939e3befd837
	github.com/lyft/ratelimit v1.3.0
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/pkg/errors v0.8.1
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/urfave/negroni v1.0.0
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/lyft/ratelimit v1.3.0 => ./vendor-ratelimit
