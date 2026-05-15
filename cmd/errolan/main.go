package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/enekos/errolan/internal/api"
	"github.com/enekos/errolan/internal/auth"
	"github.com/enekos/errolan/internal/config"
	"github.com/enekos/errolan/internal/db"
	"github.com/enekos/errolan/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}
	if cfg.JWTSecretGenerated {
		logger.Warn("ERROLAN_JWT_SECRET not set; using a freshly generated secret. Tokens will not survive restarts.")
	}

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		logger.Error("open db", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		logger.Error("migrate", "err", err)
		os.Exit(1)
	}

	st := store.New(conn)
	authSvc := auth.New(cfg.JWTSecret, cfg.TokenTTL)

	if err := bootstrapAdmin(logger, st, cfg); err != nil {
		logger.Error("bootstrap admin", "err", err)
		os.Exit(1)
	}

	sdkDir, missing := cfg.ResolveSDKDir()
	if missing {
		logger.Warn("SDK directory not found — /sdk/ disabled",
			"dir", cfg.SDKDir, "hint", "set ERROLAN_SDK_DIR or place files at ./sdk")
	}

	srv := api.NewServer(st, authSvc, api.ServerOptions{
		AdminCORS:      cfg.AdminCORS,
		SDKDir:         sdkDir,
		TrustForwarded: cfg.TrustForwarded,
		WebhookURL:     cfg.WebhookURL,
		Logger:         logger,
	})

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("errolan listening", "addr", cfg.Addr, "db", cfg.DBPath)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("shutdown error", "err", err)
	}
}

// bootstrapAdmin creates the first admin user if the database is empty and
// ERROLAN_ADMIN_EMAIL/PASSWORD are set. An empty DB without credentials is a
// soft-warning state — the operator can still register via the API and promote
// the user manually.
func bootstrapAdmin(logger *slog.Logger, st *store.Store, cfg *config.Config) error {
	n, err := st.CountUsers()
	if err != nil || n > 0 {
		return err
	}
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		logger.Warn("no users yet — set ERROLAN_ADMIN_EMAIL and ERROLAN_ADMIN_PASSWORD to bootstrap, or POST /api/auth/register and then promote manually.")
		return nil
	}
	hash, err := auth.Hash(cfg.AdminPassword)
	if err != nil {
		return err
	}
	if _, err := st.CreateUser(strings.ToLower(cfg.AdminEmail), "Admin", hash, true); err != nil {
		return err
	}
	logger.Info("created bootstrap admin", "email", cfg.AdminEmail)
	return nil
}
