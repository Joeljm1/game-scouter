package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"game-scouter-api/internal/validator"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ScopeActivation     = "Activation"
	ScopeAuthentication = "Authentication"
)

type Token struct {
	UserID    int64     `json:"-"`
	Plaintext string    `json:"-"`
	Hash      []byte    `json:"hash"`
	Scope     string    `json:"scope"`
	Expiry    time.Time `json:"expiry"`
}

type TokenModel struct {
	Pool *pgxpool.Pool
}

// crypto/rand.Read comment says it never returns an error and i believe them
func GenerateToken(userID int64, ttl time.Duration, scope string) *Token {
	tok := &Token{
		UserID: userID,
		Scope:  ScopeActivation,
		Expiry: time.Now().Add(ttl),
	}
	randomByte := make([]byte, 16)
	_, _ = rand.Read(randomByte) // comments in source code says it never returns error
	tok.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomByte)
	hash := sha256.Sum256([]byte(tok.Plaintext))
	tok.Hash = hash[:]
	return tok
}

func (m *TokenModel) Insert(tok *Token) error {
	query := `INSERT INTO token 
	(user_id,hash,expiry,scope) 
	VALUES($1,$2,$3,$4)`
	ars := []any{tok.UserID, tok.Hash, tok.Expiry, tok.Scope}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := m.Pool.Exec(ctx, query, ars...)
	if err != nil {
		return err
	}
	return nil
}

// Generated token and inserts it to db
func (m *TokenModel) GenerateAndInsertToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	tok := GenerateToken(userID, ttl, scope)
	err := m.Insert(tok)
	return tok, err
}

func ValidateToken(v *validator.Validator, token string) {
	v.Assert(token != "", "token", "should not be empty")
	v.Assert(len(token) == 26, "token", "not valid")
}

func (m *TokenModel) DeleteAllToken(userID int64, scope string) error {
	query := `DELETE FROM token WHERE user_id=$1 AND scope=$2`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := m.Pool.Exec(ctx, query, userID, scope)
	return err
}
