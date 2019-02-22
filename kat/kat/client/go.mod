module github.com/ambassador/datawire/kat/client

require (
	github.com/ambassador/datawire/kat/backend/echo v0.0.0
	github.com/gogo/protobuf v1.2.0
	github.com/golang/protobuf v1.2.1-0.20190205222052-c823c79ea157
	github.com/gorilla/websocket v1.4.0
	google.golang.org/grpc v1.18.0
)

replace github.com/ambassador/datawire/kat/backend/echo => ../../backend/echo
