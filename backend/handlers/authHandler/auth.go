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

func NewAuthTokenCookie(userID int64, token data.Token, ttl time.Duration) *http.Cookie {
	cookie := http.Cookie{
		Name:     "SessionCookie",
		Value:    token.Plaintext,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	return &cookie
}
