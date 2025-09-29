package google

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"game-scouter-api/internal/application/OIDC/jwt"
	"game-scouter-api/internal/customErr"
	"math/big"
	"net/http"
	"time"
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

type GoogleOIDCResp struct {
	Aud string `json:"aud"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
	Iss string `json:"iss"`
	Sub string `json:"sub"`

	At_hash        *string `json:"at_hash"`
	Azp            *string `json:"azp"`
	Email          *string `json:"email"`
	Email_verified *bool   `json:"email_verified"`
	Family_name    *string `json:"family_name"`
	Given_name     *string `json:"given_name"`
	Hd             *string `json:"hd"`
	Locale         *string `json:"locale"`
	Name           *string `json:"name"`
	Nonce          *string `json:"nonce"`
	Picture        *string `json:"picture"`
	Profile        *string `json:"profile"`
}

// return nil if valid.
// Nonce must be verified seperatly
func (g Google) VerifyPayload(gResp GoogleOIDCResp) error {
	if g.DocumentDiscovery == nil {
		panic("Google Discovery document not set")
	}
	if gResp.Iss != g.DocumentDiscovery.Issuer {
		return ReqError{
			Msg: "Issuer of payload is not same",
			Err: errors.New("Issuer of payload is not same"),
		}
	}
	if gResp.Aud != g.ClientID {
		return ReqError{
			Msg: "Aud of payload is not same as clientID",
			Err: errors.New("Aud of payload is not same as clientID"),
		}
	}
	t := time.Unix(gResp.Exp, 0)
	if time.Since(t) > 0 {
		return ReqError{
			Msg: "Token expired",
			Err: errors.New("Token expired"),
		}
	}
	return nil
}

// Verifies the jwt and return nonce as string to for verifying
func (g Google) Verify(jwt jwt.JWT) (bool, string, error) {
	header, err := jwt.ParseJWTHeader()
	if err != nil {
		return false, "", err
	}
	//default is RSA256
	if header.Alg != "RS256" && header.Alg != "" {
		return false, "", customErr.Err{
			Msg: "Algorithm of jwt in oidc to google not rsa256",
			Err: errors.New("algorithm of jwt in oidc to google not rsa256"),
		}
	}
	pk, err := g.validPublicKey(header.Kid)
	if err != nil {
		return false, "", err
	}
	ds, err := jwt.DecodedSign()
	if err != nil {
		return false, "", err
	}
	// OIDC tells its optional if got directly from provider but i did it
	// may be no need of verify??
	err = g.verifyRSA256(pk, jwt.Header, jwt.Payload, ds)
	if err != nil {
		return false, "", nil
	}
	decP, err := jwt.DecodedPayload()
	if err != nil {
		return false, "", err
	}
	decPReader := bytes.NewReader(decP)
	var gOIDCResp GoogleOIDCResp
	err = json.NewDecoder(decPReader).Decode(&gOIDCResp)
	if err != nil {
		return false, "", customErr.Err{
			Msg: "Error decoding Payload to GoogleOIDCResp",
			Err: err,
		}
	}
	err = g.VerifyPayload(gOIDCResp)
	if err != nil {
		return false, "", err
	}
	if gOIDCResp.Nonce == nil {
		return false, "", ReqError{
			Msg: "Nonce is missing",
			Err: errors.New("Nonce is missing"),
		}
	}
	return true, *gOIDCResp.Nonce, nil
}
