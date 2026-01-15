package logger

import (
	"sync"

	"go.uber.org/zap"
)

type Options struct {
	Level string

	Development bool
}

var (
	once sync.Once
	log  *zap.SugaredLogger = zap.NewNop().Sugar()
)

func Init() {
	InitWithOptions(Options{})
}

func InitWithOptions(opts Options) {
	once.Do(func() {
		var cfg zap.Config
		if opts.Development {
			cfg = zap.NewDevelopmentConfig()
		} else {
			cfg = zap.NewProductionConfig()
		}

		if opts.Level != "" {
			_ = cfg.Level.UnmarshalText([]byte(opts.Level))
		}

		l, err := cfg.Build()
		if err != nil {
			log = zap.NewNop().Sugar()
			return
		}
		log = l.Sugar()
	})
}

func Sync() { _ = log.Sync() }

func Debugw(msg string, keysAndValues ...any) { log.Debugw(msg, keysAndValues...) }
func Infow(msg string, keysAndValues ...any)  { log.Infow(msg, keysAndValues...) }
func Warnw(msg string, keysAndValues ...any)  { log.Warnw(msg, keysAndValues...) }
func Errorw(msg string, keysAndValues ...any) { log.Errorw(msg, keysAndValues...) }
func Fatalw(msg string, keysAndValues ...any) { log.Fatalw(msg, keysAndValues...) }
