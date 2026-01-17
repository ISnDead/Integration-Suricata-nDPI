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

	if info, err := os.Stat(deps.SocketPath); err == nil {
		if (info.Mode() & os.ModeSocket) == 0 {
			return nil, fmt.Errorf("socket path exists but is not a unix socket: %s", deps.SocketPath)
		}
		_ = os.Remove(deps.SocketPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot access socket path %s: %w", deps.SocketPath, err)
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
	}

	return &Server{
		deps:   deps,
		ln:     ln,
		server: s,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	_ = ctx
	logger.Infow("Host agent started",
		"socket", s.deps.SocketPath,
		"unit", s.deps.SuricataUnit,
		"suricata_config", s.deps.SuricataCfgPath,
		"ndpi_plugin", s.deps.NDPIPluginPath,
	)

	if err := s.server.Serve(s.ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.server.Shutdown(ctx)

	if s.ln != nil {
		_ = s.ln.Close()
	}
	_ = os.Remove(s.deps.SocketPath)

	return err
}
