package health

import (
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

type Probe interface {
	Check() bool
}

// StaticProbe always returns the specified value. This is primarily for tests but has some limited value in development
// scenarios.
type StaticProbe struct {
	Value bool
}

func (p *StaticProbe) Check() bool {
	return p.Value
}

// RandomProbe returns a randomly selected boolean value. This is primarily for tests and development but has some
// limited value in development.
//type RandomProbe struct {
//	rand *rand.Rand
//}
//
//func (p *RandomProbe) Check() bool {
//	return p.rand.Intn(2) == 0
//}

// MultiProbe executes zero or more probes.
type MultiProbe struct {
	Logger types.Logger
	probes map[string]Probe
}

func (p *MultiProbe) RegisterProbe(name string, probe Probe) {
	if p.probes == nil {
		p.probes = make(map[string]Probe)
	}

	p.probes[name] = probe
}

func (p *MultiProbe) Check() bool {
	if len(p.probes) == 0 {
		p.Logger.Warn("no probes registered, assuming healthy")
		return true
	}

	healthy := true
	for name, probe := range p.probes {
		probeResult := probe.Check()
		l := p.Logger.
			WithField("name", name).
			WithField("result", probeResult)

		if !probeResult {
			l.Errorln("probe failed")
			healthy = false
			break
		}

		l.Debug("probe succeeded")
	}

	return healthy
}

// SyncProbeHandler runs probes on demand as the handler is invoked.
//
// NOTE: An alternative implementation is to perform the checks asynchronously and just return a pre-computed result
//       when the handler is invoked. That strategy is often employed when probes are expensive/slow and blocking
//       the probing mechanism (e.g. Kubernetes liveness or readiness probes) would lead to failures.
//type SyncProbeHandler struct {
//	HealthinessProbe Probe
//	ReadinessProbe   Probe
//}
//
//func (h *SyncProbeHandler) QueryHealthiness(w http.ResponseWriter, r *http.Request) {
//	if h.HealthinessProbe.Check() {
//		w.WriteHeader(http.StatusOK)
//	} else {
//		w.WriteHeader(http.StatusInternalServerError)
//	}
//}
//
//func (h *SyncProbeHandler) QueryReadiness(w http.ResponseWriter, r *http.Request) {
//	if h.ReadinessProbe.Check() {
//		w.WriteHeader(http.StatusOK)
//	} else {
//		w.WriteHeader(http.StatusInternalServerError)
//	}
//}
