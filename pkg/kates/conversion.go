package kates

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/datawire/ambassador/v2/pkg/kates/internal"
)

func ConvertObject(scheme *runtime.Scheme, src, dst runtime.Object) error {
	return internal.ConvertObject(scheme, src, dst)
}
