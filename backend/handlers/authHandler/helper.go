package auth

import (
	"errors"
	"slices"
)

var ErrUnexpectedType = errors.New("unexpexted type")

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

// valid codes will be in an array
func (app *AuthApplication) SetOIDCState(token, code string) error {
	key := app.Cfg.Auth.OIDCStateKey
	return app.SetOIDCVal(token, code, key)
}

func (app *AuthApplication) VerifyOIDCState(token, code string) (bool, error) {
	key := app.Cfg.Auth.OIDCStateKey
	return app.VerifyOIDCCode(token, key, code)
}
func (app *AuthApplication) VerifyOIDCCode(token, key, code string) (bool, error) {
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

func (app *AuthApplication) SetOIDCNonce(token, code string) error {
	key := app.Cfg.Auth.OIDCNonceKey
	return app.SetOIDCVal(token, code, key)
}

// NOTE: there might be a race condition in the nonce updating to db part but dk if it matters
func (app *AuthApplication) VerifyOIDCNonce(token, code string) (bool, error) {
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
	nonce, ok := val.([]string)
	if !ok {
		return false, nil
	}
	if !slices.Contains(nonce, code) {
		return false, nil
	}
	newNonce := make([]string, 0, len(nonce)-1)
	for i := range nonce {
		if nonce[i] != code {
			newNonce = append(newNonce, nonce[i])
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
