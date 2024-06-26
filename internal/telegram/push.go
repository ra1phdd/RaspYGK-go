package telegram

import (
	"fmt"
	"raspygk/internal/services"
	"raspygk/pkg/cache"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func SendToPush(idShift int) error {
	data, err := services.GetUserPUSH(idShift, nil, false)
	if err != nil {
		logger.Error("ошибка при получении пользователей с включенными PUSH в функции SendToPush: ", zap.Error(err))
	}

	for _, item := range data {
		groupID, errSchedule := strconv.Atoi(item[1])
		if errSchedule != nil {
			logger.Error("ошибка при преобразовании строки в int в функции createHandlerPUSH: ", zap.Error(err))
		}

		schedule, errSchedule := services.GetSchedule(groupID, 0)
		if errSchedule != nil {
			logger.Error("ошибка при получении расписания в функции createHandlerPUSH: ", zap.Error(err))
		}

		dataSchedule, errSchedule := cache.Rdb.Get(cache.Ctx, "GetTextSchedulePUSH_"+fmt.Sprint(groupID)).Result()
		if errSchedule == nil && dataSchedule == schedule {
			return nil
		} else {
			errSchedule = cache.Rdb.Set(cache.Ctx, "GetTextSchedulePUSH_"+fmt.Sprint(groupID), schedule, 24*time.Hour).Err()
			if errSchedule != nil {
				return errSchedule
			}
		}

		logger.Info("userid", zap.Any("userid", item[0]), zap.Any("group", item[1]), zap.Any("schedule", schedule))
		userID, errSchedule := strconv.Atoi(item[0])
		if errSchedule != nil {
			logger.Error("ошибка при преобразовании строки в int в функции createHandlerPUSH: ", zap.Error(err))
		}
		chel := &tele.User{ID: int64(userID)}
		_, err = b.Send(chel, schedule)
		if err != nil && !strings.Contains(error.Error(err), "bot was blocked") && !strings.Contains(error.Error(err), "chat not found") {
			logger.Error("ошибка при отправке PUSH-уведомления пользователю: ", zap.Error(err))
		}
	}

	return nil
}

func SendToAdmin(text string, userID int64, username string, idGroup int, role int) error {
	var group string
	err := db.Conn.QueryRowx(`SELECT name FROM groups WHERE id = $1`, idGroup).Scan(&group)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы groups в функции UpdateGroup", zap.Error(err))
		return err
	}

	chel := &tele.User{ID: 1230045591}

	var roleText string
	if role == 1 {
		roleText = "пидораса"
	} else if role == 2 {
		roleText = "преподавателя"
	}

	menu, btns := Keyboards(5, fmt.Sprint(userID))
	buttons.ReplyAdmin = btns["replyAdmin"]

	_, err = b.Send(chel, "Сообщение от "+roleText+" с ID "+fmt.Sprint(userID)+" ("+fmt.Sprint(username)+") из группы "+group+": "+text, menu)
	if err != nil {
		logger.Error("ошибка при отправке сообщения админу: ", zap.Error(err))
	}

	return nil
}

func SendToAll(text string) error {
	data, err := services.GetUserPUSH(0, nil, true)
	if err != nil {
		logger.Error("ошибка при получении пользователей с включенными PUSH в функции SendToPush: ", zap.Error(err))
	}

	for _, item := range data {
		userID, err := strconv.Atoi(item[0])
		if err != nil {
			logger.Error("ошибка при преобразовании строки в int в функции createHandlerPUSH: ", zap.Error(err))
		}

		chel := &tele.User{ID: int64(userID)}
		_, err = b.Send(chel, text)
		if err != nil {
			logger.Error("ошибка при отправке PUSH-уведомления пользователю: ", zap.Error(err))
		}
	}

	return nil
}
