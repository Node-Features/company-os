// Command companyd is the CompanyOS core service: Kernel, Application,
// Governance, Identity, Runtime, and Daemon co-located in one process,
// per ADR-0004 (docs/adr/ADR-0004-first-slice-technology-stack.md).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/httpapi"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load() // best-effort local dev convenience; production sets real env vars

	log.Println("companyd: starting")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// nil until DATABASE_URL is configured, so /health can honestly report
	// "not_configured" instead of a misleading connection failure.
	var db httpapi.DBPinger
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pool, err := supabase.Connect(ctx, dsn)
		if err != nil {
			log.Printf("companyd: failed to connect to database: %v", err)
		} else {
			defer pool.Close()
			db = pool
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", httpapi.HealthHandler(db))

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		log.Printf("companyd: listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("companyd: http server error: %v", err)
		}
	}()

	// TODO: construct Kernel, Application, Governance, Identity, Runtime, and
	// Daemon, then run the Daemon lifecycle described in
	// docs/architecture/daemon.md#lifecycle.

	<-ctx.Done()
	log.Println("companyd: shutdown signal received, draining")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("companyd: http server shutdown error: %v", err)
	}

	// TODO: stop accepting work, drain bounded in-flight operations,
	// checkpoint or abandon leases safely, then close dependencies,
	// per docs/architecture/daemon.md#lifecycle.
}
