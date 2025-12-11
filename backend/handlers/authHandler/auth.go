// Package auth contain handler for
// authenticating
package auth

import (
	"context"
	"game-scouter-api/internal/application"
	"game-scouter-api/internal/data"
	"net/http"
)

type AuthApplication struct {
	*application.Application
}

// Generates token for the user, stored it in db and adds a cookie to responseWriter.
// ttl is for token in browser
func (app *AuthApplication) Login(ctx context.Context, w http.ResponseWriter, userID int64, scope data.Scope) error {
	ttl := app.Cfg.TokenLife.AuthToken.LifeDuration
	token, err := app.Models.TokenModel.GenerateAndInsertToken(ctx, userID, ttl, scope)
	if err != nil {
		return err
	}
	cookie := app.NewTokenCookie(token, ttl, app.Cfg.SessionCookie)
	http.SetCookie(w, cookie)
	return nil
}
