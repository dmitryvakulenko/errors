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

func New(kind, code int, message string, fields ...any) *Error {
	res := &Error{
		Kind:     kind,
		Code:     code,
		Message:  message,
		Stack:    buildStack(),
		Metadata: make(map[string]any, len(fields)/2),
	}

	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		res.Metadata[key] = fields[i+1]
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
