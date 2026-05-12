package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/enekos/errolan/internal/api"
	"github.com/enekos/errolan/internal/auth"
	"github.com/enekos/errolan/internal/db"
	"github.com/enekos/errolan/internal/store"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := envOr("ERROLAN_ADDR", ":8080")
	dbPath := envOr("ERROLAN_DB", "errolan.db")
	jwtSecret := os.Getenv("ERROLAN_JWT_SECRET")
	adminEmail := envOr("ERROLAN_ADMIN_EMAIL", "")
	adminPassword := envOr("ERROLAN_ADMIN_PASSWORD", "")
	adminCORS := envOr("ERROLAN_ADMIN_CORS", "*")
	sdkDir := envOr("ERROLAN_SDK_DIR", "sdk")

	if jwtSecret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			log.Fatalf("could not generate jwt secret: %v", err)
		}
		jwtSecret = hex.EncodeToString(buf)
		log.Printf("WARNING: ERROLAN_JWT_SECRET not set; using a freshly generated secret. Tokens will not survive restarts.")
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st := store.New(conn)
	authSvc := auth.New(jwtSecret, 7*24*time.Hour)

	// Bootstrap admin if none exists and credentials provided.
	if n, err := st.CountUsers(); err == nil && n == 0 {
		if adminEmail == "" || adminPassword == "" {
			log.Printf("no users yet — set ERROLAN_ADMIN_EMAIL and ERROLAN_ADMIN_PASSWORD to bootstrap, or POST /api/auth/register and then promote manually.")
		} else {
			hash, err := auth.Hash(adminPassword)
			if err != nil {
				log.Fatalf("hash admin password: %v", err)
			}
			if _, err := st.CreateUser(strings.ToLower(adminEmail), "Admin", hash, true); err != nil {
				log.Fatalf("bootstrap admin: %v", err)
			}
			log.Printf("created bootstrap admin %s", adminEmail)
		}
	}

	if _, err := os.Stat(sdkDir); errors.Is(err, os.ErrNotExist) {
		log.Printf("SDK directory %q not found — /sdk/ disabled. Set ERROLAN_SDK_DIR or place files at ./sdk.", sdkDir)
		sdkDir = ""
	}

	srv := api.NewServer(st, authSvc)
	srv.AdminCORS = adminCORS
	srv.SDKDir = sdkDir
	srv.TrustForwarded = strings.EqualFold(os.Getenv("ERROLAN_TRUST_FORWARDED"), "true")
	srv.WebhookURL = os.Getenv("ERROLAN_WEBHOOK_URL")

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("errolan listening on %s (db=%s)", addr, dbPath)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
