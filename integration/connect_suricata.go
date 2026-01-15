package integration

import (
	"fmt"
	"net"
	"time"

	"integration-suricata-ndpi/pkg/logger"
)

func ConnectSuricata(socketCandidates []string, timeout time.Duration) (*SuricataClient, error) {
	socketPath, err := FirstExistingPath(socketCandidates)
	if err != nil {
		return nil, fmt.Errorf("suricata control socket not found: %w", err)
	}

	logger.Infow("Connecting to Suricata",
		"socket_path", socketPath,
		"timeout", timeout,
	)

	conn, err := net.DialTimeout("unix", socketPath, timeout)
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
