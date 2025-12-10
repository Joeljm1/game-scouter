package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserModel struct {
	Pool *pgxpool.Pool
}

var (
	ErrUniqueViolation = errors.New("unique key violation found")
	ErrConflictFound   = errors.New("conflict found")
	ErrNoRows          = pgx.ErrNoRows
	ErrUserExists      = errors.New("user already exists")
)

// Inserts User to db
//
// Insert check for [ErrUniqueViolation] as error after inserting
func (m *UserModel) Insert(ctx context.Context, user *User) error {
	query := `INSERT INTO users
	(name ,email,password_hash,activated)
	VALUES ($1,$2,$3,$4) 
	RETURNING id, created_at,version`
	args := []any{
		user.Name,
		user.Email,
		user.Password.Hash,
		user.Activated,
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	err := m.Pool.QueryRow(ctxTimeout, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Version,
	)
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			return ErrUniqueViolation
		}
		return err
	}
	return nil
}

// created new user if oidc not done before or sets password if oidc was done before
func (m *UserModel) InsertUser(ctx context.Context, user *User) error {
	userFromDb, err := m.GetUserFromEmail(ctx, user.Email)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return m.Insert(ctx, user)
		}
		return err
	}
	if userFromDb.Password.Hash == nil { // only logged in with oidc before
		err = m.Update(ctx, user)
		return err
	}
	return ErrUserExists
}

func (m *UserModel) Update(ctx context.Context, user *User) error {
	query := `UPDATE users SET
			name=$1 ,email=$2 ,activated=$3,
			password_hash=$4 ,version=version+1
			WHERE id=$5 AND version=$6 RETURNING version`
	args := []any{
		user.Name,
		user.Email,
		user.Activated,
		user.Password.Hash,
		user.ID,
		user.Version,
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	err := m.Pool.QueryRow(ctxTimeout, query, args...).Scan(&user.Version)
	var e *pgconn.PgError
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrConflictFound
		case errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation:
			return ErrUniqueViolation
		default:
			return err
		}
	}
	return nil
}

// NOTE: Token should not be hashed one just plaintext
// NOTE: changin this need to check for errors
// Gives token from db
func (m *UserModel) GetUserfromTokenWithSess(ctx context.Context, token string, scope string) (*User, map[string]any, error) {
	// userFromCache, ok1 := m.CacheSess.getUser(token)
	// dataFromCache, ok2 := m.CacheSess.getData(token)
	// if ok1 && ok2 {
	// 	return userFromCache, dataFromCache, nil
	// }
	hashArr := sha256.Sum256([]byte(token))
	hash := hashArr[:]
	var data []byte

	query := `SELECT id,created_at,name,email,password_hash,activated,version,data
			FROM users JOIN token ON token.user_id=users.id WHERE token.hash=$1
			AND token.scope=$2 AND token.expiry>$3`
	timeNow := time.Now().UTC().Format("2006-01-02 15:04:05+00")
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var user User
	err := m.Pool.QueryRow(ctxTimeout, query, hash, scope, timeNow).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
		&data,
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, nil, ErrNoRows
		default:
			return nil, nil, err
		}
	}
	// data could be nil so giveing default val
	var dataMap = map[string]any{}
	if len(data) != 0 {
		dataReader := bytes.NewReader(data)
		err = gob.NewDecoder(dataReader).Decode(&dataMap)
		if err != nil {
			return nil, nil, err
		}
	}
	// m.CacheSess.setUser(token, &user, dataMap)
	return &user, dataMap, nil
}

// return [ErrNoRows] if no users
// if user.Password.Hash then user logged in with oidc but nat with email
func (m *UserModel) GetUserFromEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id,created_at,name,email,password_hash,activated,version
			FROM users where email=$1`
	var user User
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := m.Pool.QueryRow(ctxTimeout, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrNoRows
		default:
			return nil, err
		}
	}
	return &user, nil
}
