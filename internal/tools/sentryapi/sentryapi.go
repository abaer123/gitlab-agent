package sentryapi

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	_ Hub = &hubWrap{}
)

// Hub is an interface of the sentry.Hub.
type Hub interface {
	Clone() Hub
	LastEventID() sentry.EventID
	Scope() *sentry.Scope
	PushScope() *sentry.Scope
	PopScope()
	WithScope(f func(*sentry.Scope))
	ConfigureScope(f func(*sentry.Scope))
	CaptureEvent(event *sentry.Event) *sentry.EventID
	CaptureMessage(message string) *sentry.EventID
	CaptureException(exception error) *sentry.EventID
	AddBreadcrumb(breadcrumb *sentry.Breadcrumb, hint *sentry.BreadcrumbHint)
	Recover(err interface{}) *sentry.EventID
	RecoverWithContext(ctx context.Context, err interface{}) *sentry.EventID
	Flush(timeout time.Duration) bool
}

func NewHub(client *sentry.Client, scope *sentry.Scope) Hub {
	return hubWrap{
		Hub: sentry.NewHub(client, scope),
	}
}

// hubWrap is an implementation of the Hub interface.
// The only reason this wrapper is needed is to adapt "Clone() *sentry.Hub" into "Clone() Hub".
type hubWrap struct {
	Hub *sentry.Hub
}

func (h hubWrap) Clone() Hub {
	return hubWrap{
		Hub: h.Hub.Clone(),
	}
}

func (h hubWrap) LastEventID() sentry.EventID {
	return h.Hub.LastEventID()
}

func (h hubWrap) Scope() *sentry.Scope {
	return h.Hub.Scope()
}

func (h hubWrap) PushScope() *sentry.Scope {
	return h.Hub.PushScope()
}

func (h hubWrap) PopScope() {
	h.Hub.PopScope()
}

func (h hubWrap) WithScope(f func(*sentry.Scope)) {
	h.Hub.WithScope(f)
}

func (h hubWrap) ConfigureScope(f func(*sentry.Scope)) {
	h.Hub.ConfigureScope(f)
}

func (h hubWrap) CaptureEvent(event *sentry.Event) *sentry.EventID {
	return h.Hub.CaptureEvent(event)
}

func (h hubWrap) CaptureMessage(message string) *sentry.EventID {
	return h.Hub.CaptureMessage(message)
}

func (h hubWrap) CaptureException(exception error) *sentry.EventID {
	return h.Hub.CaptureException(exception)
}

func (h hubWrap) AddBreadcrumb(breadcrumb *sentry.Breadcrumb, hint *sentry.BreadcrumbHint) {
	h.Hub.AddBreadcrumb(breadcrumb, hint)
}

func (h hubWrap) Recover(err interface{}) *sentry.EventID {
	return h.Hub.Recover(err)
}

func (h hubWrap) RecoverWithContext(ctx context.Context, err interface{}) *sentry.EventID {
	return h.Hub.RecoverWithContext(ctx, err)
}

func (h hubWrap) Flush(timeout time.Duration) bool {
	return h.Hub.Flush(timeout)
}
