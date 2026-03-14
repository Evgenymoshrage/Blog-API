package logger

import (
	"fmt"
	"os"
	"time"
)

type EventLogger struct {
	Events chan string // канал строк, через который приложение отправляет события для логирования
	file   *os.File    // файловый дескриптор, куда будут записываться события
}

func NewEventLogger(filename string) (*EventLogger, error) {

	// Открыли файл (создаем, если нет, только для записи), добавили строки в конце
	file, err := os.OpenFile(filename,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644) // владелец пишет, остальные читают

	if err != nil {
		return nil, err
	}

	logger := &EventLogger{ // создаем буферезированный канал на 100 сообщений
		Events: make(chan string, 100),
		file:   file,
	}

	go logger.worker() // запускаем отдельную горутину для записи

	return logger, nil
}

func (l *EventLogger) worker() {

	for event := range l.Events {

		time.Sleep(2 * time.Second) // таймаут

		line := fmt.Sprintf("[%s] %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			event)

		l.file.WriteString(line) // запись в файл
	}
}

func (l *EventLogger) Close() {
	close(l.Events) // закроет канал, горутина завершится
	l.file.Close()  // закроет файл
}
