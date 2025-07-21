package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *serverApplication) run() error {
	server := http.Server{
		Addr:         fmt.Sprintf(":%v", app.Cfg.Port),
		Handler:      app.routes(),
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
		WriteTimeout: 30 * time.Second,
	}
	shutDown := make(chan error)
	go func() {
		sig := make(chan os.Signal, 1) // Need buffered cause os wont wait
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		s := <-sig
		app.Logger.Info("Shutting down Server", slog.String("signal", s.String()))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			shutDown <- err
		}
		app.Logger.Info("waiting for background processes")
		app.BackgroundWG.Wait()
		shutDown <- nil
	}()
	app.Logger.Info("Starting server ",
		slog.String("Addr", fmt.Sprintf(":%v", app.Cfg.Port)),
		slog.String("Env", app.Cfg.Env),
	)

	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	err = <-shutDown
	if err != nil {
		return err
	}
	app.Logger.Info("stopped server",
		slog.String("Addr", fmt.Sprintf(":%v", app.Cfg.Port)),
		slog.String("Env", app.Cfg.Env),
	)
	return nil
}
