// Package application contains Application
// which contains data to be shared by all
// handlers
package application

import (
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/mailer"
	"log/slog"
	"sync"
)

type Config struct {
	Port    int
	Env     string
	Limiter struct {
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
}

type Application struct {
	Cfg          Config
	Logger       *slog.Logger
	Models       data.Models
	BackgroundWG sync.WaitGroup
	Mailer       *mailer.Mailer
}
