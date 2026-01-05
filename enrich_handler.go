package errors

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

const (
	errorIdKey         = "errorId"
	errorMessageKey    = "errorMessage"
	errorTypeKey       = "errorType"
	errorStackTraceKey = "errorStackTrace"
)

type (
	EnrichHandler struct {
		handlers []slog.Handler
		minLevel slog.Level
	}

	stackTrace []uintptr
)

func (s stackTrace) LogValue() slog.Value {
	return slog.GroupValue()
}

func NewWithEnrichHandler(handlers []slog.Handler) *EnrichHandler {
	hs := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			hs = append(hs, h)
		}
	}

	return &EnrichHandler{handlers: hs}
}

func (h *EnrichHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, dst := range h.handlers {
		if dst.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h *EnrichHandler) Handle(ctx context.Context, r slog.Record) error {
	firstErr := h.findFirstError(&r)
	if firstErr == nil {
		return h.callNext(ctx, r)
	}

	r2 := r.Clone()
	r2.AddAttrs(slog.String(errorIdKey, h.generateErrorId()))

	var lastMeta *Error
	tmp := firstErr
	var resultMsg = firstErr.Error()
	for {
		if !As(tmp, &lastMeta) {
			break
		}

		if resultMsg != "" {
			resultMsg = tmp.Error()
		}
		r2.AddAttrs(lastMeta.LogAttrs()...)

		tmp = lastMeta.Unwrap()
		if tmp == nil {
			break
		}
	}

	slog.String(errorMessageKey, resultMsg)

	if lastMeta != nil {
		r2.AddAttrs(
			slog.String(errorTypeKey, fmt.Sprintf("%s:%s", lastMeta.Kind.String(), lastMeta.Code.String())),
			slog.Any(errorStackTraceKey, stackTrace(lastMeta.Stacktrace)),
		)
	}

	return h.callNext(ctx, r2)
}

func (h *EnrichHandler) callNext(ctx context.Context, r slog.Record) error {
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

func (h *EnrichHandler) generateErrorId() string {
	id := uuid.New()
	return hex.EncodeToString(id[:])
}

func (h *EnrichHandler) findFirstError(r *slog.Record) error {
	var res error
	r.Attrs(func(a slog.Attr) bool {
		v := a.Value.Any()
		err, ok := v.(error)
		if !ok || err == nil {
			return true
		}

		res = err

		return false
	})

	return res
}

func (h *EnrichHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(h.handlers) == 0 {
		return h
	}
	hs := make([]slog.Handler, 0, len(h.handlers))
	for _, dst := range h.handlers {
		hs = append(hs, dst.WithAttrs(attrs))
	}

	return &EnrichHandler{handlers: hs}
}

func (h *EnrichHandler) WithGroup(name string) slog.Handler {
	if len(h.handlers) == 0 {
		return h
	}
	hs := make([]slog.Handler, 0, len(h.handlers))
	for _, dst := range h.handlers {
		hs = append(hs, dst.WithGroup(name))
	}

	return &EnrichHandler{handlers: hs}
}

//func collectMetaErrors(err error) []MetaError {
//	var out []MetaError
//
//	seen := make(map[error]struct{}, 8) // защита от циклов (на всякий)
//	var dfs func(e error, depth int)
//	dfs = func(e error, depth int) {
//		if e == nil {
//			return
//		}
//		if _, ok := seen[e]; ok {
//			return
//		}
//		seen[e] = struct{}{}
//
//		if thisMe, ok := e.(MetaError); ok {
//			out = append(out, thisMe)
//		}
//
//		// Multi-error (Go 1.20+): Unwrap() []error
//		type unwrapperMany interface{ Unwrap() []error }
//		if m, ok := e.(unwrapperMany); ok {
//			for _, child := range m.Unwrap() {
//				dfs(child, depth+1)
//			}
//			return
//		}
//
//		// Single unwrap: Unwrap() error
//		if u := errors.Unwrap(e); u != nil {
//			dfs(u, depth+1)
//		}
//	}
//
//	dfs(err, 0)
//
//	return out
//}

//func formatStack(pcs []uintptr) *sentry.Stacktrace {
//	framesIter := runtime.CallersFrames(pcs)
//
//	frames := make([]sentry.Frame, 0, len(pcs))
//	for {
//		fr, more := framesIter.Next()
//
//		fn := fr.Function
//		mod, fun := splitModuleAndFunc(fn)
//
//		abs := fr.File
//		file := abs
//
//		sFrame := sentry.Frame{
//			AbsPath:  abs,
//			Filename: file,
//			Function: fun,
//			Module:   mod,
//			Lineno:   fr.Line,
//		}
//
//		frames = append(frames, sFrame)
//
//		if !more {
//			break
//		}
//	}
//
//	// Reverse to oldest->newest for Sentry.
//	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
//		frames[i], frames[j] = frames[j], frames[i]
//	}
//
//	return &sentry.Stacktrace{Frames: frames}
//}

// splitModuleAndFunc tries to convert runtime frame.Function into Sentry-ish Module + Function.
//
// runtime.Frame.Function examples:
//
//	"github.com/acme/proj/pkg/service.(*Svc).Do"
//	"main.main"
//	"net/http.(*conn).serve"
//
// Heuristics used here:
// - Module: everything before the last '.' that starts the function/method name
// - Function: the last segment after the module's last '.'
//func splitModuleAndFunc(full string) (module, function string) {
//	if full == "" {
//		return "", ""
//	}
//
//	// Find last '.' which usually separates pkg path from func/method name.
//	lastDot := strings.LastIndexByte(full, '.')
//	if lastDot <= 0 || lastDot >= len(full)-1 {
//		// Fallback: no dot (rare), treat everything as function.
//		return "", full
//	}
//
//	module = full[:lastDot]
//	function = full[lastDot+1:]
//
//	// If function looks like "(*T)" etc. and next part contains another dot,
//	// prefer splitting at the *last* dot anyway; already done.
//	return module, function
//}
