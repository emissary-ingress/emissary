package services

import (
	"context"
)

// Service defines a KAT backend service interface.
type Service interface {
	Start(context.Context) <-chan bool
}
