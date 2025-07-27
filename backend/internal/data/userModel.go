package data

import (
	"context"
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
)

// Insert check for ErrUniqueViolation as error after inserting
func (m *UserModel) Insert(user *User) error {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := m.Pool.QueryRow(ctx, query, args...).Scan(
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

func (m *UserModel) Update(user *User) error {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := m.Pool.QueryRow(ctx, query, args...).Scan(&user.Version)
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

func (m *UserModel) GetUserfromToken(token []byte, scope string) (*User, error) {
	query := `SELECT id,created_at,name,email,password_hash,activated,version
			FROM users JOIN token ON token.user_id=users.id WHERE token.hash=$1
			AND token.scope=$2 AND token.expiry>$3`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var user User
	err := m.Pool.QueryRow(ctx, query, token, scope, time.Now()).Scan(
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
func (m *UserModel) GetUserFromEmail(email string) (*User, error) {

	query := `SELECT id,created_at,name,email,password_hash,activated,version
			FROM users where email=$1`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var user User
	err := m.Pool.QueryRow(ctx, query, email).Scan(
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
