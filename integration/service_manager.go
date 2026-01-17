package integration

import (
	"fmt"
	"time"

	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/netutil"
)

func EnsureSuricataRunning(socketCandidates []string) error {
	return EnsureSuricataRunningWithDialer(socketCandidates, nil)
}

func EnsureSuricataRunningWithDialer(socketCandidates []string, dialer netutil.Dialer) error {
	if dialer == nil {
		dialer = netutil.DefaultDialer{}
	}

	const probeTimeout = 2 * time.Second

	socketPath, err := FirstDialableUnixSocket(socketCandidates, probeTimeout, dialer)
	if err != nil {
		return fmt.Errorf("suricata is unavailable: control socket not reachable: %w", err)
	}

	logger.Infow("Suricata is reachable via socket",
		"socket_path", socketPath,
	)
	return nil
}
