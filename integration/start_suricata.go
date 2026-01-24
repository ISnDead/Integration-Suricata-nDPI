package integration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/netutil"
	"integration-suricata-ndpi/pkg/systemd"
)

func EnsureSuricataStarted(opts SuricataStartOptions) error {
	if opts.Systemd == nil {
		opts.Systemd = systemd.NewManager(opts.SystemctlPath, nil)
	}
	if opts.Dialer == nil {
		opts.Dialer = netutil.DefaultDialer{}
	}
	if opts.StartTimeout <= 0 {
		opts.StartTimeout = 20 * time.Second
	}

	if err := EnsureSuricataRunningWithDialer(opts.SocketCandidates, opts.Dialer); err == nil {
		return nil
	}

	unit := strings.TrimSpace(opts.SystemdUnit)
	if unit == "" {
		return fmt.Errorf("suricata unit is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.StartTimeout)
	defer cancel()

	if err := opts.Systemd.Restart(ctx, unit, opts.StartTimeout); err != nil {
		return fmt.Errorf("failed to start suricata via systemd: %w", err)
	}

	if err := EnsureSuricataRunningWithDialer(opts.SocketCandidates, opts.Dialer); err != nil {
		return fmt.Errorf("suricata started but socket not reachable: %w", err)
	}

	return nil
}
