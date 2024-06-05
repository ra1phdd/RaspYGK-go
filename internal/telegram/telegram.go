package telegram

import (
	"raspygk/internal/models"
	"raspygk/pkg/logger"
	"time"

	"go.uber.org/zap"

	tele "gopkg.in/telebot.v3"
)

var (
	b         *tele.Bot
	userState models.UserState
	buttons   models.Buttons
)

func Init(TelegramAPI string, done chan bool) {
	pref := tele.Settings{
		Token:  TelegramAPI,
		Poller: &tele.LongPoller{Timeout: 1 * time.Second},
	}

	var err error
	b, err = tele.NewBot(pref)
	if err != nil {
		logger.Error("ошибка создания бота: ", zap.Error(err))
	}

	done <- true

	HandlerBot()
}
