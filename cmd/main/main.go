package main

import (
	"log"
	"raspygk/cmd/parser"
	"raspygk/internal/config"
	"raspygk/internal/telegram"
	"raspygk/pkg/cache"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
	"raspygk/pkg/metrics"

	"go.uber.org/zap"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Перехваченная паника", zap.Any("panic", r))
		}
	}()

	// Подгрузка конфигурации
	configuration, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	// Инициализация логгепа
	logger.Init(configuration.LoggerLevel)

	// Подключение к БД
	err = db.Init(configuration.DBUser, configuration.DBPassword, configuration.DBHost, configuration.DBName)
	if err != nil {
		logger.Fatal("Ошибка при подключении к БД", zap.Error(err))
	}

	// Инициализация кэша Redis
	cache.Init(configuration.RedisAddr, configuration.RedisPort, configuration.RedisPassword)

	done := make(chan bool)
	go telegram.Init(configuration.TelegramAPI, done)
	<-done

	go metrics.Init()

	err = parser.Init()
	if err != nil {
		logger.Fatal("Ошибка в работе парсера", zap.Error(err))
	}
}
