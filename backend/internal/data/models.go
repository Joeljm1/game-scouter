// Package data contains all the models
// and other data structures needed
package data

import "github.com/jackc/pgx/v5/pgxpool"

type Models struct {
	User UserModel
}

func New(pool *pgxpool.Pool) Models {
	return Models{
		User: UserModel{
			Pool: pool,
		},
	}
}
