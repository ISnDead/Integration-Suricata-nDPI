package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	
	"integration-suricata-ndpi/pkg/runner"
)

func main() {
	log.Println("Запуск интеграционного сервиса Suricata + nDPI")

	// Настраиваем перехват сигналов для безопасной остановки 
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Инициализация раннера
	srv := runner.NewRunner()

	// Запуск в горутине, чтобы не блокировать ожидание системных сигналов
	go func() {
		if err := srv.Start(ctx); err != nil {
			log.Fatalf("Критическая ошибка при работе: %v", err)
		}
	}()

	// Ожидание сигнала (Ctrl+C или завершение процесса в Docker)
	<-ctx.Done()
	
	log.Println("Получен сигнал завершения. Останавливаем сервис...")
	srv.Stop()
}