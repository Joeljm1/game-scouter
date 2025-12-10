package application

import (
	"context"
	"game-scouter-api/internal/application/OIDC/google"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Ctx       context.Context // for global context to share
	CtxCancel context.CancelFunc
	Port      int
	Env       string
	TokenLife struct {
		AuthToken struct {
			LifeStr      string
			LifeDuration time.Duration
		}
		ActivateToken struct {
			LifeStr      string
			LifeDuration time.Duration
		}
	}
	//Name of the cookie
	SessionCookie string
	Limiter       struct {
		Rps     float64
		Burst   int
		Enabled bool
		ShardNo int
	}
	Cors struct {
		TrustedOrgins []string
	}
	DB struct {
		DSN          string
		MaxOpenConns int
		// MaxIdleConns int
		MaxIdleTime string
	}
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		Sender   string
	}
	OIDC struct {
		Google google.Google
	}
	Auth struct {
		OIDCStateKey string
		OIDCNonceKey string
	}
}

func (cfg *Config) ConfigureAuthTokenLife() error {
	t, err := time.ParseDuration(cfg.TokenLife.AuthToken.LifeStr)
	if err != nil {
		return err
	}
	cfg.TokenLife.AuthToken.LifeDuration = t
	return nil
}

func (cfg *Config) ConfigureActivateTokenLife() error {
	t, err := time.ParseDuration(cfg.TokenLife.ActivateToken.LifeStr)
	if err != nil {
		return err
	}
	cfg.TokenLife.ActivateToken.LifeDuration = t
	return nil
}

func (cfg *Config) Configure() error {
	err := cfg.ConfigureAuthTokenLife()
	if err != nil {
		return err
	}
	err = cfg.ConfigureActivateTokenLife()
	if err != nil {
		return err
	}
	return nil

}

func openDB(cfg Config) (*pgxpool.Pool, error) {
	pgxPoolCfg, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, err
	}
	idletime, err := time.ParseDuration(cfg.DB.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	pgxPoolCfg.MaxConnIdleTime = idletime
	pgxPoolCfg.MaxConns = int32(cfg.DB.MaxOpenConns)
	pool, err := pgxpool.NewWithConfig(context.Background(), pgxPoolCfg)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
