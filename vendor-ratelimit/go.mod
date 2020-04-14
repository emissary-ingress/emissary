module github.com/lyft/ratelimit

go 1.13

require (
	github.com/datawire/ambassador v1.0.0-local-vendored-copy
	github.com/datawire/apro v1.0.0-local-vendored-copy
	github.com/golang/mock v1.2.0
	github.com/gorilla/mux v1.7.3
	github.com/kavu/go_reuseport v1.2.0
	github.com/kelseyhightower/envconfig v1.1.0
	github.com/lyft/goruntime v0.1.8
	github.com/lyft/gostats v0.2.6
	github.com/mediocregopher/radix.v2 v0.0.0-20180603022615-94360be26253
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	google.golang.org/grpc v1.23.0
	gopkg.in/yaml.v2 v2.2.4
)

// Point at the local checkouts
replace (
	github.com/datawire/ambassador => ../../ambassador
	github.com/datawire/apro => ../
	github.com/lyft/ratelimit => ./
)

// Inherit nescessary replacements from the apro go.mod
replace gopkg.in/russross/blackfriday.v2 => github.com/russross/blackfriday v2.0.0+incompatible
