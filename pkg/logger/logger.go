package logger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	Log  *zap.Logger = zap.NewNop()
	once sync.Once
)

// Init инициализирует zap-логгер один раз за процесс.
func Init() {
	once.Do(func() {
		l, err := zap.NewProduction()
		if err != nil {
			Log = zap.NewNop()
			return
		}
		Log = l
	})
}

// Sync корректно сбрасывает буферы логгера.
func Sync() {
	_ = Log.Sync()
}
