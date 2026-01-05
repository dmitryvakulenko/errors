package errors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
)

type testHandler struct {
	Rec slog.Record
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
		ExpectedError *errorSummary
	}{
		{
			Name: "Standard error",
			Err:  errors.New("standard error"),
			ExpectedError: &errorSummary{
				Message: "standard error",
			},
		},
		{
			Name: "Only custom error no attributes",
			Err:  New(1, 2, "hello"),
			ExpectedError: &errorSummary{
				Kind:     1,
				Code:     2,
				Message:  "hello",
				StackPCs: make([]uintptr, 3),
			},
		},
		{
			Name: "Wrapped custom error by one layer",
			Err:  fmt.Errorf("got error: %w", New(1, 2, "hello")),
			ExpectedError: &errorSummary{
				Kind:     1,
				Code:     2,
				Message:  "got error: hello",
				StackPCs: make([]uintptr, 3),
			},
		},
		{
			Name: "Multiple wrapped custom error with metadata",
			Err: fmt.Errorf("full error: %w",
				Wrap(
					fmt.Errorf(
						"got error: %w",
						New(1, 2, "hello", slog.Int("code", 1), slog.Int("code2", 2)),
					),
					3,
					4,
					"aaa",
					slog.Int("request_id", 2),
				),
			),
			ExpectedError: &errorSummary{
				Kind:     1,
				Code:     2,
				Message:  "full error: aaa",
				StackPCs: make([]uintptr, 3),
				Meta: []slog.Attr{
					slog.Int("request_id", 2),
					slog.Int("code", 1),
					slog.Int("code2", 2),
				},
			},
		},
	}

	for _, d := range testData {
		t.Run(d.Name, func(t *testing.T) {
			stub := &testHandler{}
			h := WrapWithEnrichHandler(stub)
			logger := slog.New(&h)
			logger.Info("test", "error", d.Err)

			var ok bool
			var summary *errorSummary
			stub.Rec.Attrs(func(a slog.Attr) bool {
				if a.Key == errorSummaryKey {
					tmp := a.Value.Any()

					summary, ok = tmp.(*errorSummary)
					if !ok {
						return true
					}

					return false
				}
				return true
			})

			if summary == nil {
				t.Errorf("No summary error")
			}

			if d.ExpectedError.Kind != summary.Kind {
				t.Errorf("Kind mismatch. Expected %d, real %d", d.ExpectedError.Kind, summary.Kind)
			}

			if d.ExpectedError.Code != summary.Code {
				t.Errorf("Code mismatch. Expected %d, real %d", d.ExpectedError.Code, summary.Code)
			}

			if d.ExpectedError.Message != summary.Message {
				t.Errorf("Message mismatch. Expected '%s', real '%s'", d.ExpectedError.Message, summary.Message)
			}

			if len(d.ExpectedError.StackPCs) != len(summary.StackPCs) {
				t.Errorf("Stack length mismatch. Expected %d, got %d", len(d.ExpectedError.StackPCs), len(summary.StackPCs))
			}

			if len(d.ExpectedError.Meta) != len(summary.Meta) {
				t.Errorf("Metadata length mismatch. Expected %d, got %d", len(d.ExpectedError.Meta), len(summary.Meta))
			}

			for idx := range d.ExpectedError.Meta {
				if !d.ExpectedError.Meta[idx].Equal(summary.Meta[idx]) {
					t.Errorf(
						"Attribute %s mismatch. %v != %v",
						d.ExpectedError.Meta[idx].Key,
						d.ExpectedError.Meta[idx].Value.Any(),
						summary.Meta[idx].Value.Any(),
					)
				}
			}
		})
	}
}
