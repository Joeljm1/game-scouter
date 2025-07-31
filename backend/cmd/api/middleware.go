package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"game-scouter-api/internal/application"
	customrespwriter "game-scouter-api/internal/customRespWriter"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/validator"

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
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		w.Header().Add("Vary", "Access-Control-Request-Headers")
		origin := r.Header.Get("Origin")
		if origin != "" && len(app.Cfg.Cors.TrustedOrgins) != 0 {
			if slices.Contains(app.Cfg.Cors.TrustedOrgins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" { // if preflight request
					w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Hi-From-Frontend") // add more if needed
					w.WriteHeader(http.StatusNoContent)
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

func (app *serverApplication) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Cookie")
		cookie, err := r.Cookie(app.Cfg.SessionCookie)
		if err != nil {
			//TODO: cookie not pressent so make a session
			cookie, tok, err := app.AnonUserCookie()
			if err != nil {
				app.ServerErrResponse(w, r, err)
				return
			}
			http.SetCookie(w, cookie)
			req := app.SetUser(r, data.AnonymousUser(), tok.Plaintext)
			next.ServeHTTP(w, req)
			return
		}
		token := cookie.Value
		v := validator.NewValidator()
		data.ValidateToken(v, token)
		if !v.Valid() {
			app.ValidationErrResponse(w, r, v.Errors)
			return
		}
		user, err := app.Models.UserModel.GetUserfromToken(token, data.ScopeAuthentication)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRows):
				v.AddError("token", "Invalid authentication token found")
				app.ValidationErrResponse(w, r, v.Errors)
			default:
				app.ServerErrResponse(w, r, err)
			}
			return
		}
		if user.ID == 0 {
			user = data.AnonymousUser()
		}
		req := app.SetUser(r, user, token)
		next.ServeHTTP(w, req)
	})
}

func (app *serverApplication) reqAuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.GetUser(r)
		if user.IsAnonymous() {
			app.NotAuthenticatedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Will check if user is authenticated to. No need to double check it
func (app *serverApplication) reqActivatedUser(next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.GetUser(r)
		if !user.Activated {
			app.NotActivatedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.reqAuthUser(fn)
}
