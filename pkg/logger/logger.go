package logger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	Log  *zap.Logger = zap.NewNop()
	once sync.Once
)

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

func Sync() {
	_ = Log.Sync()
}
