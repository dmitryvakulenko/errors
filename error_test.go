package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"testing"
)

type (
	simpleErr struct {
		Code int
	}

	errKindCode int
)

func (e errKindCode) String() string {
	return fmt.Sprintf("%d", int(e))
}

func (s *simpleErr) Error() string {
	return fmt.Sprintf("%d", s.Code)
}

func TestNewError(t *testing.T) {
	e := New(errKindCode(1), errKindCode(2), "test error", slog.String("key", "value"))
	if e.Kind != errKindCode(1) {
		t.Errorf("Expected Kind to be 1, got %d", e.Kind)
	}

	if e.Code != errKindCode(2) {
		t.Errorf("Expected Code to be 2, got %d", e.Code)
	}

	if e.Message != "test error" {
		t.Errorf("Expected Message to be 'test error', got '%s'", e.Message)
	}

	if e.Error() != "test error" {
		t.Errorf("Expected full error message to be 'test error [1:2]', got '%s'", e.Error())
	}

	if len(e.Meta) != 1 {
		t.Errorf("Expected Metadata to have 1 entry, got %d", len(e.Meta))
	}

	if e.Meta[0].String() != "key=value" {
		t.Errorf("Expected logging attribuge 'key' to be 'value', got '%v'", e.Meta[0].String())
	}

	if len(e.Stacktrace) != 3 {
		t.Errorf("Expected Stack should have exactly 3 entries (including testing runtime), got %d", len(e.Stacktrace))
	}

	frames := runtime.CallersFrames(e.Stacktrace)
	frame, _ := frames.Next()

	if frame.Function != "dmitryvakulenko/errors.TestNewError" {
		t.Errorf("Wrong stack - unknown function '%s'", frame.Function)
	}
}

func TestIs(t *testing.T) {
	err := errors.New("example error")
	e := Wrap(err, errKindCode(1), errKindCode(2), "wrapping")

	if Is(e, err) != true {
		t.Errorf("Error should be same")
	}
}

func TestAs(t *testing.T) {
	err := &simpleErr{Code: 255}
	e := Wrap(err, errKindCode(1), errKindCode(2), "wrapping")

	var tstErr *simpleErr
	if !As(e, &tstErr) {
		t.Errorf("Error should has same type")
	}
}
