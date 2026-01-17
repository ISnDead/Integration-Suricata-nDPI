package mocks

import "context"

type ExecRunner struct {
	CombinedOutputFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func (m *ExecRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.CombinedOutputFunc != nil {
		return m.CombinedOutputFunc(ctx, name, args...)
	}
	return nil, nil
}
