package event

import (
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type EventType int32

const (
	EventTypeUnknown EventType = iota
	EventTypeAddApp
	EventTypeDeleteApp
	EventTypeUpdateAppSpec
	EventTypeUpdateAppStatus
	EventTypeUpdateAppOperation
)

type Event struct {
	Type           EventType
	Application    *v1alpha1.Application
	AppProject     *v1alpha1.AppProject     // Forward compatibility
	ApplicationSet *v1alpha1.ApplicationSet // Forward compatibility
	Created        *time.Time
	Processed      *time.Time
}

func (et EventType) String() string {
	switch et {
	case EventTypeUnknown:
		return "unknown"
	case EventTypeAddApp:
		return "add"
	case EventTypeDeleteApp:
		return "delete"
	case EventTypeUpdateAppSpec:
		return "update_spec"
	case EventTypeUpdateAppOperation:
		return "update_operation"
	case EventTypeUpdateAppStatus:
		return "update_status"
	default:
		return "unknown"
	}
}
