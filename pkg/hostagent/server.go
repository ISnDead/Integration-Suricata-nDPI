package hostagent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/systemd"
)

type Server struct {
	deps   Deps
	ln     net.Listener
	server *http.Server
}

func New(deps Deps) (*Server, error) {
	if deps.SocketPath == "" {
		return nil, fmt.Errorf("socket path is empty")
	}
	if deps.SuricataCfgPath == "" {
		return nil, fmt.Errorf("suricata config path is empty")
	}
	if deps.NDPIPluginPath == "" {
		return nil, fmt.Errorf("ndpi plugin path is empty")
	}
	if deps.SuricataUnit == "" {
		return nil, fmt.Errorf("suricata unit is empty")
	}
	if deps.RestartTimeout <= 0 {
		deps.RestartTimeout = 20 * time.Second
	}
	if deps.Systemd == nil {
		deps.Systemd = systemd.NewManager(deps.SystemctlPath, nil)
	}
	if deps.FS == nil {
		deps.FS = fsutil.OSFS{}
	}

	if err := removeIfSocket(deps.SocketPath); err != nil {
		return nil, err
	}

	ln, err := net.Listen("unix", deps.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("listen unix %s: %w", deps.SocketPath, err)
	}

	_ = os.Chmod(deps.SocketPath, 0o660)

	h := NewHandlers(deps)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/ndpi/status", h.NDPIStatus)
	mux.HandleFunc("/ndpi/enable", h.NDPIEnable)
	mux.HandleFunc("/ndpi/disable", h.NDPIDisable)

	s := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	return &Server{
		deps:   deps,
		ln:     ln,
		server: s,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	logger.Infow("Host agent started",
		"socket", s.deps.SocketPath,
		"unit", s.deps.SuricataUnit,
		"suricata_config", s.deps.SuricataCfgPath,
		"ndpi_plugin", s.deps.NDPIPluginPath,
	)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(s.ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Stop(shCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.server.Shutdown(ctx)

	if s.ln != nil {
		_ = s.ln.Close()
	}

	if rmErr := removeIfSocket(s.deps.SocketPath); rmErr != nil {
		logger.Warnw("Failed to remove socket path", "path", s.deps.SocketPath, "error", rmErr)
	}

	return err
}

func removeIfSocket(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot access socket path %s: %w", path, err)
	}

	if (info.Mode() & os.ModeSocket) == 0 {
		return fmt.Errorf("socket path exists but is not a unix socket: %s", path)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove unix socket %s: %w", path, err)
	}
	return nil
}
