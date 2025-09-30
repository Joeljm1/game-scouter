package application

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/helpers"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Envelope map[string]any // envelopes the JSON

// WriteJSON writes a struct as json to w
// error can come only before writing to it
// so if error comes you can custom write
func (app *Application) WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	js = append(js, '\n')
	for k, v := range headers {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

// ReadJSON Read r and get json from request body as struct.
// Give pointer to struct(dst cal)
func (app *Application) ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	// Initialize the json.Decoder, and call the DisallowUnknownFields() method on it
	// before decoding. This means that if the JSON from the client now includes any
	// field which cannot be mapped to the target destination, the decoder will return
	// an error instead of just ignoring the field.
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// Use the errors.As() function to check whether the error has the type
		// *json.SyntaxError. If it does, then return a plain-english error message
		// which includes the location of the problem.
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		// In some circumstances Decode() may also return an io.ErrUnexpectedEOF error
		// for syntax errors in the JSON. So we check for this using errors.Is() and
		// return a generic error message. There is an open issue regarding this at
		// https://github.com/golang/go/issues/25956.
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// Likewise, catch any *json.UnmarshalTypeError errors. These occur when the
		// JSON value is the wrong type for the target destination. If the error relates
		// to a specific field, then we include that in our error message to make it
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

			// An io.EOF error will be returned by Decode() if the request body is empty. We
			// check for this with errors.Is() and return a plain-english error message
			// instead.
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

			// If the JSON contains a field which cannot be mapped to the target destination
			// then Decode() will now return an error message in the format "json: unknown
			// field "<name>"". We check for this, extract the field name from the error,
			// and interpolate it into our custom error message. Note that there's an open
			// issue at https://github.com/golang/go/issues/29035 regarding turning this
			// into a distinct error type in the future.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s ", fieldName)

			// Use the errors.As() function to check whether the error has the type
			// *http.MaxBytesError. If it does, then it means the request body exceeded our
			// size limit of 1MB and we return a clear error message.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

			// A json.InvalidUnmarshalError error will be returned if we pass something
			// that is not a non-nil pointer to Decode(). We catch this and panic,
			// rather than returning an error to our handler. At the end of this chapter
			// we'll talk about panicking versus returning errors, and discuss why it's an
			// appropriate thing to do in this specific situation.
		case errors.As(err, &invalidUnmarshalError):
			// panic becasue it is an unexpected error which should never occur ie programmer problem
			// invalidUnmarshalError means smthing other than non nil pointer given for decoder
			panic(err)

		default:
			return err
		}
	}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must contain only single JSON")
	}
	return nil
}

func (app *Application) Background(fn func()) {
	app.BackgroundWG.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				app.Logger.Error("Go routine paniced", slog.Any("Error", err))
			}
			app.BackgroundWG.Done()
		}()
		fn()
	}()
}

// empty string if key does not exist
func GetFromQuery(r *http.Request, key string) string {
	val := r.URL.Query().Get(key)
	return val
}

var userKey = data.ContextKey("user")

type UserVal struct {
	User *data.User
	Tok  string
	Data *SessionData
}

type SessionData struct {
	data    map[string]any
	written bool
}

func (sd *SessionData) Set(key string, value any) {
	if !sd.written {
		sd.written = true
	}
	sd.data[key] = value
}

func (sd *SessionData) Remove(key string) {
	if !sd.written {
		sd.written = true
	}
	delete(sd.data, key)
}

func (sd *SessionData) Get(key string) (any, bool) {
	val, ok := sd.data[key]
	if !ok {
		return nil, false
	}
	return val, true
}

// sets user,session data map and toekn val to context
func (app *Application) SetUserDetailsToCtx(r *http.Request, user *data.User, tok string, data map[string]any) *http.Request {
	if user == nil || tok == "" {
		//TODO: prolly make this an error instead
		panic("user is nill or token is empty")
	}
	// prolly better to use pointer here so that i can get the val from even if req is made to a new req in middlewares
	setVal := &UserVal{
		User: user,
		Tok:  tok,
		Data: &SessionData{
			data:    data,
			written: false,
		},
	}
	ctx := context.WithValue(r.Context(), userKey, setVal)
	req := r.WithContext(ctx)
	return req
}

func (app *Application) GetUser(r *http.Request) *data.User {
	userVal, ok := r.Context().Value(userKey).(*UserVal)
	if !ok || userVal.User == nil {
		panic("User not found")
	}
	return userVal.User
}

func (app *Application) GetTok(r *http.Request) string {
	userVal, ok := r.Context().Value(userKey).(*UserVal)
	if !ok || userVal.Tok == "" {
		panic("Token not found")
	}
	return userVal.Tok
}

func (app *Application) GetSessData(r *http.Request) *SessionData {
	userVal, ok := r.Context().Value(userKey).(*UserVal)
	if !ok || userVal.Tok == "" {
		panic("Token not found")
	}
	return userVal.Data
}

// return nil if session was not written
func (app *Application) WrittenSess(r *http.Request) (map[string]any, error) {
	userVal, ok := r.Context().Value(userKey).(*UserVal)
	if !ok || userVal.Tok == "" {
		panic("Token not found")
	}
	if !userVal.Data.written {
		return nil, nil
	}
	return userVal.Data.data, nil
}

func (app *Application) NewTokenCookie(token *data.Token, ttl time.Duration, name string) *http.Cookie {
	cookie := http.Cookie{
		Name:     name,
		Value:    token.Plaintext,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	return &cookie
}

// generated token and insterts it to db with user0 and return cookie
func (app *Application) AnonUserCookie(ctx context.Context) (*http.Cookie, *data.Token, error) {
	tok, err := app.Models.TokenModel.GenerateAndInsertToken(ctx, 0, app.Cfg.TokenLife.AuthToken.LifeDuration, data.ScopeAuthentication)
	if err != nil {
		return nil, nil, err
	}
	cookie := app.NewTokenCookie(tok, app.Cfg.TokenLife.AuthToken.LifeDuration, app.Cfg.SessionCookie)
	return cookie, tok, nil
}

// bits expects the no of bits of entropy
func (app *Application) CryptoRandomStr(bits int) string {
	buff := make([]byte, bits)
	_, _ = rand.Read(buff)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buff)
}

// Used to get dataMap from []bytes in tok.Data
// Asking for bytes and not taking token for fn and doing
// as i can resuse a token from calling function for stuff like updating
// to reduce db calls
func (app *Application) GetSessDataMap(SessData []byte) (map[string]any, error) {
	if len(SessData) == 0 { //nil check
		return map[string]any{}, nil
	}
	var tokData map[string]any
	err := helpers.DeserializeGoB(SessData, &tokData)
	if err != nil {
		return nil, err
	}
	return tokData, nil
}
