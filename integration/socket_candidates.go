package integration

import (
	"fmt"
	"os"
)

func FirstExistingSocket(paths []string) (string, error) {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if (info.Mode() & os.ModeSocket) != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("no unix socket found in candidates: %v", paths)
}
