package integration

import (
	"fmt"
	"os"
)

// FirstExistingPath возвращает первый путь из списка, который существует в системе.
func FirstExistingPath(paths []string) (string, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("не найден ни один из путей: %v", paths)
}
