package errors

import (
	stdErr "errors"
	"fmt"
	"runtime"
)

type Error struct {
	Kind     int
	Code     int
	Message  string
	Metadata map[string]any
	Stack    []uintptr
	Previous error
}

func Is(err, target error) bool {
	return stdErr.Is(err, target)
}

func As(err error, target any) bool {
	return stdErr.As(err, target)
}

func WrapWithStack(err error, kind, code int, message string, fields ...any) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Stack:    buildStack(),
		Metadata: buildFields(fields),
		Previous: err,
	}

	return res
}

func Wrap(err error, kind, code int, message string, fields ...any) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Metadata: buildFields(fields),
		Previous: err,
	}

	return res
}

func New(kind, code int, message string, fields ...any) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Stack:    buildStack(),
		Metadata: buildFields(fields),
	}

	return res
}

func buildFields(fields []any) map[string]any {
	var maxIdx = len(fields)
	if maxIdx%2 == 1 {
		maxIdx -= 1
	}

	res := make(map[string]any, len(fields)/2)
	for i := 0; i < maxIdx; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		res[key] = fields[i+1]
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
