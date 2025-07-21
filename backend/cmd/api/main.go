package main

import (
	"context"
	"flag"
	"fmt"
	"game-scouter-api/internal/application"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/jsonlog"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "1.0.0"

func openDB(cfg application.Config) (*pgxpool.Pool, error) {
	pgxPoolCfg, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, err
	}
	idletime, err := time.ParseDuration(cfg.DB.MaxIdleTIme)
	if err != nil {
		return nil, err
	}
	pgxPoolCfg.MaxConnIdleTime = idletime
	pgxPoolCfg.MaxConns = int32(cfg.DB.MaxOpenConns)
	pool, err := pgxpool.New(context.Background(), cfg.DB.DSN)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func main() {
	// in dev recieved from the make file
	env := os.Getenv("SCOUTER_DB_DSN")
	fmt.Println(env)
	cfg := application.Config{}

	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "environment", "development", "development|staging|production")

	flag.Float64Var(&cfg.Limiter.Rps, "limiter-rps", 2, "Rate limiter for max usage per sec")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 4, "Rate limiter for max burst usage ")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")

	// db
	flag.StringVar(&cfg.DB.DSN, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	// flag.IntVar(&cfg.DB.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.DB.MaxIdleTIme, "db-max-idel-time", "15m", "PostgreSQL max connection idle time input in form 10s,10m etc")
	flag.Func("cors-trusted-orgins", "Trusted CORS orgins (space seperated)", func(s string) error {
		cfg.Cors.TrustedOrgins = strings.Fields(" ")
		return nil
	})

	flag.Parse()

	app := &application.Application{
		Cfg:    cfg,
		Logger: jsonlog.New(os.Stdout),
	}
	pool, err := openDB(app.Cfg)
	if err != nil {
		app.Logger.Error("DB pool connection error", "Err", err.Error())
	}
	app.Models = data.New(pool)
	serverApp := serverApplication{Application: app}
	err = serverApp.run()
	if err != nil {
		app.Logger.Error("Error running server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
