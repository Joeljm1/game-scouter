package application

import (
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
}

type Application struct {
	Cfg          Config
	Logger       *slog.Logger
	BackgroundWG sync.WaitGroup
}
