package google

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"game-scouter-api/internal/application/OIDC/jwt"
	"game-scouter-api/internal/customErr"
	"math/big"
	"net/http"
)

const googleDiscoveryURL = "https://accounts.google.com/.well-known/openid-configuration"

type Google struct {
	ClientID        string
	ClientSecret    string
	OIDCRedirectURL string
	//better to be pointer so i check if it exists
	DocumentDiscovery *GoogleDiscovery
	JWKS              JWKS
}
type GoogleDiscovery struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// TODO: Prolly make this type outside to share
type AuthError struct {
	Err error
	Msg string
}

func (err AuthError) Error() string {
	return fmt.Sprintf("error:%v , Msg:%v", err.Err.Error(), err.Msg)
}

type JWK struct {
	Kty    string `json:"kty"`
	Use    string `json:"use"`
	Kid    string `json:"kid"`
	N      string `json:"n"`
	E      string `json:"e"`
	Alg    string `json:"alg"`
	PubKey *rsa.PublicKey
}

func (j *JWK) SetRSAPublicKey() error {
	eBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return AuthError{
			Msg: "Error Decoding base64Url of E in JWT for rsa public key",
			Err: err,
		}
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(j.N)
	if err != nil {
		return AuthError{
			Msg: "Error Decoding base64Url of N in JWT for rsa public key",
			Err: err,
		}
	}

	e := 0
	//coverting bytes to int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	pk := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}
	j.PubKey = pk
	return nil
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

func (g *Google) GoogleDiscoveryDoc(client *http.Client) error {
	resp, err := client.Get(googleDiscoveryURL)
	if err != nil {
		return AuthError{
			Err: err,
			Msg: "Error getting discovery doc from url",
		}
	}
	defer resp.Body.Close()
	var gd GoogleDiscovery
	err = json.NewDecoder(resp.Body).Decode(&gd)
	if err != nil {
		return AuthError{
			Err: err,
			Msg: "Decoder error in google discovery",
		}
	}
	g.DocumentDiscovery = &gd
	return nil
}

// Should be called only after discovery doc is set
func (g *Google) GetGoogleJWKToken(client *http.Client) error {
	if client == nil {
		panic("Error: CLient is nil")
	}
	if g.DocumentDiscovery == nil {
		panic("Google discovery doc not set")
	}
	resp, err := client.Get(g.DocumentDiscovery.JwksURI)
	if err != nil {
		return AuthError{
			Err: err,
			Msg: "Failed to get from token endpont",
		}
	}
	defer resp.Body.Close()
	var jwsResp JWKS
	err = json.NewDecoder(resp.Body).Decode(&jwsResp)
	if err != nil {
		return AuthError{
			Err: err,
			Msg: " Failed to decode JWS resp from token endpoint",
		}
	}
	for i := range jwsResp.Keys {
		err = jwsResp.Keys[i].SetRSAPublicKey()
		if err != nil {
			return err
		}
	}
	g.JWKS = jwsResp
	return nil
}

// Configures go OIDC details
func (g *Google) Configure(client *http.Client) error {
	err := g.GoogleDiscoveryDoc(client)
	if err != nil {
		return err
	}
	err = g.GetGoogleJWKToken(client)
	if err != nil {
		return err
	}
	return nil
}

func (g *Google) validPublicKey(kid string) (*rsa.PublicKey, error) {
	for _, j := range g.JWKS.Keys {
		if j.Kid == kid {
			return j.PubKey, nil
		}
	}
	// No public key with same kid found
	return nil, customErr.Err{
		Msg: "no valid public key found in google oidc for user jwt",
		Err: errors.New("no valid public key found in google oidc for user jwt"),
	}
}

func (g Google) verifyRSA256(pk *rsa.PublicKey, header string, payload string, sig []byte) error {
	if pk == nil {
		return customErr.Err{
			Msg: "public key in verify RSA256 is nil",
			Err: errors.New("public key in verify RSA256 is nil"),
		}
	}
	sigInp := header + "." + payload
	hash := sha256.New()
	hash.Write([]byte(sigInp))
	hashSum := hash.Sum(nil)
	err := rsa.VerifyPKCS1v15(pk, crypto.SHA256, hashSum, sig)
	//not used currently
	if err != nil {
		return customErr.Err{
			Msg: "Error in verifyRSA256 for verifying rs256 of google oidc",
			Err: err,
		}
	}
	return nil
}

// Verifies the jwt
func (g Google) Verify(jwt jwt.JWT) (bool, error) {
	header, err := jwt.ParseJWTHeader()
	if err != nil {
		return false, err
	}
	if header.Alg != "RS256" {
		return false, customErr.Err{
			Msg: "Algorithm of jwt in oidc to google not rsa256",
			Err: errors.New("algorithm of jwt in oidc to google not rsa256"),
		}
	}
	pk, err := g.validPublicKey(header.Kid)
	if err != nil {
		return false, err
	}
	ds, err := jwt.DecodedSign()
	if err != nil {
		return false, err
	}
	err = g.verifyRSA256(pk, jwt.Header, jwt.Payload, ds)
	if err != nil {
		return false, nil
	}
	return true, nil
}
