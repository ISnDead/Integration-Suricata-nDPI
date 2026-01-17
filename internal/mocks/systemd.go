package mocks

import (
	"context"
	"time"
)

type SystemdManager struct {
	RestartFunc func(ctx context.Context, unit string, timeout time.Duration) error
}

func (m *SystemdManager) Restart(ctx context.Context, unit string, timeout time.Duration) error {
	if m.RestartFunc != nil {
		return m.RestartFunc(ctx, unit, timeout)
	}
	return nil
}
