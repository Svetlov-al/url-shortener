package main

import (
	"context"
	"net/http"
	"os"
	ssogrpc "url-shortener/internal/clients/sso/grpc"
	"url-shortener/internal/config"
	"url-shortener/internal/http-server/router"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage/sqlite"

	"log/slog"
)

const (
	envLocal      = "local"
	envProduction = "production"
)

func main() {
	cfg := config.LoadConfig("./config/local.yaml")

	logger := setupLogger(cfg.Env)

	logger.Info("starting url-shortener", "env", cfg.Env)

	ssoClient, err := ssogrpc.New(
		context.Background(),
		logger,
		cfg.Clients.SSO.Address,
		cfg.Clients.SSO.Timeout,
		cfg.Clients.SSO.RetriesCount,
	)
	if err != nil {
		logger.Error("failed to create sso client", sl.Err(err))
		os.Exit(1)
	}

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		logger.Error(
			"Не удалось инициализировать базу данных", sl.Err(err))
		os.Exit(1)
	}

	logger.Info("База инициализирована", "path", cfg.StoragePath)

	r := router.New(
		logger,
		storage,
		ssoClient,
		cfg.AppSecret,
		cfg.Clients.SSO.Timeout,
	)

	logger.Info("server started", slog.String("address", cfg.HTTPServer.Address))

	server := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Error("failed to start server", sl.Err(err))
		os.Exit(1)
	}

}

func setupLogger(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case envLocal:
		logger = setupPrettySlog()
	case envProduction:
		logger = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return logger
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}
	h := opts.NewPrettyHandler(os.Stdout)

	return slog.New(h)
}
