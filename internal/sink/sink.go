package sink

import "github.com/jannfis/argocd-agent/internal/application"

type SyncStatus int

const (
	SyncStatusUnknown SyncStatus = 0
	SyncStatusOK      SyncStatus = 1
	SyncStatusBad     SyncStatus = 2
)

type SyncResult struct{}

type Sink interface {
	Store(app *application.AppSync) (*SyncResult, error)
}
