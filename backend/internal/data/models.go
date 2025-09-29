// Package data contains all the models
// and other data structures needed
package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Models struct {
	UserModel  UserModel
	TokenModel TokenModel
	CacheSess  *CachedSess
}

func NewModels(pool *pgxpool.Pool, ctx context.Context) Models {
	cs := NewCache(5 * time.Minute)
	//TODO:i doubt this will panic but ther could be nil pointer so need to put a
	//recover over it
	go cs.clean(ctx)
	return Models{
		UserModel: UserModel{
			Pool: pool,
		},
		TokenModel: TokenModel{
			Pool: pool,
		},
		CacheSess: cs,
	}
}

// error can be [ErrNoRows]
func (m Models) GetUserWithData(ctx context.Context, tok string) (*User, map[string]any, error) {
	user, dataMap, ok := m.CacheSess.getUserAndData(tok)
	if ok {
		return user, dataMap, nil
	}
	user, dataMap, err := m.UserModel.GetUserfromTokenWithSess(ctx, tok, ScopeAuthentication)
	if err != nil {
		return nil, nil, err
	}
	m.CacheSess.setUser(tok, user, dataMap)
	return user, dataMap, nil
}
