// This is in a separate 'internal' package so that I don't need to fus with names in
// `borrowed_webhook.go` to make things public/private, because I want to keep that file as close as
// possible to the upstream `webhook.go`.

package internal

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func ConvertObject(scheme *runtime.Scheme, src, dst runtime.Object) error {
	wh := &Webhook{
		scheme: scheme,
	}
	return wh.convertObject(src, dst)
}
