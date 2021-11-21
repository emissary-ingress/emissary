package v3alpha1

import (
	v2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/pkg/errors"
	conv "sigs.k8s.io/controller-runtime/pkg/conversion"
)

var _ conv.Convertible = &AuthService{}
var _ conv.Convertible = &ConsulResolver{}
var _ conv.Convertible = &DevPortal{}
var _ conv.Convertible = &Host{}
var _ conv.Convertible = &KubernetesEndpointResolver{}
var _ conv.Convertible = &KubernetesServiceResolver{}
var _ conv.Convertible = &LogService{}
var _ conv.Convertible = &Mapping{}
var _ conv.Convertible = &Module{}
var _ conv.Convertible = &RateLimitService{}
var _ conv.Convertible = &TCPMapping{}
var _ conv.Convertible = &TLSContext{}

func (in *AuthService) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.AuthService), in, nil)
}

func (in *AuthService) ConvertTo(hub conv.Hub) error {
	// s, err := SchemeBuilder.Build()
	// if err != nil {
	// 	return err
	// }
	// return s.Convert(in, hub.(*v2.AuthService), nil)
	return errors.New("will not convert from v3alpha1 AuthService back to v2 AuthService")
}

func (in *ConsulResolver) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.ConsulResolver), in, nil)
}

func (in *ConsulResolver) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 ConsulResolver back to v2 ConsulResolver")
}

func (in *DevPortal) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.DevPortal), in, nil)
}

func (in *DevPortal) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 DevPortal back to v2 DevPortal")
}

func (in *Host) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.Host), in, nil)
}

func (in *Host) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 Host back to v2 Host")
}

func (in *KubernetesEndpointResolver) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.KubernetesEndpointResolver), in, nil)
}

func (in *KubernetesEndpointResolver) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 KubernetesEndpointResolver back to v2 KubernetesEndpointResolver")
}

func (in *KubernetesServiceResolver) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.KubernetesServiceResolver), in, nil)
}

func (in *KubernetesServiceResolver) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 KubernetesServiceResolver back to v2 KubernetesServiceResolver")
}

func (in *LogService) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.LogService), in, nil)
}

func (in *LogService) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 LogService back to v2 LogService")
}

func (in *Mapping) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.Mapping), in, nil)
}

func (in *Mapping) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 Mapping back to v2 Mapping")
}

func (in *Module) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.Module), in, nil)
}

func (in *Module) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 Module back to v2 Module")
}

func (in *RateLimitService) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.RateLimitService), in, nil)
}

func (in *RateLimitService) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 RateLimitService back to v2 RateLimitService")
}

func (in *TCPMapping) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.TCPMapping), in, nil)
}

func (in *TCPMapping) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 TCPMapping back to v2 TCPMapping")
}

func (in *TLSContext) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.TLSContext), in, nil)
}

func (in *TLSContext) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 TLSContext back to v2 TLSContext")
}

func (in *TracingService) ConvertFrom(hub conv.Hub) error {
	s, err := SchemeBuilder.Build()
	if err != nil {
		return err
	}
	return s.Convert(hub.(*v2.TracingService), in, nil)
}

func (in *TracingService) ConvertTo(hub conv.Hub) error {
	return errors.New("will not convert from v3alpha1 TracingService back to v2 TracingService")
}
