// Package jsonlog the package used for loggging
package jsonlog

import (
	"context"
	"errors"
	"game-scouter-api/internal/data"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
)

type CustomHandler struct {
	slog.Handler
}

func (h CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add stack trace if it's an error level
	if r.Level >= slog.LevelError {
		stack := string(debug.Stack())
		r.AddAttrs(slog.String("stack", stack))
	}

	return h.Handler.Handle(ctx, r)
}

func New(w io.Writer) *slog.Logger {
	baseH := slog.NewJSONHandler(w, &slog.HandlerOptions{})
	customH := CustomHandler{baseH}
	logger := slog.New(customH)
	return logger
}

var logKey = data.ContextKey("logger")

func SetLogger(r *http.Request, logger *slog.Logger) *http.Request {
	if logger == nil {
		panic("user is nill or token is empty")
	}
	ctx := context.WithValue(r.Context(), logKey, logger)
	req := r.WithContext(ctx)
	return req
}

var (
	ErrLoggerNil = errors.New("logger is nil")
	ErrNoLogger  = errors.New("logger does not exist")
)

func GetLogger(ctx context.Context) (*slog.Logger, error) {
	logger, ok := ctx.Value(logKey).(*slog.Logger)
	if !ok {
		return nil, ErrNoLogger
	}
	if logger == nil {
		return nil, ErrLoggerNil
	}
	return logger, nil
}
