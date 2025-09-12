package jwt

import (
	"encoding/base64"
	"encoding/json"
	"game-scouter-api/internal/customErr"
	"strings"
)

type JWT struct {
	Header    string
	Payload   string
	Signature string
}

// splits the token string to [JWT]
func New(token string) (JWT, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return JWT{}, customErr.Err{
			Msg: "JWT token invalid format. Not 3 parts",
			Err: nil,
		}
	}
	return JWT{
		Header:    parts[0],
		Payload:   parts[1],
		Signature: parts[2],
	}, nil
}

func (j JWT) DecodedHeader() ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(j.Header)
	if err != nil {
		return nil, customErr.Err{
			Msg: "Error when decoding jwt header",
			Err: err,
		}
	}
	return b, nil
}

type JWTHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

func (j JWT) ParseJWTHeader() (*JWTHeader, error) {
	hd, err := j.DecodedHeader()
	if err != nil {
		return nil, err
	}

	var JWTHeader JWTHeader
	err = json.Unmarshal(hd, &JWTHeader)
	return &JWTHeader, err
}
func (j JWT) DecodedPayload() ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(j.Payload)
	if err != nil {
		return nil, customErr.Err{
			Msg: "Error when decoding jwt payload",
			Err: err,
		}
	}
	return b, nil
}

func (j JWT) DecodedSign() ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(j.Signature)
	if err != nil {
		return nil, customErr.Err{
			Msg: "Error when decoding jwt signature",
			Err: err,
		}
	}
	return b, nil
}

