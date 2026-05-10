package rich_error

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

	if e.Error() != "[1:2] test error" {
		t.Errorf("Expected full error message to be '[1:2] test error', got '%s'", e.Error())
	}

	if len(e.Attributes) != 1 {
		t.Errorf("Expected Metadata to have 1 entry, got %d", len(e.Attributes))
	}

	if e.Attributes[0].String() != "key=value" {
		t.Errorf("Expected logging attribuge 'key' to be 'value', got '%v'", e.Attributes[0].String())
	}

	if len(e.Stacktrace) != 3 {
		t.Errorf("Expected Stack should have exactly 3 entries (including testing runtime), got %d", len(e.Stacktrace))
	}

	frames := runtime.CallersFrames(e.Stacktrace)
	frame, _ := frames.Next()

	expectdFn := "github.com/dmitryvakulenko/errors/rich_error.TestNewError"
	if frame.Function != expectdFn {
		t.Errorf("Wrong stack - unknown function '%s', expected - '%s'", frame.Function, expectdFn)
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

func TestKindOf(t *testing.T) {
	err := New(StrStringer("kind"), StrStringer("code"), "message")
	err2 := fmt.Errorf("error: %w", err)

	if !KindOf(err2, StrStringer("kind")) {
		t.Errorf("Error should be kind of")
	}

	if KindOf(err2, ByteStringer(1)) {
		t.Errorf("Error should not be kind of")
	}
}

func TestSquash(t *testing.T) {
	kind := StrStringer("kind")
	code := StrStringer("code")
	err := New(kind, code, "message", slog.String("key", "value"))
	err2 := fmt.Errorf("error: %w", err)
	err3 := WrapMeta(err2, slog.Int("user_id", 123))

	resErr := Squash(err3)

	if resErr.Kind != kind {
		t.Errorf("Wrong kind: %s. Expected %s.", resErr.Kind, kind)
	}

	if resErr.Code != code {
		t.Errorf("Wrong code: %s. Expected %s.", resErr.Code, code)
	}

	if len(resErr.Attributes) != 2 {
		t.Errorf("Wrong number of attributes: %d. Expected 2.", len(resErr.Attributes))
	}

	if len(resErr.Stacktrace) != 3 {
		t.Errorf("Expected Stack should have exactly 3 entries (including testing runtime), got %d", len(resErr.Stacktrace))
	}

	expMsg := "error: [kind:code] message"
	if err3.Error() != expMsg {
		t.Errorf("Wrong error message: '%s'. Expected '%s'.", err3.Error(), expMsg)
	}
}
