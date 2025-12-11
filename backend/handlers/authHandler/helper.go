// TODO: change error handling of this package to use custom error with
// custom messgages to
// TODO: Prolly cache sessions to
package auth

import (
	"bytes"
	"errors"
	"net/http"
	"slices"
	"strings"
)

var ErrUnexpectedType = errors.New("unexpexted type")

const OIDCURL = "https://accounts.google.com/o/oauth2/v2/auth?client_id=%v&response_type=code&state=%v&scope=openid%20profile%20email&redirect_uri=%v&nonce=%v"

// TODO: Add like a limit like 10 and then remove oldest ones
func (app *AuthApplication) SetOIDCVal(r *http.Request, token, code, key string) error {
	sess := app.GetSessData(r)

	codes, ok := sess.Get(key)
	var err error = nil
	var codesSlice []string
	if !ok {
		codesSlice = []string{}
	} else {
		codesSlice, ok = codes.([]string)
		if !ok {
			codesSlice = []string{}
			err = ErrUnexpectedType
		}
	}
	codesSlice = append(codesSlice, code)
	sess.Set(key, codesSlice)
	return err
}

// valid codes will be set in an array
func (app *AuthApplication) SetOIDCState(r *http.Request, token, state string) error {
	key := app.Cfg.Auth.OIDCStateKey
	return app.SetOIDCVal(r, token, state, key)
}

func (app *AuthApplication) VerifyOIDCState(r *http.Request, state string) (bool, error) {
	key := app.Cfg.Auth.OIDCStateKey
	return app.VerifyOIDCCode(r, key, state)
}

func (app *AuthApplication) SetOIDCNonce(r *http.Request, token, state string) error {
	key := app.Cfg.Auth.OIDCNonceKey
	return app.SetOIDCVal(r, token, state, key)
}

// TODO: remove old codes like an lru cache
func (app *AuthApplication) VerifyOIDCCode(r *http.Request, key, code string) (bool, error) {
	// Check if error returned is \[pgx.ErrNoRows]

	// codes, ok, err := app.Models.TokenModel.GetSessionVal(ctx, token, key)
	// codes, ok, err := app.Models.TokenModel.GetSessionVal(ctx, token, key)
	sess := app.GetSessData(r)
	codes, ok := sess.Get(key)
	// if err != nil {
	// 	return false, err
	// }
	if !ok {
		return false, nil
	}
	codesSlice, ok := codes.([]string)
	if !ok {
		app.Logger.Error("Code is not slice")
		return false, ErrUnexpectedType
	}
	if !slices.Contains(codesSlice, code) {
		return false, nil
	}
	newCodes := make([]string, 0, len(codesSlice)-1)
	// remove used nonce to prevent double login attack or smthing like that forgot attack name
	for i := range codesSlice {
		if codesSlice[i] != code {
			newCodes = append(newCodes, codesSlice[i])
		}
	}
	sess.Set(key, newCodes)
	return true, nil
}

var ErrEmailInvalid = errors.New("Invalid email")

// verifies nonce and checks if nonce is no
func (app *AuthApplication) VerifyOIDCNonceAndEmail(r *http.Request, nonce *string, email *string, verified *bool) (bool, error) {

	if email == nil || strings.TrimSpace(*email) == "" || verified == nil || !*verified {
		// OIDC supposed to give oidc but just in case
		return false, ErrEmailInvalid
	}
	key := app.Cfg.Auth.OIDCNonceKey
	if nonce == nil {
		return false, nil
	}
	return app.VerifyOIDCCode(r, key, *nonce)
}

func WelcomeHTML(name string) (*bytes.Buffer, error) {
	buff := new(bytes.Buffer)
	tmplData := struct {
		Name string
	}{
		Name: name,
	}
	err := welcomeTmpl.ExecuteTemplate(buff, "welcome", tmplData)
	if err != nil {
		return nil, err
	}
	return buff, err

}
