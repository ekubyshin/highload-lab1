// Package monitoring provides the foundation for HTTP instrumentation and route discovery.
package monitoring

import "github.com/prometheus/client_golang/prometheus"

// NewRegistry creates an isolated registry without default process or Go collectors.
// Pass it to the instrumentation and metrics handler instead of using global registration.
func NewRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}
