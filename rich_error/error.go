package rich_error

import (
	stdErr "errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
)

type (
	Error struct {
		Kind       fmt.Stringer
		Code       fmt.Stringer
		Message    string
		Attributes []slog.Attr
		Stacktrace []uintptr
		Previous   error
	}

	ChainData struct {
		Kind       fmt.Stringer
		Code       fmt.Stringer
		Attributes []slog.Attr
		Stacktrace []uintptr
	}
)

func Is(err, target error) bool {
	return stdErr.Is(err, target)
}

func As(err error, target any) bool {
	return stdErr.As(err, target)
}

func KindOf(err error, kind any) bool {
	if e, ok := AsType[*Error](err); ok {
		return e.Kind == kind
	} else {
		return false
	}
}

func AsType[E error](err error) (E, bool) {
	return stdErr.AsType[E](err)
}

func WrapWithStack(err error, kind, code fmt.Stringer, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:       kind,
		Code:       code,
		Message:    message,
		Stacktrace: buildStack(),
		Attributes: attrs,
		Previous:   err,
	}

	return res
}

func Wrap(err error, kind, code fmt.Stringer, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:       kind,
		Code:       code,
		Message:    message,
		Attributes: attrs,
		Previous:   err,
	}

	return res
}

func WrapMeta(err error, attrs ...slog.Attr) *Error {
	res := &Error{
		Attributes: attrs,
		Previous:   err,
	}

	return res
}

func WrapMetaWithStack(err error, attrs ...slog.Attr) *Error {
	res := &Error{
		Stacktrace: buildStack(),
		Attributes: attrs,
		Previous:   err,
	}

	return res
}

func New(kind, code fmt.Stringer, message string, attrs ...slog.Attr) *Error {
	res := &Error{
		Kind:       kind,
		Code:       code,
		Message:    message,
		Stacktrace: buildStack(),
		Attributes: attrs,
	}

	return res
}

func buildStack() []uintptr {
	stackFrames := make([]uintptr, 32)
	n := runtime.Callers(3, stackFrames)
	return stackFrames[:n]
}

func (e *Error) Error() string {
	msg := strings.Builder{}

	if e.Kind != nil || e.Code != nil {
		msg.WriteString(fmt.Sprintf("[%s:%s] ", e.Kind, e.Code))
	}

	if e.Message != "" {
		msg.WriteString(e.Message)
	}

	if e.Previous != nil {
		if msg.Len() > 0 {
			msg.WriteString(": ")
		}

		msg.WriteString(e.Previous.Error())
	}

	return msg.String()
}

func (e *Error) Unwrap() error {
	return e.Previous
}

// Squash collects all error attributes from the chain into ChainData.
// It uses Kind, Code, and Stacktrace from the last Error in the chain.
// If no rich_errors in the chain all fields will be empty.
func Squash(err error) ChainData {
	curErr, ok := AsType[*Error](err)
	if !ok {
		return ChainData{}
	}

	prevErr := curErr
	resAttrs := curErr.Attributes
	for curErr.Previous != nil {
		curErr, ok = AsType[*Error](curErr.Previous)
		if !ok {
			break
		}
		resAttrs = append(resAttrs, curErr.Attributes...)
		prevErr = curErr
	}

	return ChainData{
		Kind:       prevErr.Kind,
		Code:       prevErr.Code,
		Attributes: resAttrs,
		Stacktrace: prevErr.Stacktrace,
	}
}
