package handler

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/dmitryvakulenko/errors"
	"github.com/google/uuid"
)

const (
	errorIdKey         = "errorId"
	errorMessageKey    = "errorMessage"
	errorTypeKey       = "errorType"
	errorStackTraceKey = "errorStackTrace"
)

type (
	Enrich struct {
		handlers []slog.Handler
		minLevel slog.Level
	}

	stackTrace []uintptr
)

func (s stackTrace) LogValue() slog.Value {
	return slog.GroupValue()
}

func NewEnrich(handlers ...slog.Handler) *Enrich {
	hs := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			hs = append(hs, h)
		}
	}

	return &Enrich{handlers: hs}
}

func (h *Enrich) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, dst := range h.handlers {
		if dst.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h *Enrich) Handle(ctx context.Context, r slog.Record) error {
	r2 := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	var firstErr error
	r.Attrs(func(a slog.Attr) bool {
		v := a.Value.Any()
		err, ok := v.(error)
		if !ok || err == nil {
			r2.AddAttrs(a)
		}

		firstErr = err

		return false
	})

	if firstErr == nil {
		return h.callNext(ctx, r)
	}

	r2.AddAttrs(slog.String(errorIdKey, h.generateErrorId()))

	var lastMeta *errors.Error
	tmp := firstErr
	var resultMsg = firstErr.Error()
	for {
		if !errors.As(tmp, &lastMeta) {
			break
		}

		if resultMsg == "" {
			resultMsg = tmp.Error()
		}
		r2.AddAttrs(lastMeta.LogAttrs()...)

		tmp = lastMeta.Unwrap()
		if tmp == nil {
			break
		}
	}

	r2.AddAttrs(slog.String(errorMessageKey, resultMsg))

	if lastMeta != nil {
		r2.AddAttrs(
			slog.String(errorTypeKey, fmt.Sprintf("%s:%s", lastMeta.Kind.String(), lastMeta.Code.String())),
			slog.Any(errorStackTraceKey, stackTrace(lastMeta.Stacktrace)),
		)
	}

	return h.callNext(ctx, r2)
}

func (h *Enrich) callNext(ctx context.Context, r slog.Record) error {
	var handlerErr error
	for _, dst := range h.handlers {
		if !dst.Enabled(ctx, r.Level) {
			continue
		}

		if err := dst.Handle(ctx, r); err != nil && handlerErr == nil {
			handlerErr = err
		}
	}

	return handlerErr
}

func (h *Enrich) generateErrorId() string {
	id := uuid.New()
	return hex.EncodeToString(id[:])
}

func (h *Enrich) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(h.handlers) == 0 {
		return h
	}
	hs := make([]slog.Handler, 0, len(h.handlers))
	for _, dst := range h.handlers {
		hs = append(hs, dst.WithAttrs(attrs))
	}

	return &Enrich{handlers: hs}
}

func (h *Enrich) WithGroup(name string) slog.Handler {
	if len(h.handlers) == 0 {
		return h
	}
	hs := make([]slog.Handler, 0, len(h.handlers))
	for _, dst := range h.handlers {
		hs = append(hs, dst.WithGroup(name))
	}

	return &Enrich{handlers: hs}
}
