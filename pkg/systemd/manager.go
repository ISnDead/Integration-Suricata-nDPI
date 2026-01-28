package systemd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/executil"
)

type Manager interface {
	Restart(ctx context.Context, unit string, timeout time.Duration) error
}

type ServiceManager struct {
	CommandPath string
	Runner      executil.Runner
}

func NewManager(commandPath string, runner executil.Runner) *ServiceManager {
	if strings.TrimSpace(commandPath) == "" {
		commandPath = "systemctl"
	}
	if runner == nil {
		runner = executil.DefaultRunner{}
	}

	return &ServiceManager{
		CommandPath: commandPath,
		Runner:      runner,
	}
}

func (m *ServiceManager) Restart(parent context.Context, unit string, timeout time.Duration) error {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return fmt.Errorf("systemd unit is empty")
	}
	if timeout <= 0 {
		return fmt.Errorf("restart timeout must be > 0")
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	out, err := m.Runner.CombinedOutput(ctx, m.CommandPath, "restart", unit)
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("systemctl restart timed out for unit=%s", unit)
	}
	if err != nil {
		return fmt.Errorf("systemctl restart failed for unit=%s: %v output=%q", unit, err, strings.TrimSpace(string(out)))
	}
	return nil
}
