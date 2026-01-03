package errors

import (
	"runtime"
	"testing"
)

func TestNewError(t *testing.T) {
	e := New(1, 2, "test error", "key", "value")
	if e.Kind != 1 {
		t.Errorf("Expected Kind to be 1, got %d", e.Kind)
	}

	if e.Code != 2 {
		t.Errorf("Expected Code to be 2, got %d", e.Code)
	}

	if e.Message != "test error" {
		t.Errorf("Expected Message to be 'test error', got '%s'", e.Message)
	}

	if e.Error() != "test error [1:2]" {
		t.Errorf("Expected full error message to be 'test error [1:2]', got '%s'", e.Error())
	}

	if len(e.Metadata) != 1 {
		t.Errorf("Expected Metadata to have 1 entry, got %d", len(e.Metadata))
	}

	if e.Metadata["key"] != "value" {
		t.Errorf("Expected Metadata['key'] to be 'value', got '%v'", e.Metadata["key"])
	}

	if len(e.Stack) != 3 {
		t.Errorf("Expected Stack should have exactly 3 entries (including testing runtime), got %d", len(e.Stack))
	}

	frames := runtime.CallersFrames(e.Stack)
	frame, _ := frames.Next()

	if frame.Function != "dmitryvakulenko/errors.TestNewError" {
		t.Errorf("Wrong stack - unknown function '%s'", frame.Function)
	}
}
