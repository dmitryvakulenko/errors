package errors

import (
	stdErr "errors"
	"log/slog"
	"runtime"
)

type Error struct {
	Kind       int
	Code       int
	Message    string
	Meta       []slog.Attr
	Stacktrace []uintptr
	Previous   error
}

func Is(err, target error) bool {
	return stdErr.Is(err, target)
}

func As(err error, target any) bool {
	return stdErr.As(err, target)
}

func WrapWithStack(err error, kind, code int, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:       kind,
		Code:       code,
		Message:    message,
		Stacktrace: buildStack(),
		Meta:       attrs,
		Previous:   err,
	}

	return res
}

func Wrap(err error, kind, code int, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Meta:     attrs,
		Previous: err,
	}

	return res
}

func WrapMeta(err error, attrs ...slog.Attr) *Error {
	res := &Error{
		Meta:     attrs,
		Previous: err,
	}

	return res
}

func WrapMetaWithStack(err error, attrs ...slog.Attr) *Error {
	res := &Error{
		Stacktrace: buildStack(),
		Meta:       attrs,
		Previous:   err,
	}

	return res
}

func New(kind, code int, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:       kind,
		Code:       code,
		Message:    message,
		Stacktrace: buildStack(),
		Meta:       attrs,
	}

	return res
}

func buildStack() []uintptr {
	stackFrames := make([]uintptr, 32)
	n := runtime.Callers(3, stackFrames)
	return stackFrames[:n]
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Previous
}

func (e *Error) LogAttrs() []slog.Attr {
	return e.Meta
}

func (e *Error) Stack() []uintptr {
	return e.Stacktrace
}
