// Package jsonlog the package used for loggging
package jsonlog

import (
	"context"
	"io"
	"log/slog"
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
