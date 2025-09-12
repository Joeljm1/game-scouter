package auth

import (
	"errors"
	"slices"
)

var ErrUnexpectedType = errors.New("unexpexted type")

const OIDCURL = "https://accounts.google.com/o/oauth2/v2/auth?client_id=%v&response_type=code&state=%v&scope=openid%20profile%20email&redirect_uri=%v&nonce=%v"

// TODO: Add like a limit like 10 and then remove oldest ones
func (app *AuthApplication) SetOIDCVal(token, code, key string) error {
	tok, err := app.Models.TokenModel.GetTokenFromTokenStr(token)
	if err != nil {
		return err
	}
	tokData, err := app.GetSessDataMap(tok.Data)
	if err != nil {
		return err
	}
	val, ok := tokData[key]
	if !ok {
		tokData[key] = []string{code}
	} else {
		codes, ok := val.([]string)
		if !ok {
			return ErrUnexpectedType
		}
		codes = append(codes, code)
		tokData[key] = codes
	}
	byteData, err := app.SerializeGoB(tokData)
	if err != nil {
		return err
	}
	tok.Data = byteData
	err = app.Models.TokenModel.Update(tok)
	if err != nil {
		return err
	}
	return nil
}

// valid codes will be set in an array
func (app *AuthApplication) SetOIDCState(token, state string) error {
	key := app.Cfg.Auth.OIDCStateKey
	return app.SetOIDCVal(token, state, key)
}

func (app *AuthApplication) VerifyOIDCState(token, state string) (bool, error) {
	key := app.Cfg.Auth.OIDCStateKey
	return app.VerifyOIDCCode(token, key, state)
}

func (app *AuthApplication) VerifyOIDCCode(token, key, code string) (bool, error) {
	// Check if error returned is \[pgx.ErrNoRows]
	codes, ok, err := app.Models.TokenModel.GetSessionVal(token, key)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	codesSlice, ok := codes.([]string)
	if !ok {
		app.Logger.Error("Code is not slice")
		return false, ErrUnexpectedType
	}
	return slices.Contains(codesSlice, code), nil
}

func (app *AuthApplication) SetOIDCNonce(token, nonce string) error {
	key := app.Cfg.Auth.OIDCNonceKey
	return app.SetOIDCVal(token, nonce, key)
}

// NOTE: there might be a race condition in the nonce updating to db part but dk if it matters
func (app *AuthApplication) VerifyOIDCNonce(token, nonce string) (bool, error) {
	key := app.Cfg.Auth.OIDCNonceKey
	tok, err := app.Models.TokenModel.GetTokenFromTokenStr(token)
	if err != nil {
		return false, err
	}
	dataMap, err := app.GetSessDataMap(tok.Data)
	if err != nil {
		return false, err
	}
	val, ok := dataMap[key]
	if !ok {
		return false, nil
	}
	nonces, ok := val.([]string)
	if !ok {
		return false, nil
	}
	if !slices.Contains(nonces, nonce) {
		return false, nil
	}
	newNonce := make([]string, 0, len(nonces)-1)
	for i := range nonces {
		if nonces[i] != nonce {
			newNonce = append(newNonce, nonces[i])
		}
	}
	dataMap[key] = newNonce
	tokDat, err := app.SerializeGoB(dataMap)
	if err != nil {
		return false, err
	}
	tok.Data = tokDat
	err = app.Models.TokenModel.Update(tok)
	if err != nil {
		return false, err
	}
	return true, nil
}
