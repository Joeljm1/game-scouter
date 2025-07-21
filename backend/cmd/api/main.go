package main

import (
	"flag"
	"game-scouter-api/internal/jsonlog"
	"log/slog"
	"os"
	"strings"
	"sync"
)

const version = "1.0.0"

type config struct {
	port    int
	env     string
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	cors struct {
		trustedOrgins []string
	}
}

type application struct {
	cfg          config
	logger       *slog.Logger
	backgroundWG sync.WaitGroup
}

func main() {
	cfg := config{}

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "environment", "development", "development|staging|production")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter for max usage per sec")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter for max burst usage ")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Func("cors-trusted-orgins", "Trusted CORS orgins (space seperated)", func(s string) error {
		cfg.cors.trustedOrgins = strings.Fields(" ")
		return nil
	})
	flag.Parse()

	app := application{
		cfg:    cfg,
		logger: jsonlog.New(os.Stdout),
	}
	err := app.run()
	if err != nil {
		app.logger.Error("Error running server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
