package handler

import (
	"log/slog"
	"runtime"
	"strings"

	"github.com/getsentry/sentry-go"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

func SentryConverter(
	addSource bool,
	replaceAttr func(groups []string, a slog.Attr) slog.Attr,
	loggerAttr []slog.Attr,
	groups []string,
	record *slog.Record,
	hub *sentry.Hub) *sentry.Event {
	evt := &sentry.Event{
		Level:   sentryslog.LogLevels[record.Level],
		Message: record.Message,
		Extra:   make(map[string]any, len(loggerAttr)+record.NumAttrs()),
	}

	exception := sentry.Exception{}
	record.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case errorIdKey:
			evt.EventID = sentry.EventID(a.Value.String())
		case errorMessageKey:
			exception.Value = a.Value.String()
		case errorTypeKey:
			exception.Type = a.Value.String()
		case errorStackTraceKey:
			exception.Stacktrace = formatStack([]uintptr(a.Value.Any().(stackTrace)))
		default:
			evt.Extra[a.Key] = a.Value.Any()
		}

		return true
	})

	if exception.Value != "" {
		evt.Exception = []sentry.Exception{exception}
	}

	for _, a := range loggerAttr {
		evt.Extra[a.Key] = a.Value.Any()
	}

	return evt
}

func formatStack(pcs []uintptr) *sentry.Stacktrace {
	framesIter := runtime.CallersFrames(pcs)

	frames := make([]sentry.Frame, 0, len(pcs))
	for {
		fr, more := framesIter.Next()

		fn := fr.Function
		mod, fun := splitModuleAndFunc(fn)

		abs := fr.File
		file := abs

		sFrame := sentry.Frame{
			AbsPath:  abs,
			Filename: file,
			Function: fun,
			Module:   mod,
			Lineno:   fr.Line,
		}

		frames = append(frames, sFrame)

		if !more {
			break
		}
	}

	// Reverse to oldest->newest for Sentry.
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}

	return &sentry.Stacktrace{Frames: frames}
}

func splitModuleAndFunc(full string) (module, function string) {
	if full == "" {
		return "", ""
	}

	// Find last '.' which usually separates pkg path from func/method name.
	lastDot := strings.LastIndexByte(full, '.')
	if lastDot <= 0 || lastDot >= len(full)-1 {
		// Fallback: no dot (rare), treat everything as function.
		return "", full
	}

	module = full[:lastDot]
	function = full[lastDot+1:]

	// If function looks like "(*T)" etc. and next part contains another dot,
	// prefer splitting at the *last* dot anyway; already done.
	return module, function
}
