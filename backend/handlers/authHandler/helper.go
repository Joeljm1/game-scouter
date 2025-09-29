// TODO: change error handling of this package to use custom error with
// custom messgages to
// TODO: Prolly cache sessions to
package auth

import (
	"errors"
	"net/http"
	"slices"
)

var ErrUnexpectedType = errors.New("unexpexted type")

const OIDCURL = "https://accounts.google.com/o/oauth2/v2/auth?client_id=%v&response_type=code&state=%v&scope=openid%20profile%20email&redirect_uri=%v&nonce=%v"

// TODO: Add like a limit like 10 and then remove oldest ones
func (app *AuthApplication) SetOIDCVal(r *http.Request, token, code, key string) error {
	// tok, err := app.Models.TokenModel.GetTokenFromTokenStr(ctx, token)
	// if err != nil {
	// 	return err
	// }
	// tokData, err := app.GetSessDataMap(tok.Data)
	// if err != nil {
	// 	return err
	// }
	// val, ok := tokData[key]
	// if !ok {
	// 	tokData[key] = []string{code}
	// } else {
	// 	codes, ok := val.([]string)
	// 	if !ok {
	// 		return ErrUnexpectedType
	// 	}
	// 	codes = append(codes, code)
	// 	tokData[key] = codes
	// }
	// byteData, err := helpers.SerializeGoB(tokData)
	// if err != nil {
	// 	return err
	// }
	// tok.Data = byteData
	// err = app.Models.TokenModel.Update(ctx, tok)
	// if err != nil {
	// 	return err
	// }
	// return nil
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
		codesSlice = append(codesSlice, code)
	}
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
	return slices.Contains(codesSlice, code), nil
}

func (app *AuthApplication) VerifyOIDCNonce(r *http.Request, nonce string) (bool, error) {
	// key := app.Cfg.Auth.OIDCNonceKey
	// tok, err := app.Models.TokenModel.GetTokenFromTokenStr(ctx, token)
	// if err != nil {
	// 	return false, err
	// }
	// dataMap, err := app.GetSessDataMap(tok.Data)
	// if err != nil {
	// 	return false, err
	// }
	// val, ok := dataMap[key]
	// if !ok {
	// 	return false, nil
	// }
	// nonces, ok := val.([]string)
	// if !ok {
	// 	return false, errors.New("value of nonces not []slice")
	// }
	// if !slices.Contains(nonces, nonce) {
	// 	return false, nil
	// }
	// newNonce := make([]string, 0, len(nonces)-1)
	// for i := range nonces {
	// 	if nonces[i] != nonce {
	// 		newNonce = append(newNonce, nonces[i])
	// 	}
	// }
	// dataMap[key] = newNonce
	// tokDat, err := helpers.SerializeGoB(dataMap)
	// if err != nil {
	// 	return false, err
	// }
	// tok.Data = tokDat
	// err = app.Models.TokenModel.Update(ctx, tok)
	// if err != nil {
	// 	return false, err
	// }
	// return true, nil
	key := app.Cfg.Auth.OIDCNonceKey
	sess := app.GetSessData(r)
	nonces, ok := sess.Get(key)
	if !ok {
		return false, nil
	}
	nonceSlice, ok := nonces.([]string)
	if !ok {
		app.Logger.Error("Code is not slice")
		return false, ErrUnexpectedType
	}
	if !slices.Contains(nonceSlice, nonce) {
		return false, nil
	}
	newNonce := make([]string, 0, len(nonceSlice)-1)
	for i := range nonceSlice {
		if nonceSlice[i] != nonce {
			newNonce = append(newNonce, nonceSlice[i])
		}
	}
	sess.Set(key, newNonce)
	return true, nil
}
