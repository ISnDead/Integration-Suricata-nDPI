package runner

import "context"

type Service interface {
	Run(ctx context.Context) error
	Stop()
}

type Runner struct {
	svc Service
}

func New(svc Service) *Runner {
	return &Runner{svc: svc}
}

func (r *Runner) Run(ctx context.Context) error {
	return r.svc.Run(ctx)
}

func (r *Runner) Stop() {
	r.svc.Stop()
}
