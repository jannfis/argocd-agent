package agent

import (
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/glob"
	"github.com/jannfis/argocd-agent/internal/filter"
)

// DefaultFilterChain returns a FilterChain with a set of default filters that
// the agent will evaluate for every change.
func (a *Agent) DefaultFilterChain() *filter.Chain {
	fc := &filter.Chain{}

	// Admit based on namespace of the application
	fc.AppendAdmitFilter(func(app *v1alpha1.Application) bool {
		admit := glob.MatchStringInList(append([]string{a.opts.namespace}, a.opts.namespaces...), app.Namespace, false)
		return admit
	})

	return fc
}

func (a *Agent) WithLabelFilter() {
}
