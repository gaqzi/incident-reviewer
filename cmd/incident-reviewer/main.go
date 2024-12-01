package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/gaqzi/incident-reviewer/internal/app"
)

func main() {
	cfg := app.NewConfig()
	ctx, cancel := context.WithCancel(context.Background())
	server, err := app.Start(ctx, cfg)
	if err != nil {
		slog.Error("failed to start server", "error", err)
	}

	slog.Info("server started", "addr", "http://"+server.Config.Addr)

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt, os.Kill)

	for {
		sig := <-shutdown
		switch sig {
		case os.Interrupt, os.Kill:
			cancel()
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)

			if err := server.Stop(shutCtx); err != nil {
				slog.Error("failed to shut safely", "error", err)
				shutCancel()
				os.Exit(1)
			}

			shutCancel()
			return
		default:
			slog.Warn("unhandled signal", "signal", sig.String())
		}
	}
}