package main

import (
	"fmt"
	"game-scouter-api/internal/application"
	"log/slog"
	_ "net/http/pprof"
	"os"
)

const version = "1.0.0"

func main() {
	// uncomment below code when wanting pprof
	// go func() {
	// 	http.ListenAndServe("localhost:6060", nil)
	// }()
	app := application.App()
	err := app.Configure()
	if err != nil {
		fmt.Println(err)
		if app.Logger != nil {
			app.Logger.Error("Configuring Config failed", "Err", err.Error())
		} else {
			slog.Log(app.Cfg.Ctx, slog.LevelError, "error", err)
		}
		os.Exit(1)
	}
	serverApp := serverApplication{Application: app}
	err = serverApp.run()
	if err != nil {
		app.Logger.Error("Error running server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
