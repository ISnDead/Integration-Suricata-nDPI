package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunWithSignals(parent context.Context, svc Service, shutdownTimeout time.Duration) error {
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.Start(ctx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	stopErr := svc.Stop(stopCtx)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return stopErr
	case <-stopCtx.Done():
		return fmt.Errorf("shutdown timed out after %s", shutdownTimeout)
	}
}
