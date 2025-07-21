package data

import "github.com/jackc/pgx/v5/pgxpool"

type UserModel struct {
	Pool *pgxpool.Pool
}
