package errors

import (
	stdErr "errors"
	"fmt"
	"log/slog"
	"runtime"
)

type Error struct {
	Kind     int
	Code     int
	Message  string
	LogAttrs []slog.Attr
	Stack    []uintptr
	Previous error
}

func Is(err, target error) bool {
	return stdErr.Is(err, target)
}

func As(err error, target any) bool {
	return stdErr.As(err, target)
}

func WrapWithStack(err error, kind, code int, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Stack:    buildStack(),
		LogAttrs: attrs,
		Previous: err,
	}

	return res
}

func Wrap(err error, kind, code int, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		LogAttrs: attrs,
		Previous: err,
	}

	return res
}

func New(kind, code int, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Stack:    buildStack(),
		LogAttrs: attrs,
	}

	return res
}

func buildStack() []uintptr {
	stackFrames := make([]uintptr, 32)
	n := runtime.Callers(3, stackFrames)
	return stackFrames[:n]
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s [%d:%d]", e.Message, e.Kind, e.Code)
}

func (e *Error) Unwrap() error {
	return e.Previous
}
