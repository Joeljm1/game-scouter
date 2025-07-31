// Package application contains Application
// which contains data to be shared by all
// handlers
package application

import (
	"encoding/base64"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/mailer"
	"log/slog"
	"sync"
	"time"
)

type Config struct {
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
	Key struct {
		Base64URLEncodeStr string
		SecretKey          []byte
	}
	Oauth struct {
		Google struct {
			ClientID     string
			ClientSecret string
		}
	}
}

type Application struct {
	Cfg          *Config
	Logger       *slog.Logger
	Models       data.Models
	BackgroundWG sync.WaitGroup
	Mailer       *mailer.Mailer
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

func (cfg *Config) ConfigureSecretKey() error {
	if cfg.Key.Base64URLEncodeStr == "" {
		panic("secretKey not set")
	}
	secret, err := base64.URLEncoding.DecodeString(cfg.Key.Base64URLEncodeStr)
	if err != nil {
		return err
	}
	cfg.Key.SecretKey = secret
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
	err = cfg.ConfigureSecretKey()
	if err != nil {
		return err
	}
	return nil
}
