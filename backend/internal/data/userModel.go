package data

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CachedUser struct {
	User     *User
	lastUsed time.Time
}

type CashedSess struct {
	Users map[string]*CachedUser
	sync.RWMutex
}

func (cs *CashedSess) getUser(token string) (*User, bool) {
	cs.RLock()
	defer cs.RUnlock()
	user, ok := cs.Users[token]
	if !ok {
		return nil, false
	}
	user.lastUsed = time.Now()
	return user.User, true
}

func (cs *CashedSess) setUser(token string, user *User) {
	cs.Lock()
	defer cs.Unlock()
	cs.Users[token] = &CachedUser{
		User:     user,
		lastUsed: time.Now(),
	}
}

type UserModel struct {
	Pool      *pgxpool.Pool
	CacheSess CashedSess
}

// TODO: clear cache val after some time and be concurrent safe
// in [UserModel.GetUserfromToken]

var (
	ErrUniqueViolation = errors.New("unique key violation found")
	ErrConflictFound   = errors.New("conflict found")
	ErrNoRows          = pgx.ErrNoRows
)

// Insert check for ErrUniqueViolation as error after inserting
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

// NOTE: Token should not be hashed one just pllaintext
// TODO: May be if need chache the tokens to temporarily
func (m *UserModel) GetUserfromToken(ctx context.Context, token string, scope string) (*User, error) {
	userFromCache, ok := m.CacheSess.getUser(token)
	if ok {
		return userFromCache, nil
	}
	hashArr := sha256.Sum256([]byte(token))
	hash := hashArr[:]
	query := `SELECT id,created_at,name,email,password_hash,activated,version
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
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrNoRows
		default:
			return nil, err
		}
	}
	m.CacheSess.setUser(token, &user)
	return &user, nil
}
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
