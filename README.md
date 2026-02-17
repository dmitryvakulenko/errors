[English](README.md) \| [Русский](README.ru.md)

# errors --- rich errors for Go

The library provides "rich errors" --- an extension of the standard Go
error model that adds:
-   classification (Kind/Code)
-   structured metadata
-   call stack
-   integration with `slog` and `Sentry`
-   compatible with standard `errors.Is` / `errors.As` (via `Unwrap`)
-   able to wrap any errors implementing `error`

The handler package:
-   contains `slog.Handler` that automatically enriches log records with
    data from the error chain
-   contains a converter for Sentry that builds `sentry.Event` based on
    fields from `slog.Record`

## Installation

``` bash
go get github.com/dmitryvakulenko/errors
```

## Quick start

### Creating a rich error

``` go
import (
  "log/slog"
  "github.com/dmitryvakulenko/errors/rich_error"
)

type Kind string
func (k Kind) String() string { return string(k) }

type Code string
func (c Code) String() string { return string(c) }

const (
  KindValidation Kind = "validation"
  CodeBadEmail   Code = "bad_email"
)

func validate(email string) error {
  if email == "" {
    return rich_error.New(
      KindValidation,
      CodeBadEmail,
      "email is empty",
      slog.String("field", "email"),
    )
  }
  return nil
}
```

### Wrapping an error

-   Wrap --- wraps an error and adds metadata (without stacktrace).
-   WrapWithStack --- wraps + captures stacktrace at wrapping point.
    Intended for wrapping standard errors.
-   WrapMeta / WrapMetaWithStack --- adds metadata (+/- stack) without
    kind/code/message. These functions are intended only for adding
    metadata to an error. It is assumed that a `rich_error.Error` with
    kind/code was already created at lower levels.

``` go
err := doSomething()
if err != nil {
  return rich_error.WrapWithStack(
    err,
    Kind("external"),
    Code("timeout"),
    "partner request failed",
    slog.String("partner", "X"),
    slog.Int("attempt", 2),
  )
}
```

### Working with standard errors.Is / errors.As

``` go
if rich_error.Is(err, context.DeadlineExceeded) { ... }

var re *rich_error.Error
if rich_error.As(err, &re) {
  // re.Kind, re.Code, re.Meta, re.Stacktrace ...
}
```

## Integration with slog

The handler package contains EnrichSlogHandler --- a composable handler
that converts `rich_error.Error` into `slog.Record`.

Algorithm for enriching slog.Record:

-   an attribute named `errorId` with generated uuid (hex
    representation) is added
-   the **first** error (implementing standard error interface) is
    searched in attributes
-   the message from this error (`err.Error()`), if present, is added
    into slog.Record attributes under key `errorMessage`
-   the found error is unwrapped via Unwrap and attributes from all
    encountered `rich_error.Error` are copied into slog.Record
-   the **last** found `rich_error.Error` in the chain (i.e., the
    innermost error) is considered primary and used to generate
    attributes:
    -   `errorType` --- composed as "Kind:Code"
    -   `errorStackTrace` --- call stack

### Connection example

``` go
import (
  "log/slog"
  "os"

  "github.com/dmitryvakulenko/errors/handler"
)

func main() {
  json := slog.NewJSONHandler(os.Stdout, nil)

  h := handler.NewEnrichSlogHandler(json)

  logger := slog.New(h)
  slog.SetDefault(logger)

  // example:
  err := someFunc()
  slog.Error("request failed", slog.Any("err", err))
}
```

## Integration with Sentry

Implementation is currently minimal. handler.SentryConverter is intended
for integration with `github.com/getsentry/sentry-go/slog` and creates a
**sentry.Event** based on slog.Record:

-   level → evt.Level
-   message → evt.Message
-   errorId → evt.EventID
-   errorMessage → Exception.Value
-   errorType → Exception.Type
-   errorStackTrace → Exception.Stacktrace
-   remaining attributes → evt.Extra

### Usage example

``` go
package main

import (
    "log/slog"
    "os"
    "time"
    "github.com/getsentry/sentry-go"
    sentryslog "github.com/getsentry/sentry-go/slog"
    "github.com/dmitryvakulenko/errors/handler"
)

func main() {
    err := sentry.Init(sentry.ClientOptions{
        Dsn: "https://your-dsn@sentry.io/...",
    })
    if err != nil {
        panic(err)
    }
    defer sentry.Flush(2 * time.Second)

    sentryHandler := sentryslog.NewSentryHandler(
        sentryslog.Options{Level: slog.LevelError},
        handler.SentryConverter,
    )

    baseHandler := slog.NewTextHandler(os.Stdout, nil)
    enrichHandler := handler.NewEnrichSlogHandler(sentryHandler, baseHandler)

    logger := slog.New(enrichHandler)
    slog.SetDefault(logger)

    slog.Error("request failed", slog.Any("err", err))
}
```
