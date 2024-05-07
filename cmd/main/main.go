package main

import (
	"log"
	"raspygk/internal/config"
	"raspygk/internal/parser"
	"raspygk/internal/telegram"
	"raspygk/pkg/cache"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
	"raspygk/pkg/metrics"
	"time"

	"go.uber.org/zap"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("panic occurred:", err)
		}
	}()

	// Подгрузка конфигурации
	config, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	// Инициализация логгепа
	logger.Init(config.LoggerLevel)

	// Подключение к БД
	err = db.Init(config.DBUser, config.DBPassword, config.DBHost, config.DBName)
	if err != nil {
		logger.Fatal("Ошибка при подключении к БД", zap.Error(err))
	}

	// Инициализация кэша Redis
	cache.Init(config.RedisAddr, config.RedisPort, config.RedisPassword)

	go telegram.Init(config.TelegramAPI)

	go metrics.Init()

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err = parser.Init()
		if err != nil {
			logger.Fatal("ошибка в работе парсера: ", zap.Error(err))
		}
	}
}
