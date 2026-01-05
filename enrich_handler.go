package errors

import (
	"context"
	"log/slog"
)

const errorSummaryKey = "summaryError"

type (
	EnrichHandler struct {
		next     slog.Handler
		minLevel slog.Level
	}

	errorSummary struct {
		Kind     int
		Code     int
		Message  string
		Meta     []slog.Attr
		StackPCs []uintptr
	}
)

type Option func(*EnrichHandler)

func WithMinLevel(lvl slog.Level) Option {
	return func(h *EnrichHandler) { h.minLevel = lvl }
}

func WrapWithEnrichHandler(next slog.Handler, opts ...Option) EnrichHandler {
	h := &EnrichHandler{
		next: next,
	}

	for _, opt := range opts {
		opt(h)
	}
	return *h
}

func (h *EnrichHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.next.Enabled(ctx, lvl)
}

func (h *EnrichHandler) Handle(ctx context.Context, r slog.Record) error {
	firstErr := h.findFirstError(&r)
	if firstErr == nil {
		return h.next.Handle(ctx, r)
	}

	attrs := make([]slog.Attr, 0, 10)
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
		attrs = append(attrs, lastMeta.LogAttrs()...)

		tmp = lastMeta.Unwrap()
		if tmp == nil {
			break
		}
	}

	r2 := r.Clone()

	var summary *errorSummary
	if lastMeta == nil {
		summary = &errorSummary{Message: firstErr.Error()}
	} else {
		summary = &errorSummary{
			Kind:     lastMeta.Kind,
			Code:     lastMeta.Code,
			Message:  firstErr.Error(),
			Meta:     attrs,
			StackPCs: lastMeta.Stacktrace,
		}
	}

	r2.AddAttrs(slog.Any(errorSummaryKey, summary))

	return h.next.Handle(ctx, r2)
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

func (h *EnrichHandler) WithAttrs(as []slog.Attr) slog.Handler {
	cp := *h
	cp.next = h.next.WithAttrs(as)
	return &cp
}

func (h *EnrichHandler) WithGroup(name string) slog.Handler {
	cp := *h
	cp.next = h.next.WithGroup(name)
	return &cp
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
