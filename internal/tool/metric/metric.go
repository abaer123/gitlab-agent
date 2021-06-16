package metric

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

func Register(registerer prometheus.Registerer, toRegister ...prometheus.Collector) (func(), error) {
	registered := make([]prometheus.Collector, 0, len(toRegister))
	cleanup := func() {
		for _, c := range registered {
			registerer.Unregister(c)
		}
	}
	for _, c := range toRegister {
		if err := registerer.Register(c); err != nil {
			cleanup()
			return nil, fmt.Errorf("registering %T: %v", c, err)
		}
		registered = append(registered, c)
	}
	return cleanup, nil
}
