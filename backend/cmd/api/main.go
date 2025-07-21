package main

import (
	"flag"
	"game-scouter-api/internal/application"
	"game-scouter-api/internal/jsonlog"
	"log/slog"
	"os"
	"strings"
)

const version = "1.0.0"

func main() {
	cfg := application.Config{}

	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "environment", "development", "development|staging|production")

	flag.Float64Var(&cfg.Limiter.Rps, "limiter-rps", 2, "Rate limiter for max usage per sec")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 4, "Rate limiter for max burst usage ")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Func("cors-trusted-orgins", "Trusted CORS orgins (space seperated)", func(s string) error {
		cfg.Cors.TrustedOrgins = strings.Fields(" ")
		return nil
	})
	flag.Parse()

	app := &application.Application{
		Cfg:    cfg,
		Logger: jsonlog.New(os.Stdout),
	}
	serverApp := serverApplication{Application: app}
	err := serverApp.run()
	if err != nil {
		app.Logger.Error("Error running server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
