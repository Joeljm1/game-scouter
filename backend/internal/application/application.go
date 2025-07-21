// Package application contains Application
// which contains data to be shared by all
// handlers
package application

import (
	"game-scouter-api/internal/data"
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
}

type Application struct {
	Cfg          Config
	Logger       *slog.Logger
	Models       data.Models
	BackgroundWG sync.WaitGroup
}
