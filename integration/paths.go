package integration

import (
	"fmt"
	"os"
)

func FirstExistingPath(paths []string) (string, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("none of the paths were found: %v", paths)
}
