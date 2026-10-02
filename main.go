// Command banking-app-go is the Go port of the Notify.ai banking example. It
// triggers banking events through the Notify.ai Go SDK.
//
//	NOTIFY_AI_CLIENT_TOKEN=client-... go run .
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/notify-ai-org/banking-app-go/internal/api"
	"github.com/notify-ai-org/banking-app-go/internal/service"
	"github.com/notify-ai-org/banking-app-go/internal/store"
	"github.com/notify-ai-org/client-go/notify"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := notify.ConfigFromEnv()
	if cfg.ACPServerURL == "" {
		cfg.ACPServerURL = "http://localhost:8080"
	}
	if cfg.ApplicationName == "" {
		cfg.ApplicationName = "banking-app-go"
	}
	cfg.BasePackage = "github.com/notify-ai-org/banking-app-go"
	cfg.Logger = log
	client := notify.New(cfg)

	svc := service.New(client, store.NewAccountStore(), store.NewTransactionStore(), log)
	if err := client.Start(); err != nil {
		log.Error("notify start", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	api.Routes(mux, svc)
	addr := ":" + envOr("PORT", "8091")
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Info("banking-app-go listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "error", err)
			stop()
		}
	}()
	<-ctx.Done()

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
	if err := client.Close(shutdown); err != nil {
		log.Warn("notify close", "error", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
