package targetpb

import (
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

func FromProm(group *targetgroup.Group) *Targetgroup {
	tg := &Targetgroup{
		Labels: make(map[string]string, len(group.Labels)),
	}

	for k, v := range group.Labels {
		tg.Labels[string(k)] = string(v)
	}

	tg.Targets = make([]string, 0, len(group.Targets))
	for _, target := range group.Targets {
		addr := string(target[model.AddressLabel])
		tg.Targets = append(tg.Targets, addr)
	}

	return tg
}
