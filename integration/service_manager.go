package integration

import (
	"fmt"
	"net"
	"os"
	"time"

	"integration-suricata-ndpi/pkg/logger"
)

func EnsureSuricataRunning(socketCandidates []string) error {
	socketPath, err := FirstExistingPath(socketCandidates)
	if err != nil {
		return fmt.Errorf("suricata is unavailable: control socket not found: %w", err)
	}

	logger.Infow("Checking Suricata availability",
		"socket_path", socketPath,
	)

	info, err := os.Stat(socketPath)
	if err != nil {
		return fmt.Errorf("failed to stat suricata socket (%s): %w", socketPath, err)
	}

	if (info.Mode() & os.ModeSocket) == 0 {
		return fmt.Errorf("path exists but is not a unix socket: %s", socketPath)
	}

	const probeTimeout = 2 * time.Second
	conn, err := net.DialTimeout("unix", socketPath, probeTimeout)
	if err != nil {
		return fmt.Errorf("suricata socket exists but dial failed (%s): %w", socketPath, err)
	}
	_ = conn.Close()

	logger.Infow("Suricata is reachable via socket",
		"socket_path", socketPath,
	)
	return nil
}
