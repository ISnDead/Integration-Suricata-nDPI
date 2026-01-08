package integration

import (
	"fmt"
	"os"
)

// FirstExistingPath возвращает первый существующий путь из списка.
// Нужен для смешанной установки (и чтобы не копировать одну и ту же логику в разных файлах).
func FirstExistingPath(paths []string) (string, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("ни один из путей не найден: %v", paths)
}
