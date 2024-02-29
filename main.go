package main

import (
	"fmt"
	"go.uber.org/zap"
	"time"
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
	atomicLevel.SetLevel(zap.WarnLevel)
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

	go TelegramBot()

	err := Parser()
	if err != nil {
		logger.Fatal("ошибка в работе парсера: ", zap.Error(err))
	}

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := Parser()
		if err != nil {
			logger.Fatal("ошибка в работе парсера: ", zap.Error(err))
		}
	}
}

func handleError(message string, err error) error {
	logger.Error(message, zap.Error(err))
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return nil
}
