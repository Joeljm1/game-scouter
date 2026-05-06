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
	Cache struct {
		MaxEntries  int
		CacheTTLStr string
		CleanDurStr string
		CleanDur    time.Duration
		CacheTTL    time.Duration
	}
}

func ParseDurAndSet(s string, d *time.Duration) error {
	t, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = t
	return nil
}

func (cfg *Config) Configure() error {
	err := ParseDurAndSet(cfg.TokenLife.AuthToken.LifeStr, &cfg.TokenLife.AuthToken.LifeDuration)
	if err != nil {
		return err
	}

	err = ParseDurAndSet(cfg.TokenLife.ActivateToken.LifeStr, &cfg.TokenLife.ActivateToken.LifeDuration)
	if err != nil {
		return err
	}

	err = ParseDurAndSet(cfg.Cache.CacheTTLStr, &cfg.Cache.CacheTTL)
	if err != nil {
		return err
	}
	err = ParseDurAndSet(cfg.Cache.CleanDurStr, &cfg.Cache.CleanDur)
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
