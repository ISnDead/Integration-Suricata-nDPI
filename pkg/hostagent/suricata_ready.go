package hostagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"time"
)

type suricatascResp struct {
	Return  string `json:"return"`
	Message any    `json:"message"`
}

func waitSuricataReady(ctx context.Context, suricatascPath, suricataSock string, maxWait time.Duration) error {
	if suricatascPath == "" {
		return errors.New("suricatasc path is empty")
	}
	if suricataSock == "" {
		return errors.New("suricata socket is empty")
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	const (
		initial = 150 * time.Millisecond
		capD    = 2 * time.Second
	)

	var lastErr error
	for attempt := 1; ; attempt++ {
		select {
		case <-deadlineCtx.Done():
			if lastErr == nil {
				lastErr = deadlineCtx.Err()
			}
			return fmt.Errorf("suricata not ready within %s: %w", maxWait, lastErr)
		default:
		}

		ready, err := probeUptime(deadlineCtx, suricatascPath, suricataSock)
		if err == nil && ready {
			return nil
		}
		if err != nil {
			lastErr = err
		}

		sleep := time.Duration(float64(initial) * math.Pow(1.5, float64(attempt-1)))
		if sleep > capD {
			sleep = capD
		}

		t := time.NewTimer(sleep)
		select {
		case <-deadlineCtx.Done():
			t.Stop()
		case <-t.C:
		}
	}
}

func probeUptime(ctx context.Context, suricatascPath, suricataSock string) (bool, error) {
	cmd := exec.CommandContext(ctx, suricatascPath, "-c", "uptime", suricataSock)

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("suricatasc uptime failed: %w (stderr=%q, stdout=%q)", err, errOut.String(), out.String())
	}

	var resp suricatascResp
	if jerr := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); jerr == nil {
		return resp.Return == "OK", nil
	}

	s := out.String()
	if bytes.Contains(bytes.ToUpper([]byte(s)), []byte("OK")) {
		return true, nil
	}
	return false, fmt.Errorf("unexpected suricatasc uptime output: %q (stderr=%q)", out.String(), errOut.String())
}
