// Package application contains Application
// which contains data to be shared by all
// handlers and thier configuration
package application

import (
	"context"
	"flag"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/jsonlog"
	"game-scouter-api/internal/mailer"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Application struct {
	Cfg          *Config
	Logger       *slog.Logger
	Models       data.Models
	BackgroundWG sync.WaitGroup
	Mailer       *mailer.Mailer
	HttpClient   *http.Client
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
func App() *Application {

	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{
		Ctx:       ctx,
		CtxCancel: cancel,
	}
	// in dev env recieved from the make file
	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "environment", "development", "development|staging|production")

	//Rate limiter
	flag.Float64Var(&cfg.Limiter.Rps, "limiter-rps", 2, "Rate limiter for max usage per sec")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 4, "Rate limiter for max burst usage ")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.IntVar(&cfg.Limiter.ShardNo, "shard", 8, "Number of shards for rate limiter")

	// db
	// flag.StringVar(&cfg.DB.DSN, "db-dsn", os.Getenv("SCOUTER_DB_DSN"), "PostgreSQL DSN")
	flag.StringVar(&cfg.DB.DSN, "db-dsn", "postgres://scouter:pa55word@localhost/gamescouter?sslmode=disable", "PostgreSQL DSN")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	// flag.IntVar(&cfg.DB.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.DB.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time input in form 10s,10m etc")
	flag.Func("cors-trusted-orgins", "Trusted CORS orgins (space seperated)", func(s string) error {
		cfg.Cors.TrustedOrgins = strings.Fields(s)
		return nil
	})

	//SMTP
	flag.StringVar(&cfg.SMTP.Host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.SMTP.Username, "smtp-username", "6aaeac8c642e51", "SMTP username")
	flag.StringVar(&cfg.SMTP.Password, "smtp-password", "39664138a0eab9", "SMTP password")
	flag.StringVar(&cfg.SMTP.Sender, "smtp-sender", "joeljosephcl10@gmail.com", "SMTP sender")

	flag.StringVar(&cfg.TokenLife.AuthToken.LifeStr, "authTokenLife", "24h", "How long a session token is alive. Units are all valid units in time.ParseDuration")
	flag.StringVar(&cfg.TokenLife.ActivateToken.LifeStr, "activateTokenLife", "48h", "How long a activation token is alive. Units are all valid units in time.ParseDuration")

	//Auth
	flag.StringVar(&cfg.SessionCookie, "session-cookie", "SessionCookie", "Name of the session cookie")
	// flag.StringVar(&cfg.OIDC.Google.ClientID, "google-outh-client-id", os.Getenv("ClientID"), "client id for google outh")
	// flag.StringVar(&cfg.OIDC.Google.ClientSecret, "google-outh-client-secret", os.Getenv("ClientSecret"), "client secret for google outh")
	//NOTE: Change this later to get from env var
	flag.StringVar(&cfg.OIDC.Google.ClientID, "google-outh-client-id", "910004182414-m8ret44ge5jmrdlte6cl6oae9g3chcok.apps.googleusercontent.com", "client id for google outh")
	//NOTE: Change this later to get from env var
	flag.StringVar(&cfg.OIDC.Google.ClientSecret, "google-outh-client-secret", "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", "client secret for google outh")
	flag.StringVar(&cfg.OIDC.Google.OIDCRedirectURL, "google-outh-redirect-url", "http://localhost:4000/auth/google/redirect", "client secret for google outh")

	flag.StringVar(&cfg.Auth.OIDCStateKey, "OIDCStateKey", "OIDCState", "key for storing OIDC state for CSRF protection")
	flag.StringVar(&cfg.Auth.OIDCNonceKey, "OIDCNonceKey", "OIDCNonce", "key for storing OIDC nonce for oidc")

	flag.Parse()

	app := &Application{
		Cfg:    &cfg,
		Logger: jsonlog.New(os.Stdin),
	}
	return app
}

// Logger needs to initialized in app before calling this
// serperate config cause newApp wont error so logger can be initialized first
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
	app.Logger.Info("Google OIDC discovery and config complete")
	cfg := *app.Cfg
	pool, err := openDB(cfg)
	if err != nil {
		return err
	}
	app.Models = data.NewModels(pool, cfg.Ctx)
	m, err := mailer.New(cfg.SMTP.Host, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender, cfg.SMTP.Port)
	if err != nil {
		return err
	}
	app.Mailer = m
	return nil
}
