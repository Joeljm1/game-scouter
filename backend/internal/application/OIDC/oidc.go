package oidc

import (
	"encoding/json"
	"fmt"
	"io"
)

type OIDCTokenResp struct {
	AccessToken  string  `json:"access_token"`
	TokenType    string  `json:"token_type"`    // must be Bearer
	RefreshToken *string `json:"refresh_token"` //Since its option, its a pointer
	ExpiresIn    int     `json:"expires_in"`
	IDToken      string  `json:"id_token"` // is the jwt of the oidc flow
}

type OIDCError struct {
	Msg string
	Err error
}

func (je OIDCError) Error() string {
	return fmt.Sprintf("Msg:%vErr:%v", je.Msg, je.Err.Error())
}

// expects resp.Body
func NewResp(t io.Reader) (*OIDCTokenResp, error) {
	var tr OIDCTokenResp
	err := json.NewDecoder(t).Decode(&tr)
	if err != nil {
		return nil, OIDCError{
			Msg: "Decoding OIDCToken error",
			Err: err,
		}
	}
	return &tr, nil

}
