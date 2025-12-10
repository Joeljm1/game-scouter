package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"game-scouter-api/internal/helpers"
	"game-scouter-api/internal/validator"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	ScopeActivation     = "Activation"
	ScopeAuthentication = "Authentication"
	ScopeOIDC           = "OIDC_Authentication"
)

type Token struct {
	UserID    int64     `json:"-"`
	Plaintext string    `json:"-"`
	Hash      []byte    `json:"hash"`
	Scope     string    `json:"scope"`
	Expiry    time.Time `json:"expiry"`
	Data      []byte    `json:"-"`
	//TODO: add auth type to may be in scope??
}

type TokenModel struct {
	Pool *pgxpool.Pool
}

// WARN: Data initially nil
func GenerateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	tok := &Token{
		UserID: userID,
		Scope:  scope,
		Expiry: time.Now().UTC().Add(ttl),
		Data:   nil,
	}
	randomByte := make([]byte, 16)
	_, err := rand.Read(randomByte)
	if err != nil {
		return nil, err
	}
	tok.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomByte)
	hash := sha256.Sum256([]byte(tok.Plaintext))
	tok.Hash = hash[:]
	return tok, nil
}

func (m *TokenModel) Insert(ctx context.Context, tok *Token) error {
	query := `INSERT INTO token
	(user_id,hash,expiry,scope,data)
	VALUES($1,$2,$3,$4,$5)`
	ars := []any{tok.UserID, tok.Hash, tok.Expiry, tok.Scope, tok.Data}
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	_, err := m.Pool.Exec(ctx, query, ars...)
	if err != nil {
		return err
	}
	return nil
}

// check for [ErrNoRows] which is = [pgx.ErrNoRows]
func (m *TokenModel) Update(ctx context.Context, tok *Token) error {
	query := `UPDATE token SET expiry=$1,data=$2 where hash=$3 `
	args := []any{tok.Expiry, tok.Data, tok.Hash}
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	cmdTag, err := m.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

// Generated token and inserts it to db
func (m *TokenModel) GenerateAndInsertToken(ctx context.Context, userID int64, ttl time.Duration, scope string) (*Token, error) {
	tok, err := GenerateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	err = m.Insert(ctx, tok)
	return tok, err
}

func ValidateToken(v *validator.Validator, token string) {
	v.Assert(token != "", "token", "should not be empty")
	v.Assert(len(token) == 26, "tokenVal", "not valid")
}

func (m *TokenModel) DeleteAllToken(ctx context.Context, userID int64, scope string) error {
	query := `DELETE FROM token WHERE user_id=$1 AND scope=$2`
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	_, err := m.Pool.Exec(ctx, query, userID, scope)
	return err
}

func MatchPassword(plainText string, hash []byte) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hash, []byte(plainText))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

// Check if error returned from this is [pgx.ErrNoRows]
// Returns token struct if present
// TODO: prolly delete this
func (m *TokenModel) GetTokenFromTokenStr(ctx context.Context, token string) (*Token, error) {
	hashArr := sha256.Sum256([]byte(token))
	hash := hashArr[:]
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `Select user_id,hash,expiry,scope,data from token where hash=$1`
	tok := Token{}
	err := m.Pool.QueryRow(ctx, query, hash).Scan(
		&tok.UserID,
		&tok.Hash,
		&tok.Expiry,
		&tok.Scope,
		&tok.Data,
	)
	return &tok, err
}

// used to store session data into db and cache after req is over an is written
func (m *TokenModel) StoreSessionData(ctx context.Context, token string, dataMap map[string]any) error {
	b, err := helpers.SerializeGoB(dataMap)
	if err != nil {
		return helpers.Err{
			Msg: "Error at store session data while serializing dataMap to gob",
			Err: err,
		}
	}

	hashArr := sha256.Sum256([]byte(token))
	hash := hashArr[:]
	query := `UPDATE token SET data=$1 where hash=$2 `
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	cmdTag, err := m.Pool.Exec(ctx, query, b, hash)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}
