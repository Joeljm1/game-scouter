// Package auth contain handler for
// authenticating
package auth

import (
	"game-scouter-api/internal/application"
	"game-scouter-api/internal/data"
	"net/http"
	"time"
)

type AuthApplication struct {
	*application.Application
}

// Generates token for the user, stored it in db and adds a cookie to responseWriter
func (app *AuthApplication) Login(w http.ResponseWriter, userID int64, ttl time.Duration) error {
	token, err := app.Models.TokenModel.GenerateAndInsertToken(userID, ttl, data.ScopeAuthentication)
	if err != nil {
		return err
	}
	cookie := app.NewTokenCookie(token, ttl, app.Cfg.SessionCookie)
	http.SetCookie(w, cookie)
	return nil
}
