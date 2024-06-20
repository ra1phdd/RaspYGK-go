package parser

import (
	"log"
	"raspygk/internal/parser"
	"time"
)

func Init() error {
	// Инициализация парсера и метрик
	err := parser.Start()
	if err != nil {
		log.Fatalf("Ошибка в работе парсера: %v", err)
	}

	// Определение времени начала и интервала для разных временных периодов
	type schedulePeriod struct {
		startTime time.Time
		endTime   time.Time
		interval  time.Duration
	}

	schedule := []schedulePeriod{
		{
			startTime: parseTime("09:00"),
			endTime:   parseTime("11:00"),
			interval:  15 * time.Minute,
		},
		{
			startTime: parseTime("11:00"),
			endTime:   parseTime("12:30"),
			interval:  5 * time.Minute,
		},
		{
			startTime: parseTime("12:30"),
			endTime:   parseTime("18:00"),
			interval:  time.Hour,
		},
		// Необходимо добавить любой неактивный интервал (18:00 до 09:00 следующего дня), но его не нужно обрабатывать
		{
			startTime: parseTime("18:00"),
			endTime:   parseTime("09:00").AddDate(0, 0, 1),
			interval:  0,
		},
	}

	// Запуск цикла для выбранного времени
	for {
		now := time.Now()
		var currentInterval time.Duration

		// Поиск текущего интервала по расписанию
		for _, period := range schedule {
			if now.After(period.startTime) && now.Before(period.endTime) {
				currentInterval = period.interval
				break
			}
		}

		// Если нашли интервал, запускаем таймер
		if currentInterval > 0 {
			timer := time.NewTicker(currentInterval)
			defer timer.Stop()

			for range timer.C {
				err := parser.Start()
				if err != nil {
					log.Fatalf("Ошибка в работе парсера: %v", err)
				}
			}
		} else {
			// Если не нашли подходящий интервал, ждем 1 минуту перед повторной проверкой
			time.Sleep(time.Minute)
		}
	}
}

func parseTime(strTime string) time.Time {
	// Парсинг времени в формате "HH:MM"
	t, err := time.Parse("15:04", strTime)
	if err != nil {
		log.Fatalf("Ошибка парсинга времени: %v", err)
	}
	return t
}
