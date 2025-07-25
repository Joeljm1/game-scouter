package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"game-scouter-api/internal/application"
	customrespwriter "game-scouter-api/internal/customRespWriter"

	"golang.org/x/time/rate"
)

func (app *serverApplication) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// m := httpsnoop.CaptureMetrics(next, w, r)
		newW := customrespwriter.New(w)
		next.ServeHTTP(newW, r)
		app.Logger.Info("API call",
			slog.String("Path", r.URL.String()),
			slog.String("Method", r.Method),
			slog.String("Duration", time.Since(newW.StartTime).String()),
			slog.Int("RespCode", newW.Status),
		)
	})
}

func (app *serverApplication) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.ServerErrResponse(w, r, fmt.Errorf("%s", err))
				app.Logger.Error("Panic", slog.Any("Error", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *serverApplication) RateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter     *rate.Limiter
		lastAccesed time.Time
	}
	var mu sync.Mutex
	clients := make(map[string]*client)
	go func() {
		mu.Lock()
		for ip, client := range clients {
			if time.Since(client.lastAccesed) > time.Minute*3 {
				delete(clients, ip)
			}
		}
		mu.Unlock()
		time.Sleep(time.Minute)
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.Cfg.Limiter.Enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.ServerErrResponse(w, r, err)
				return
			}
			mu.Lock()
			if _, ok := clients[ip]; !ok {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.Cfg.Limiter.Rps), app.Cfg.Limiter.Burst),
				}
			}
			clients[ip].lastAccesed = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.RateLimitExceededResponse(w, r)
				return
			}
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

func (app *serverApplication) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Orgin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		orgin := r.Header.Get("Orgin")
		if orgin != "" && len(app.Cfg.Cors.TrustedOrgins) != 0 {
			if slices.Contains(app.Cfg.Cors.TrustedOrgins, orgin) {
				w.Header().Set("Access-Control-Allow-Orgin", orgin)
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" { // if preflight request
					w.Header().Set("Access-Control-Allow-Methods", "OPTION, PUT, PATCH, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Hi-From-Frontend") // add more if needed
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// NOTE: Used to prevent csrf attack for endpoints that cause changes
func (app *serverApplication) CheckCustomHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			hi := r.Header.Get("Hi-From-Frontend")
			if hi != "true" {
				msg := application.Envelope{"Error": "Forbidden"}
				err := app.WriteJSON(w, http.StatusForbidden, msg, nil)
				if err != nil {
					app.ServerErrResponse(w, r, err)
				}
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
