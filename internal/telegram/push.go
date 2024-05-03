package telegram

import (
	"fmt"
	"log"
	"raspygk/internal/services"
	"raspygk/pkg/logger"
	"strconv"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func SendToPush(idshift int) error {
	data, err := services.GetUserPUSH(idshift, false)
	if err != nil {
		logger.Error("ошибка при получении пользователей с включенными PUSH в функции SendToPush: ", zap.Error(err))
	}

	for _, item := range data {
		schedule, err := services.GetSchedule(fmt.Sprint(item[1]))
		if err != nil {
			logger.Error("ошибка при получении расписания в функции createHandlerPUSH: ", zap.Error(err))
		}

		logger.Info("userid", zap.Any("userid", item[0]), zap.Any("group", item[1]), zap.Any("schedule", schedule))
		userid := item[0]
		userID, err := strconv.ParseInt(userid, 10, 64)
		if err != nil {
			// Обработка ошибки преобразования
			log.Printf("Ошибка преобразования userid: %v", err)
			continue
		}
		chel := &tele.User{ID: userID}
		go func() {
			for {
				time.Sleep(3 * time.Second)
				if b != nil {
					_, err := b.Send(chel, schedule)
					if err != nil {
						logger.Error("ошибка при отправке PUSH-уведомления пользователю: ", zap.Error(err))
					}
					break
				} else {
					logger.Error("объект Bot не инициализирован")
				}
			}
		}()
	}

	return nil
}

func SendToAdmin(text string, userID int64, username string, group string, role string) error {
	chel := &tele.User{ID: 1230045591}
	_, err := b.Send(chel, "Сообщение от пидораса с ID "+fmt.Sprint(userID)+" ("+fmt.Sprint(username)+") из группы "+group+": "+text)
	if err != nil {
		logger.Error("ошибка при отправке сообщения админу: ", zap.Error(err))
	}

	return nil
}

func SendToAll(text string) error {
	data, err := services.GetUserPUSH(0, true)
	if err != nil {
		logger.Error("ошибка при получении пользователей с включенными PUSH в функции SendToPush: ", zap.Error(err))
	}

	for _, item := range data {
		logger.Info("userid", zap.Any("userid", item[0]), zap.Any("group", item[1]))
		userid := item[0]
		userID, err := strconv.ParseInt(userid, 10, 64)
		if err != nil {
			// Обработка ошибки преобразования
			log.Printf("Ошибка преобразования userid: %v", err)
			continue
		}
		chel := &tele.User{ID: userID}
		for {
			time.Sleep(3 * time.Second)
			if b != nil {
				_, err := b.Send(chel, text)
				if err != nil {
					logger.Error("ошибка при отправке PUSH-уведомления пользователю: ", zap.Error(err))
				}
				break
			} else {
				logger.Error("объект Bot не инициализирован")
			}
		}
	}

	return nil
}
