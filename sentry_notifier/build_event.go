package sentry_notifier

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/dmitryvakulenko/errors/rich_error"
	"github.com/getsentry/sentry-go"
)

func buildEvent(inError error) *sentry.Event {
	errData := rich_error.Squash(inError)

	evt := &sentry.Event{
		Level:   sentry.LevelError,
		Message: inError.Error(),
		Extra:   make(map[string]any, len(errData.Attributes)),
	}

	exception := sentry.Exception{
		Type:  fmt.Sprintf("[%s:%s]", errData.Kind.String(), errData.Code.String()),
		Value: inError.Error(),
	}

	if len(errData.Stacktrace) > 0 {
		exception.Stacktrace = formatStack(errData.Stacktrace)
	}

	for _, a := range errData.Attributes {
		evt.Extra[a.Key] = a.Value.Any()
	}

	if exception.Value != "" {
		evt.Exception = []sentry.Exception{exception}
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
