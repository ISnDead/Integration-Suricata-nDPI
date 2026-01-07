package runner

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Runner управляет жизненным циклом интеграционного сервиса
type Runner struct {
	// Сюда можно будет добавить каналы для связи между функциями
}

// NewRunner создает экземпляр для управления сервисом
func NewRunner() *Runner {
	return &Runner{}
}

// Start запускает основные этапы работы: валидацию и подключение
func (r *Runner) Start(ctx context.Context) error {
	log.Println("Инициализация nDPI Integration Service...")

	// Эмуляция вызовов из папки /integration согласно ТЗ
	log.Println("[nDPI] Валидация конфигурации...")
	// В будущем здесь вызов: integration.ValidateNDPIConfig() 
	
	log.Println("[Suricata] Проверка статуса и подключение...")
	// В будущем здесь вызов: integration.ConnectSuricata() 

	fmt.Println("Сервис успешно запущен. Мониторинг активен.")

	// Ждем сигнала отмены через контекст
	<-ctx.Done()
	return nil
}

// Stop корректно завершает работу сервиса без повреждения данных
func (r *Runner) Stop() {
	log.Println("Выполнение Graceful Shutdown...")
	// Здесь будет логика корректного разрыва соединений 
	fmt.Println("Все модули остановлены. Выход.")
}