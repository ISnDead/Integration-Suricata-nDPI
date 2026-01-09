package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/runner"

	"go.uber.org/zap"
)

func main() {
	// Инициализация логгера
	logger.Init()
	defer logger.Sync()

	configPath := flag.String("config", "config/config.yaml", "Путь к файлу конфигурации")
	flag.Parse()

	logger.Log.Info("Запуск микросервиса интеграции Suricata + nDPI",
		zap.String("config", *configPath),
	)

	// Контекст отменится при Ctrl+C или SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := runner.NewRunner()

	if err := srv.Start(ctx, *configPath); err != nil {
		logger.Log.Fatal("Сервис завершился с ошибкой", zap.Error(err))
	}

	logger.Log.Info("Микросервис завершил работу")
}
