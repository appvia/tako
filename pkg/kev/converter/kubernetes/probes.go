package kubernetes

import (
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/appvia/kev/pkg/kev/config"
)

func LivenessProbeToV1Probe(lp config.LivenessProbe) (*v1.Probe, error) {
	return v1probe(lp.Type, lp.ProbeConfig)
}

func ReadinessProbeToV1Probe(rp config.ReadinessProbe) (*v1.Probe, error) {
	return v1probe(rp.Type, rp.ProbeConfig)
}

func v1probe(probeType string, pc config.ProbeConfig) (*v1.Probe, error) {
	pt, ok := config.ProbeTypeFromString(probeType)
	if !ok {
		return nil, errors.New("invalid probe type")
	}

	if pt == config.ProbeTypeNone {
		return nil, nil
	}

	return &v1.Probe{
		Handler:             handlerFromType(pt, pc),
		InitialDelaySeconds: int32(pc.InitialDelay.Seconds()),
		TimeoutSeconds:      int32(pc.Timeout.Seconds()),
		PeriodSeconds:       int32(pc.Period.Seconds()),
		SuccessThreshold:    int32(pc.FailureThreashold),
		FailureThreshold:    int32(pc.FailureThreashold),
	}, nil
}

func handlerFromType(probeType config.ProbeType, pc config.ProbeConfig) v1.Handler {
	switch probeType {
	case config.ProbeTypeTCP:
		return v1.Handler{
			TCPSocket: &v1.TCPSocketAction{
				Port: intstr.FromInt(pc.TCP.Port),
			},
		}
	case config.ProbeTypeHTTP:
		return v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: pc.HTTP.Path,
				Port: intstr.FromInt(pc.HTTP.Port),
			},
		}
	case config.ProbeTypeExec:
		return v1.Handler{
			Exec: &v1.ExecAction{
				Command: pc.Exec.Command,
			},
		}
	default:
	}

	return v1.Handler{}
}
