package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"integration-suricata-ndpi/pkg/logger"
)

func (r *Runner) startHTTPServer(ctx context.Context) error {
	if r.cfg == nil {
		return fmt.Errorf("config is not loaded")
	}

	addr := r.cfg.HTTP.Addr
	if addr == "" {
		return fmt.Errorf("http.addr is empty")
	}

	mux := http.NewServeMux()
	r.registerRoutes(mux)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http listen %s: %w", addr, err)
	}

	r.httpServer = server

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warnw("HTTP server shutdown failed", "error", err)
		}
	}()

	go func() {
		logger.Infow("HTTP server started", "addr", addr)
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Errorw("HTTP server failed", "error", err)
			select {
			case r.httpErrCh <- err:
			default:
			}
		}
	}()

	return nil
}

func (r *Runner) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", r.handleHealth)
	mux.HandleFunc("/plan", r.handlePlan)
	mux.HandleFunc("/apply", r.handleApply)
	mux.HandleFunc("/ndpi/enable", r.handleNDPIEnable)
	mux.HandleFunc("/ndpi/disable", r.handleNDPIDisable)
}
