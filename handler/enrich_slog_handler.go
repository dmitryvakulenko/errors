package handler

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/dmitryvakulenko/errors/rich_error"
	"github.com/google/uuid"
)

const (
	errorIdKey         = "errorId"
	errorMessageKey    = "errorMessage"
	errorTypeKey       = "errorType"
	errorStackTraceKey = "errorStackTrace"
)

type (
	EnrichSlogHandler struct {
		handlers []slog.Handler
		minLevel slog.Level
	}

	stackTrace []uintptr
)

func (s stackTrace) LogValue() slog.Value {
	return slog.GroupValue()
}

func NewEnrichSlogHandler(handlers ...slog.Handler) *EnrichSlogHandler {
	hs := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			hs = append(hs, h)
		}
	}

	return &EnrichSlogHandler{handlers: hs}
}

func (h *EnrichSlogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, dst := range h.handlers {
		if dst.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h *EnrichSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	r2 := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	var firstErr error
	r.Attrs(func(a slog.Attr) bool {
		v := a.Value.Any()
		err, ok := v.(error)
		if ok && err != nil && firstErr == nil {
			firstErr = err
		} else {
			r2.AddAttrs(a)
		}

		return true
	})

	if firstErr == nil {
		return h.callNext(ctx, r)
	}

	r2.AddAttrs(slog.String(errorIdKey, h.generateErrorId()))

	var curRichErr *rich_error.Error
	tmp := firstErr
	var resultMsg = firstErr.Error()
	for {
		if !rich_error.As(tmp, &curRichErr) {
			break
		}

		if resultMsg == "" {
			resultMsg = tmp.Error()
		}
		r2.AddAttrs(curRichErr.Attrs()...)

		tmp = curRichErr.Unwrap()
		if tmp == nil {
			break
		}
	}

	r2.AddAttrs(slog.String(errorMessageKey, resultMsg))

	if curRichErr != nil {
		kind := "<nil>"
		if curRichErr.Kind != nil {
			kind = curRichErr.Kind.String()
		}
		code := "<nil>"
		if curRichErr.Code != nil {
			code = curRichErr.Code.String()
		}
		r2.AddAttrs(
			slog.String(errorTypeKey, fmt.Sprintf("%s:%s", kind, code)),
			slog.Any(errorStackTraceKey, stackTrace(curRichErr.Stacktrace)),
		)
	}

	return h.callNext(ctx, r2)
}

func (h *EnrichSlogHandler) callNext(ctx context.Context, r slog.Record) error {
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

func (h *EnrichSlogHandler) generateErrorId() string {
	id := uuid.New()
	return hex.EncodeToString(id[:])
}

func (h *EnrichSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(h.handlers) == 0 {
		return h
	}
	hs := make([]slog.Handler, 0, len(h.handlers))
	for _, dst := range h.handlers {
		hs = append(hs, dst.WithAttrs(attrs))
	}

	return &EnrichSlogHandler{handlers: hs}
}

func (h *EnrichSlogHandler) WithGroup(name string) slog.Handler {
	if len(h.handlers) == 0 {
		return h
	}
	hs := make([]slog.Handler, 0, len(h.handlers))
	for _, dst := range h.handlers {
		hs = append(hs, dst.WithGroup(name))
	}

	return &EnrichSlogHandler{handlers: hs}
}
