package integration

import (
	"fmt"
	"time"

	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/netutil"
)

func ConnectSuricata(socketCandidates []string, timeout time.Duration) (*SuricataClient, error) {
	return ConnectSuricataWithDialer(socketCandidates, timeout, nil)
}

func ConnectSuricataWithStart(opts SuricataStartOptions, timeout time.Duration) (*SuricataClient, error) {
	if err := EnsureSuricataStarted(opts); err != nil {
		return nil, err
	}
	return ConnectSuricataWithDialer(opts.SocketCandidates, timeout, opts.Dialer)
}

func ConnectSuricataWithDialer(socketCandidates []string, timeout time.Duration, dialer netutil.Dialer) (*SuricataClient, error) {
	if dialer == nil {
		dialer = netutil.DefaultDialer{}
	}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	socketPath, err := FirstExistingSocket(socketCandidates)
	if err != nil {
		return nil, fmt.Errorf("suricata control socket not found: %w", err)
	}

	logger.Infow("Connecting to Suricata",
		"socket_path", socketPath,
		"timeout", timeout,
	)

	conn, err := dialer.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		logger.Errorw("Failed to connect to unix socket",
			"socket_path", socketPath,
			"error", err,
		)
		return nil, fmt.Errorf("failed to connect to %s: %w", socketPath, err)
	}

	_ = conn.SetDeadline(time.Now().Add(timeout))

	logger.Infow("Suricata connection established",
		"socket_path", socketPath,
	)

	return &SuricataClient{
		Conn: conn,
		Path: socketPath,
	}, nil
}
