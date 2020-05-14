package services

// Service defines a KAT backend service interface.
type Service interface {
	Start() <-chan bool
}
