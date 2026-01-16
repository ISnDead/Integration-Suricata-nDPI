package integration

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/logger"
)

func RestartUnit(parent context.Context, unit string, timeout time.Duration) error {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return fmt.Errorf("systemd unit is empty")
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	logger.Infow("Restarting systemd unit", "unit", unit)

	cmd := exec.CommandContext(ctx, "systemctl", "restart", unit)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("systemctl restart timed out for unit=%s", unit)
	}
	if err != nil {
		return fmt.Errorf("systemctl restart failed for unit=%s: %v output=%q", unit, err, strings.TrimSpace(string(out)))
	}
	return nil
}
