package v1

import (
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func convertFrom(src conversion.Hub, dst conversion.Convertible) error {
	scheme := conversionScheme()
	var cur k8sRuntime.Object = src
	for i := len(conversionIntermediates) - 1; i >= 0; i-- {
		gv := conversionIntermediates[i]
		var err error
		cur, err = scheme.ConvertToVersion(cur, gv)
		if err != nil {
			return err
		}
	}
	return scheme.Convert(cur, dst, nil)
}

func convertTo(src conversion.Convertible, dst conversion.Hub) error {
	scheme := conversionScheme()
	var cur k8sRuntime.Object = src
	for _, gv := range conversionIntermediates {
		var err error
		cur, err = scheme.ConvertToVersion(cur, gv)
		if err != nil {
			return err
		}
	}
	return scheme.Convert(cur, dst, nil)
}

func (dst *AuthService) ConvertFrom(src conversion.Hub) error      { return convertFrom(src, dst) }
func (src *AuthService) ConvertTo(dst conversion.Hub) error        { return convertTo(src, dst) }
func (dst *DevPortal) ConvertFrom(src conversion.Hub) error        { return convertFrom(src, dst) }
func (src *DevPortal) ConvertTo(dst conversion.Hub) error          { return convertTo(src, dst) }
func (dst *LogService) ConvertFrom(src conversion.Hub) error       { return convertFrom(src, dst) }
func (src *LogService) ConvertTo(dst conversion.Hub) error         { return convertTo(src, dst) }
func (dst *Mapping) ConvertFrom(src conversion.Hub) error          { return convertFrom(src, dst) }
func (src *Mapping) ConvertTo(dst conversion.Hub) error            { return convertTo(src, dst) }
func (dst *Module) ConvertFrom(src conversion.Hub) error           { return convertFrom(src, dst) }
func (src *Module) ConvertTo(dst conversion.Hub) error             { return convertTo(src, dst) }
func (dst *RateLimitService) ConvertFrom(src conversion.Hub) error { return convertFrom(src, dst) }
func (src *RateLimitService) ConvertTo(dst conversion.Hub) error   { return convertTo(src, dst) }
func (dst *KubernetesServiceResolver) ConvertFrom(src conversion.Hub) error {
	return convertFrom(src, dst)
}
func (src *KubernetesServiceResolver) ConvertTo(dst conversion.Hub) error { return convertTo(src, dst) }
func (dst *KubernetesEndpointResolver) ConvertFrom(src conversion.Hub) error {
	return convertFrom(src, dst)
}
func (src *KubernetesEndpointResolver) ConvertTo(dst conversion.Hub) error {
	return convertTo(src, dst)
}
func (dst *ConsulResolver) ConvertFrom(src conversion.Hub) error { return convertFrom(src, dst) }
func (src *ConsulResolver) ConvertTo(dst conversion.Hub) error   { return convertTo(src, dst) }
func (dst *TCPMapping) ConvertFrom(src conversion.Hub) error     { return convertFrom(src, dst) }
func (src *TCPMapping) ConvertTo(dst conversion.Hub) error       { return convertTo(src, dst) }
func (dst *TLSContext) ConvertFrom(src conversion.Hub) error     { return convertFrom(src, dst) }
func (src *TLSContext) ConvertTo(dst conversion.Hub) error       { return convertTo(src, dst) }
func (dst *TracingService) ConvertFrom(src conversion.Hub) error { return convertFrom(src, dst) }
func (src *TracingService) ConvertTo(dst conversion.Hub) error   { return convertTo(src, dst) }
