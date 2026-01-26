package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"integration-suricata-ndpi/internal/httpapi"
	"integration-suricata-ndpi/pkg/agentclient"
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
	srv := httpapi.New(httpapi.Deps{
		Plan: func(ctx context.Context) (any, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			return PlanConfig(r.opts.Apply)
		},

		Apply: func(ctx context.Context) (any, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			return ApplyConfigWithContext(ctx, r.opts.Apply)
		},

		EnsureSuricata: func(ctx context.Context) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			return r.ensureSuricataViaHostAgent(ctx)
		},

		EnableNDPI: func(ctx context.Context) (any, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			return r.callHostAgent(ctx, true)
		},

		DisableNDPI: func(ctx context.Context) (any, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			return r.callHostAgent(ctx, false)
		},
	})

	srv.Register(mux)
}

func (r *Runner) callHostAgent(ctx context.Context, enable bool) (*agentclient.ToggleResponse, error) {
	if r.cfg == nil {
		return nil, fmt.Errorf("config is not loaded")
	}

	socket := r.cfg.HTTP.HostAgentSocket
	if socket == "" {
		return nil, fmt.Errorf("http.host_agent_socket is empty")
	}

	timeout := r.cfg.HTTP.HostAgentTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := agentclient.New(socket, timeout)

	if enable {
		return client.EnableNDPI(ctx)
	}
	return client.DisableNDPI(ctx)
}

func (r *Runner) ensureSuricataViaHostAgent(ctx context.Context) error {
	if r.cfg == nil {
		return fmt.Errorf("config is not loaded")
	}

	socket := r.cfg.HTTP.HostAgentSocket
	if socket == "" {
		return fmt.Errorf("http.host_agent_socket is empty")
	}

	timeout := r.cfg.HTTP.HostAgentTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := agentclient.New(socket, timeout)

	resp, err := client.EnsureSuricataStarted(ctx)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("host-agent ensure suricata failed: %s (%s)", resp.Message, resp.Code)
	}

	logger.Infow("Suricata ensured via host-agent",
		"started", resp.Started,
		"socket", resp.Socket,
	)
	return nil
}
