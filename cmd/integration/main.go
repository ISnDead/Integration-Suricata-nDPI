package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/runner"

	"go.uber.org/zap"
)

func main() {
	// Инициализация Zap логгера (находится в pkg/logger согласно архитектуре) [cite: 2026-01-07]
	logger.Init()
	// Гарантируем сброс буфера логов при выходе
	defer logger.Sync()

	logger.Log.Info("Запуск микросервиса интеграции Suricata + nDPI")

	// Создаем контекст, который отменится при нажатии Ctrl+C или сигнале SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Инициализируем Runner
	srv := runner.NewRunner()

	// Запускаем основной цикл.
	// Наш Runner сам умеет ждать ctx.Done() и вызывать Stop() внутри себя.
	if err := srv.Start(ctx); err != nil {
		logger.Log.Fatal("Фатальная ошибка при работе микросервиса", zap.Error(err))
	}

	logger.Log.Info("Микросервис успешно завершил работу")
}
