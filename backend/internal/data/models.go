// Package data contains all the models
// and other data structures needed
package data

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Models struct {
	UserModel  UserModel
	TokenModel TokenModel
}

func New(pool *pgxpool.Pool) Models {
	return Models{
		UserModel: UserModel{
			Pool: pool,
		},
		TokenModel: TokenModel{
			Pool: pool,
		},
	}
}
