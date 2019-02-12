package services

// GRPC server object (all fields are required).
type GRPC struct {
	Port       int16
	SecurePort int16
	Cert       string
	Key        string
}

// Start initializes the HTTP server.
func (g *GRPC) Start() <-chan bool {
	exited := make(chan bool)

	go func() {
		close(exited)
	}()

	go func() {
		close(exited)
	}()

	return exited
}
