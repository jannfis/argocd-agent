// Copyright 2024 The argocd-agent Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
)

// ApplicationWatcherMetrics holds metrics about Applications watched by the agent
type ApplicationWatcherMetrics struct {
	AppsWatched prometheus.Gauge
	AppsAdded   prometheus.Counter
	AppsUpdated prometheus.Counter
	AppsRemoved prometheus.Counter
	Errors      prometheus.Counter
}

type ApplicationClientMetrics struct {
	AppsCreated *prometheus.CounterVec
	AppsUpdated *prometheus.CounterVec
	AppsDeleted *prometheus.CounterVec
	Errors      prometheus.Counter
}

// NewApplicationWatcherMetrics returns a new instance of ApplicationMetrics
func NewApplicationWatcherMetrics() *ApplicationWatcherMetrics {
	am := &ApplicationWatcherMetrics{
		AppsWatched: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "argocd_agent_watcher_applications_watched",
			Help: "The total number of applications watched by the agent",
		}),
		AppsAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_watcher_applications_added",
			Help: "The number of applications that have been added to the agent",
		}),
		AppsUpdated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_watcher_applications_updated",
			Help: "The number of applications that have been updated",
		}),
		AppsRemoved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_watcher_applications_removed",
			Help: "The number of applications that have been removed from the agent",
		}),
	}
	return am
}

func NewApplicationClientMetrics() *ApplicationClientMetrics {
	return &ApplicationClientMetrics{
		AppsCreated: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "argocd_agent_client_applications_created",
			Help: "The total number of applications created by the application client",
		}, []string{"namespace"}),
		AppsUpdated: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "argocd_agent_client_applications_updated",
			Help: "The total number of applications updated by the application client",
		}, []string{"namespace"}),
		AppsDeleted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "argocd_agent_client_applications_deleted",
			Help: "The total number of applications deleted by the application client",
		}, []string{"namespace"}),
		Errors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "argocd_agent_client_applications_errors",
			Help: "The total number of applications deleted by the application client",
		}),
	}
}

// func (am *ApplicationWatcherMetrics) SetWatched(num int64) {
// 	am.AppsWatched.Set(float64(num))
// }

// func (am *ApplicationWatcherMetrics) AppAdded() {
// 	am.AppsWatched.Inc()
// 	am.AppsAdded.Inc()
// }

// func (am *ApplicationWatcherMetrics) AppRemoved() {
// 	am.AppsWatched.Dec()
// 	am.AppsRemoved.Inc()
// }

// func (am *ApplicationWatcherMetrics) AppUpdated() {
// 	am.AppsUpdated.Inc()
// }

// func (am *ApplicationWatcherMetrics) Error() {
// 	am.Errors.Inc()
// }
