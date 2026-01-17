package integration

import (
	"fmt"
	"os"
	"time"

	"integration-suricata-ndpi/pkg/netutil"
)

func FirstExistingPath(paths []string) (string, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("none of the paths were found: %v", paths)
}

func FirstUnixSocketPath(paths []string) (string, error) {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if (info.Mode() & os.ModeSocket) != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("none of the unix socket paths were found: %v", paths)
}

func FirstDialableUnixSocket(paths []string, timeout time.Duration, dialer netutil.Dialer) (string, error) {
	if dialer == nil {
		dialer = netutil.DefaultDialer{}
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	var lastErr error
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			lastErr = err
			continue
		}
		if (info.Mode() & os.ModeSocket) == 0 {
			continue
		}

		c, err := dialer.DialTimeout("unix", p, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		_ = c.Close()
		return p, nil
	}

	if lastErr != nil {
		return "", fmt.Errorf("no dialable unix socket among candidates: %v (last err: %v)", paths, lastErr)
	}
	return "", fmt.Errorf("no dialable unix socket among candidates: %v", paths)
}
