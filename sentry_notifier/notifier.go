package sentry_notifier

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
)

type Notifier struct {
	client *sentry.Hub
	logger *slog.Logger
}

func NewNotifier(client *sentry.Hub, logger *slog.Logger) *Notifier {
	return &Notifier{client: client, logger: logger}
}

func (n *Notifier) CaptureError(e error) {
	errId := n.client.CaptureEvent(buildEvent(e))
	if errId != nil {
		n.logger.Error(e.Error(), "sentry_id", *errId)
	}
}
