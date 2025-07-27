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

func NewAuthTokenCookie(userID int64, token *data.Token, ttl time.Duration, name string) *http.Cookie {
	cookie := http.Cookie{
		Name:     name,
		Value:    token.Plaintext,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	return &cookie
}

// Generates token for the user, stored it in db and adds a cookie to responseWriter
func (app *AuthApplication) Login(w http.ResponseWriter, userID int64, ttl time.Duration) error {
	token, err := app.Models.TokenModel.GenerateAndInsertToken(userID, ttl, data.ScopeAuthentication)
	if err != nil {
		return err
	}
	cookie := NewAuthTokenCookie(userID, token, ttl, app.Cfg.SessionCookie)
	http.SetCookie(w, cookie)
	return nil
}
