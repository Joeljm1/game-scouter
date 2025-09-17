// Package data contains all the models
// and other data structures needed
package data

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Models struct {
	UserModel  UserModel
	TokenModel TokenModel
}

func NewModels(pool *pgxpool.Pool, ctx context.Context) Models {
	cs := &CashedSess{
		Users: map[string]*CachedUser{},
	}
	//TODO:i doubt this will panic but ther could be nil pointer so need to put a
	//recover over it
	go cs.clean(ctx)
	return Models{
		UserModel: UserModel{
			Pool:      pool,
			CacheSess: cs,
		},
		TokenModel: TokenModel{
			Pool: pool,
		},
	}
}
