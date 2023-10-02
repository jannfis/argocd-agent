package application

import (
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type AppSync struct {
	local *argo.Application
}
