package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
)

// ApplicationMetrics holds metrics about Applications watched by the agent
type ApplicationMetrics struct {
	appsWatched prometheus.Gauge
	appsAdded   prometheus.Counter
	appsUpdated prometheus.Counter
	appsRemoved prometheus.Counter
	errors      prometheus.Counter
}

// NewApplicationMetrics returns a new instance of ApplicationMetrics
func NewApplicationMetrics() *ApplicationMetrics {
	am := &ApplicationMetrics{
		appsWatched: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "argocd_agent_applications_watched",
			Help: "The total number of apps watched by the agent",
		}),
		appsAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_applications_added",
			Help: "The number of applicatins that have been added to the agent",
		}),
		appsUpdated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_applications_updated",
			Help: "The number of applications that have been updated",
		}),
		appsRemoved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_applications_removed",
			Help: "The number of applications that have been removed from the agent",
		}),
	}
	return am
}

func (am *ApplicationMetrics) SetWatched(num int64) {
	am.appsWatched.Set(float64(num))
}

func (am *ApplicationMetrics) AddApp() {
	am.appsWatched.Inc()
	am.appsAdded.Inc()
}

func (am *ApplicationMetrics) RemoveApp() {
	am.appsWatched.Dec()
	am.appsRemoved.Inc()
}

func (am *ApplicationMetrics) UpdateApp() {
	am.appsUpdated.Inc()
}

func (am *ApplicationMetrics) Error() {
	am.errors.Inc()
}
