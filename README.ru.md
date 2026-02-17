[English](README.md) | [Русский](README.ru.md)

# errors --- rich errors for Go
Библиотека предоставляет "rich errors" — расширение стандартной модели ошибок Go,
которое добавляет:
- классификацию (Kind/Code)
- структурированные метаданные
- стек вызовов
- интеграцию с `slog` и `Sentry`
- совместима со стандартным `errors.Is` / `errors.As` (через `Unwrap`)
- умеет оборачивать любые ошибки, реализующие `error`

Пакет handler
- содержит `slog.Handler`, который автоматически обогащает записи логов данными из цепочки ошибок
- содержит конвертер для Sentry, который формирует `sentry.Event` на основе полей из `slog.Record`

## Установка
```bash
go get github.com/dmitryvakulenko/errors
```
## Quick start
### Создание rich error
```go
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
### Оборачивание ошибки
- Wrap — оборачивает ошибку и добавляет meta (без stacktrace).
- WrapWithStack — оборачивает + снимает stacktrace на месте обёртки. Нужна для оборачивания стандартных ошибок.
- WrapMeta / WrapMetaWithStack — добавляет meta (+/- stack) без kind/code/message. Эти функции нужны только для добавления метаданных к ошибке. Т.е. предполагается, что на нижних уровнях уже была создана `rich_error.Error` с kind/code и здесь они могут быть пустыми.
```go
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
### Работа со стандартным errors.Is / errors.As
Библиотека просто проксирует стандартные функции:
```go
if rich_error.Is(err, context.DeadlineExceeded) { ... }

var re *rich_error.Error
if rich_error.As(err, &re) {
  // re.Kind, re.Code, re.Meta, re.Stacktrace ...
}
```
## Интеграция с slog
Пакет handler содержит EnrichSlogHandler — handler-компоновщик, который преобразует `rich_error.Error` в `slog.Record`.
Алгоритм обогащения slog.Record:
- в атрибуты добавляется атрибут с именем `errorId` и сгенерированным значением uuid в hex-представлении
- ищется **первая** ошибка (реализующая стандартный интерфейс error) в списке атрибутов
- сообщение из этой ошибки (`err.Error()`), если есть, добавляется в атрибуты `slog.Record` с ключом `errorMessage`
- найденная ошибка разворачивается с помощью Unwrap и атрибуты из всех встреченных `rich_error.Error` копируются в `slog.Record`
- **последняя** найденная `rich_error.Error` в цепочке (т.е. самая внутренняя ошибка) считается основной и из неё генерируются следующие атрибуты
  - `errorType` — состоит из "Kind:Code"
  - errorStackTrace — стек вызовов
### Пример подключения
```go
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

  // пример:
  err := someFunc()
  slog.Error("request failed", slog.Any("err", err))
}
```
## Интеграция с Sentry
Реализация пока простейшая. handler.SentryConverter предназначен для интеграции с `github.com/getsentry/sentry-go/slog` и создаёт **sentry.Event** на основе slog.Record:
- уровень → evt.Level
- сообщение → evt.Message
- errorId → evt.EventID
- errorMessage → Exception.Value
- errorType → Exception.Type
- errorStackTrace → Exception.Stacktrace
- остальные атрибуты → evt.Extra
### Пример использования
```go
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
