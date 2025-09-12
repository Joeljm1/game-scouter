// Package application contains Application
// which contains data to be shared by all
// handlers and thier configuration
package application

import (
	"context"
	"game-scouter-api/internal/application/OIDC/google"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/mailer"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	Ctx       context.Context
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
		MaxIdleTIme string
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
		CacheUser    data.CachedUser
	}
}

type Application struct {
	Cfg          *Config
	Logger       *slog.Logger
	Models       data.Models
	BackgroundWG sync.WaitGroup
	Mailer       *mailer.Mailer
	HttpClient   *http.Client
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

func (app *Application) ConfigureHttp() {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	app.HttpClient = client

}

func (app *Application) ConfigureGoogle() error {
	// done like this cause google client secret and all were set at initalisation in main
	err := app.Cfg.OIDC.Google.Configure(app.HttpClient)
	if err != nil {
		return err
	}
	return nil
}
func (app *Application) Configure() error {
	app.ConfigureHttp()
	err := app.Cfg.Configure()
	if err != nil {
		return err
	}
	err = app.ConfigureGoogle()
	if err != nil {
		return err
	}
	return nil
}
