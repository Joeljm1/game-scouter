package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	customrespwriter "game-scouter-api/internal/customRespWriter"

	"golang.org/x/time/rate"
)

func (app *application) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// m := httpsnoop.CaptureMetrics(next, w, r)
		newW := customrespwriter.New(w)
		next.ServeHTTP(newW, r)
		app.logger.Info("API call",
			slog.String("Path", r.URL.String()),
			slog.String("Method", r.Method),
			slog.String("Duration", time.Since(newW.StartTime).String()),
			slog.Int("RespCode", newW.Status),
		)
	})
}

func (app *application) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrResponse(w, r, fmt.Errorf("%s", err))
				app.logger.Error("Panic", slog.Any("Error", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) RateLimit(next http.Handler) http.Handler {
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
		if app.cfg.limiter.enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrResponse(w, r, err)
				return
			}
			mu.Lock()
			if _, ok := clients[ip]; !ok {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.cfg.limiter.rps), app.cfg.limiter.burst),
				}
			}
			clients[ip].lastAccesed = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Orgin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		orgin := r.Header.Get("Orgin")
		if orgin != "" && len(app.cfg.cors.trustedOrgins) != 0 {
			if slices.Contains(app.cfg.cors.trustedOrgins, orgin) {
				w.Header().Set("Access-Control-Allow-Orgin", orgin)
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" { // if preflight request
					w.Header().Set("Access-Control-Allow-Methods", "OPTION, PUT, PATCH, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // add more if needed
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
