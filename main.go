package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	// Перехватываем панику, чтобы бот не ложился от любой ошибки
	defer func() {
		if r := recover(); r != nil {
			logger.Fatal("паника в работе программы: ", zap.Any("r", r))
		}
	}()

	// Логгирование
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.DebugLevel)
	config := zap.Config{
		Level:       atomicLevel,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, _ = config.Build()
	defer logger.Sync()

	ConnectDB()

	err := Parser()
	if err != nil {
		logger.Fatal("ошибка в работе парсера: ", zap.Error(err))
	}

	go TelegramBot()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := Parser()
		if err != nil {
			logger.Fatal("ошибка в работе парсера: ", zap.Error(err))
		}
		data, err := getUserPUSH(1)
		if err != nil {
			logger.Fatal("ошибка в работе пуша: ", zap.Error(err))
		}
		logger.Info("Отправка PUSH-уведомлений", zap.Any("data", data))
	}
}

func handleError(message string, err error) error {
	logger.Error(message, zap.Error(err))
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return nil
}
