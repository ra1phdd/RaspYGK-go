package metrics

import (
	"net/http"
	"raspygk/pkg/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func Init() {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe(":4000", mux)
	if err != nil {
		logger.Fatal("Ошибка при запуске HTTP-сервера", zap.Error(err))
	}
	logger.Debug("HTTP-сервер запущен")
}
