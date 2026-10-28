// Command server menjalankan layanan landing page pemberitahuan perpindahan
// domain untuk beberapa project.
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

	"gantibdomain/internal/config"
	"gantibdomain/internal/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("konfigurasi tidak valid", "error", err)
		os.Exit(1)
	}

	handler, err := web.New(cfg, logger)
	if err != nil {
		logger.Error("gagal menyiapkan handler", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 14,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server berjalan", "addr", server.Addr, "projects", len(cfg.Projects))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Error("server berhenti tidak normal", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("menghentikan server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("gagal menghentikan server dengan rapi", "error", err)
			os.Exit(1)
		}
	}
}
