package v2

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func convert(src, dst runtime.Object) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(src, dst, nil)
}

func (dst *AuthService) ConvertFrom(src conversion.Hub) error               { return convert(src, dst) }
func (src *AuthService) ConvertTo(dst conversion.Hub) error                 { return convert(src, dst) }
func (dst *DevPortal) ConvertFrom(src conversion.Hub) error                 { return convert(src, dst) }
func (src *DevPortal) ConvertTo(dst conversion.Hub) error                   { return convert(src, dst) }
func (dst *Host) ConvertFrom(src conversion.Hub) error                      { return convert(src, dst) }
func (src *Host) ConvertTo(dst conversion.Hub) error                        { return convert(src, dst) }
func (dst *LogService) ConvertFrom(src conversion.Hub) error                { return convert(src, dst) }
func (src *LogService) ConvertTo(dst conversion.Hub) error                  { return convert(src, dst) }
func (dst *Mapping) ConvertFrom(src conversion.Hub) error                   { return convert(src, dst) }
func (src *Mapping) ConvertTo(dst conversion.Hub) error                     { return convert(src, dst) }
func (dst *Module) ConvertFrom(src conversion.Hub) error                    { return convert(src, dst) }
func (src *Module) ConvertTo(dst conversion.Hub) error                      { return convert(src, dst) }
func (dst *RateLimitService) ConvertFrom(src conversion.Hub) error          { return convert(src, dst) }
func (src *RateLimitService) ConvertTo(dst conversion.Hub) error            { return convert(src, dst) }
func (dst *KubernetesServiceResolver) ConvertFrom(src conversion.Hub) error { return convert(src, dst) }
func (src *KubernetesServiceResolver) ConvertTo(dst conversion.Hub) error   { return convert(src, dst) }
func (dst *KubernetesEndpointResolver) ConvertFrom(src conversion.Hub) error {
	return convert(src, dst)
}
func (src *KubernetesEndpointResolver) ConvertTo(dst conversion.Hub) error { return convert(src, dst) }
func (dst *ConsulResolver) ConvertFrom(src conversion.Hub) error           { return convert(src, dst) }
func (src *ConsulResolver) ConvertTo(dst conversion.Hub) error             { return convert(src, dst) }
func (dst *TCPMapping) ConvertFrom(src conversion.Hub) error               { return convert(src, dst) }
func (src *TCPMapping) ConvertTo(dst conversion.Hub) error                 { return convert(src, dst) }
func (dst *TLSContext) ConvertFrom(src conversion.Hub) error               { return convert(src, dst) }
func (src *TLSContext) ConvertTo(dst conversion.Hub) error                 { return convert(src, dst) }
func (dst *TracingService) ConvertFrom(src conversion.Hub) error           { return convert(src, dst) }
func (src *TracingService) ConvertTo(dst conversion.Hub) error             { return convert(src, dst) }
