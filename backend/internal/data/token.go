package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/gob"
	"errors"
	"game-scouter-api/internal/validator"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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
	Data      []byte    `json:"-"`
}

type TokenModel struct {
	Pool *pgxpool.Pool
}

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

func (m *TokenModel) Insert(tok *Token) error {
	query := `INSERT INTO token 
	(user_id,hash,expiry,scope,data) 
	VALUES($1,$2,$3,$4,$5)`
	ars := []any{tok.UserID, tok.Hash, tok.Expiry, tok.Scope, tok.Data}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := m.Pool.Exec(ctx, query, ars...)
	if err != nil {
		return err
	}
	return nil
}

// check for [ErrConflictFound] and [ErrNoRows] which is = [pgx.ErrNoRows]
func (m *TokenModel) Update(tok *Token) error {
	query := `UPDATE token SET expiry=$1,data=$2 where hash=$3 `
	args := []any{tok.Expiry, tok.Data, tok.Hash}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	cmdTag, err := m.Pool.Exec(ctx, query, args...)
	if err != nil {
		return ErrConflictFound
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

// Generated token and inserts it to db
func (m *TokenModel) GenerateAndInsertToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	tok, err := GenerateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	err = m.Insert(tok)
	return tok, err
}

func ValidateToken(v *validator.Validator, token string) {
	v.Assert(token != "", "token", "should not be empty")
	v.Assert(len(token) == 26, "tokenVal", "not valid")
}

func (m *TokenModel) DeleteAllToken(userID int64, scope string) error {
	query := `DELETE FROM token WHERE user_id=$1 AND scope=$2`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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

// Check if error returned is [pgx.ErrNoRows]
func (m *TokenModel) GetTokenFromTokenStr(token string) (*Token, error) {
	hashArr := sha256.Sum256([]byte(token))
	hash := hashArr[:]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

// Check if error returned is [pgx.ErrNoRows] and [ErrConflictFound]
func (m *TokenModel) StoreSessionVal(token, key string, val any) error {
	tok, err := m.GetTokenFromTokenStr(token)
	if err != nil {
		return err
	}
	dataMap := map[string]any{}
	if len(tok.Data) > 0 {
		dataReader := bytes.NewReader(tok.Data)
		err = gob.NewDecoder(dataReader).Decode(&dataMap)
		if err != nil {
			return err
		}
	}
	dataMap[key] = val
	buff := new(bytes.Buffer)
	err = gob.NewEncoder(buff).Encode(dataMap)
	if err != nil {
		return err
	}
	tok.Data = buff.Bytes()
	err = m.Update(tok)
	return err
}

// Check if error returned is \[pgx.ErrNoRows]
func (m *TokenModel) GetSessionVal(token string, key string) (any, bool, error) {
	tok, err := m.GetTokenFromTokenStr(token)
	if err != nil {
		return nil, false, err
	}
	dataMap := map[string]any{}
	if len(tok.Data) > 0 {
		dataReader := bytes.NewReader(tok.Data)
		err = gob.NewDecoder(dataReader).Decode(&dataMap)
		if err != nil {
			return nil, false, err
		}
		val, ok := dataMap[key]
		return val, ok, nil
	}
	return nil, false, nil
}
