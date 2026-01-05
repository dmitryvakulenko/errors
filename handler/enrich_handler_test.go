package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	errors2 "github.com/dmitryvakulenko/errors"
)

type (
	testHandler struct {
		Rec slog.Record
	}

	expectedErrorData struct {
		Type       string
		Message    string
		Meta       []slog.Attr
		StackPCs   []uintptr
		TotalAttrs int
	}

	testKind int
	testCode int
)

const (
	kind1 testKind = iota + 1
	kind2
)

const (
	code1 testCode = iota + 1
	code2
)

func (t testKind) String() string {
	switch t {
	case kind1:
		return "kind1"
	case kind2:
		return "kind2"
	default:
		return "unknown"
	}
}

func (t testCode) String() string {
	switch t {
	case code1:
		return "code1"
	case code2:
		return "code2"
	default:
		return "unknown"
	}
}

func (t *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (t *testHandler) Handle(ctx context.Context, record slog.Record) error {
	t.Rec = record
	return nil
}

func (t *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return t
}

func (t *testHandler) WithGroup(name string) slog.Handler {
	return t
}

func TestAll(t *testing.T) {
	testData := []struct {
		Name          string
		Err           error
		ExpectedError *expectedErrorData
	}{
		{
			Name: "Standard error",
			Err:  errors.New("standard error"),
			ExpectedError: &expectedErrorData{
				Message:    "standard error",
				TotalAttrs: 2,
			},
		},
		{
			Name: "Only custom error no attributes",
			Err:  errors2.New(kind1, code1, "hello"),
			ExpectedError: &expectedErrorData{
				Type:       "kind1:code1",
				Message:    "hello",
				StackPCs:   make([]uintptr, 3),
				TotalAttrs: 4,
			},
		},
		{
			Name: "Wrapped custom error by one layer",
			Err:  fmt.Errorf("got error: %w", errors2.New(kind1, code1, "hello")),
			ExpectedError: &expectedErrorData{
				Type:       "kind1:code1",
				Message:    "got error: hello",
				StackPCs:   make([]uintptr, 3),
				TotalAttrs: 4,
			},
		},
		{
			Name: "Multiple wrapped custom error with metadata",
			Err: fmt.Errorf("full error: %w",
				errors2.Wrap(
					fmt.Errorf(
						"got error: %w",
						errors2.New(kind1, code1, "hello", slog.Int("code", 1), slog.Int("code2", 2)),
					),
					kind2,
					code2,
					"aaa",
					slog.Int("request_id", 2),
				),
			),
			ExpectedError: &expectedErrorData{
				Type:     "kind1:code1",
				Message:  "full error: aaa",
				StackPCs: make([]uintptr, 3),
				Meta: []slog.Attr{
					slog.Int("request_id", 2),
					slog.Int("code", 1),
					slog.Int("code2", 2),
				},
				TotalAttrs: 7,
			},
		},
	}

	for _, d := range testData {
		t.Run(d.Name, func(t *testing.T) {
			stub := &testHandler{}
			h := NewEnrichHandler(stub)
			logger := slog.New(h)
			logger.Info("test", "error", d.Err)

			hasId := false
			stub.Rec.Attrs(func(a slog.Attr) bool {
				switch a.Key {
				case errorIdKey:
					val := a.Value.String()
					if len(val) != 0 {
						hasId = true
					}
				case errorMessageKey:
					val := a.Value.String()
					if d.ExpectedError.Message != val {
						t.Errorf("Message mismatch. Expected '%s', real '%s'", d.ExpectedError.Message, val)
					}
				case errorTypeKey:
					val := a.Value.String()
					if d.ExpectedError.Type != val {
						t.Errorf("Error type mismatch. Expected '%s', real '%s'", d.ExpectedError.Type, val)
					}
				case errorStackTraceKey:
					stack, ok := a.Value.Any().(stackTrace)
					if !ok {
						t.Errorf("Wrong stack type")
					}
					if len(d.ExpectedError.StackPCs) != len(stack) {
						t.Errorf("Stack length mismatch. Expected %d, got %d", len(d.ExpectedError.StackPCs), len(stack))
					}
				default:
					for _, m := range d.ExpectedError.Meta {
						if m.Key == a.Key && !m.Equal(a) {
							t.Errorf(
								"Attribute %s mismatch. %v != %v",
								m.Key,
								m.Value,
								a.Value,
							)
						}
					}
				}

				return true
			})

			if !hasId {
				t.Errorf("Error has no id")
			}

			if d.ExpectedError.TotalAttrs != stub.Rec.NumAttrs() {
				t.Errorf("Wrong attributes amount. Expected %d, got %d", d.ExpectedError.TotalAttrs, stub.Rec.NumAttrs())
			}
		})
	}
}
