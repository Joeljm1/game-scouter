package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"slices"
	"time"

	"game-scouter-api/internal/application"
	customrespwriter "game-scouter-api/internal/customRespWriter"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/jsonlog"
	ratelimiter "game-scouter-api/internal/rateLimter"
	"game-scouter-api/internal/validator"
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

// May be make limiter based on user ID later instead of IP.
//
// WARN: Need to check if this can cause error
// cause its my custom rate limiter.
// Also not sure if the mutex sharding was really needed.
// Also if i attack in mutiple router multiple shard will be created but this is
// used only in the main router so no problem
// if problem later move it a init() or [serverApplication.run] or Configure() fn
func (app *serverApplication) RateLimit(next http.Handler) http.Handler {
	shards := ratelimiter.NewNShards(app.Cfg.Limiter.ShardNo)
	go shards.CleanShardStore(app.Cfg.Ctx)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.Cfg.Limiter.Enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.ServerErrResponse(w, r, err)
				return
			}
			userShard, err := shards.GetShardFromIP(ip)
			if err != nil {
				app.ServerErrResponse(w, r, err)
				return
			}
			userShard.Lock()
			if _, ok := userShard.Clients[ip]; !ok {
				userShard.Clients[ip] = &ratelimiter.Client{
					Limiter: ratelimiter.New(app.Cfg.Limiter.Burst, app.Cfg.Limiter.Rps),
				}
			}
			userShard.Clients[ip].LastAccesed = time.Now()
			if !userShard.Clients[ip].Limiter.Allow() {
				userShard.Unlock()
				app.RateLimitExceededResponse(w, r)
				return
			}
			userShard.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

//	func (app *serverApplication) RateLimit(next http.Handler) http.Handler {
//		var mu sync.Mutex
//		clients := make(map[string]*ratelimiter.Client)
//		go func() {
//			for {
//				mu.Lock()
//				for ip, client := range clients {
//					if time.Since(client.LastAccesed) > time.Minute*3 {
//						delete(clients, ip)
//					}
//				}
//				mu.Unlock()
//				time.Sleep(time.Minute)
//			}
//		}()
//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			if app.Cfg.Limiter.Enabled {
//				ip, _, err := net.SplitHostPort(r.RemoteAddr)
//				if err != nil {
//					app.ServerErrResponse(w, r, err)
//					return
//				}
//				mu.Lock()
//				if _, ok := clients[ip]; !ok {
//					clients[ip] = &ratelimiter.Client{
//						Limiter: ratelimiter.New(app.Cfg.Limiter.Burst, app.Cfg.Limiter.Rps),
//					}
//				}
//				clients[ip].LastAccesed = time.Now()
//				if !clients[ip].Limiter.Allow() {
//					mu.Unlock()
//					app.RateLimitExceededResponse(w, r)
//					return
//				}
//				mu.Unlock()
//			}
//			next.ServeHTTP(w, r)
//		})
//	}

// TODO: check if Access-Control-Max-Age will not cause csrf attack or smthing
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
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Hi-From-Frontend") // add more if needed
					//TODO: change from hardcode to config
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// NOTE: Used to prevent csrf attack for endpoints that cause changes
// as session cookie is of type SameSiteNone mode due to sep frontend and backend
// look into may be checking Sec-Fetch-Site and Orgin headers to for modern browsers like stdlib does it
//
// this method is from [https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html#employing-custom-request-headers-for-ajaxapi]
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

// TODO: save token in context of req
func (app *serverApplication) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// not sure about below
		// w.Header().Add("Cache-Control", `no-cache="Set-Cookie"`)
		w.Header().Add("Vary", "Cookie")
		cookie, err := r.Cookie(app.Cfg.SessionCookie)
		if err != nil {
			cookie, tok, err := app.AnonUserCookie(r.Context())
			if err != nil {
				app.ServerErrResponse(w, r, err)
				return
			}
			http.SetCookie(w, cookie)
			req := app.SetUserDetailsToCtx(r, data.AnonymousUser(), tok.Plaintext, map[string]any{}, data.ScopeUnknown)
			next.ServeHTTP(w, req)
			m, _ := app.WrittenSess(req)
			if m != nil {
				//TODO: may have race cond when same session write to same session togeather in same tab but low priority not sure if needed
				err = app.Models.TokenModel.StoreSessionData(req.Context(), tok.Plaintext, m)
				if err != nil {
					app.LogErr("err", r, err)
				}
			}
			return
		}
		token := cookie.Value
		v := validator.NewValidator()
		data.ValidateToken(v, token)
		if !v.Valid() {
			app.ValidationErrResponse(w, r, v.Errors)
			return
		}
		user, dataMap, scope, err := app.Models.GetUserWithData(r.Context(), token)
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
		req := app.SetUserDetailsToCtx(r, user, token, dataMap, scope)
		next.ServeHTTP(w, req)
		//TODO: remaining prolly have to use my custom respWriter

		// check if written if yes then send to db
		// if app.IsWritten(req) {
		// 	//write to db
		// 	app.Models.TokenModel.StoreSessionData(req.Context(),token,)
		// }
		m, _ := app.WrittenSess(req)
		// if err != nil {
		// 	app.Logger.Error("WrittensSess gave an error ", "error", err)
		// 	return
		// }
		if m != nil {
			//TODO: may have race cond when same session write to same session togeather in same tab but low priority not sure if needed
			err = app.Models.TokenModel.StoreSessionData(req.Context(), token, m)
			if err != nil {
				app.LogErr("err", r, err)
			}
		}
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

// Use it for endpoints that require authenticated user
// Will check if user is authenticated to. No need to double check it
func (app *serverApplication) reqActivatedUser(next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.GetUser(r)
		scope := app.GetScope(r)
		if scope != data.ScopeOIDC && !user.Activated {
			app.NotActivatedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.reqAuthUser(fn)
}

// NOTE: is this needed???
func (app *serverApplication) PassLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := jsonlog.SetLogger(r, app.Logger)
		next.ServeHTTP(w, req)
	})
}
