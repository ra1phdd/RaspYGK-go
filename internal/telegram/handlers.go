package telegram

import (
	"raspygk/internal/services"
	"raspygk/pkg/logger"
	"strconv"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func HandlerBot() {
	b.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return tele.HandlerFunc(func(c tele.Context) error {
			userState.chatID = c.Chat().ID
			userState.userID = c.Sender().ID
			userState.name = c.Sender().Username
			userState.ID, userState.group, userState.role, userState.push, userState.err = services.GetUserData(userState.userID)

			if (userState.role == "" || userState.group == "") && c.Text() != "/start" && c.Callback() == nil {
				c.Send("Чтобы начать пользоваться ботом, введите /start")
			}

			return next(c)
		})
	})

	b.Handle("/start", func(c tele.Context) error {
		menu, _ := Keyboards(0)
		c.Send("Добро пожаловать! Это неофициальный бот для получения расписания в ЯГК.", menu)
		if userState.ID == 0 {
			err := services.AddUser(userState.userID)
			if err != nil {
				logger.Error("ошибка добавления пользователя: ", zap.Error(err))
			}
		}
		if userState.role == "" {
			menu, btns := Keyboards(2)
			for _, key := range roleButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerRole(typeButtons, button.Text))
			}
			c.EditOrSend("Выберите роль:", menu)
		}
		if userState.group == "" {
			menu, btns := Keyboards(3)
			for _, key := range typeButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerType(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		}

		return nil
	})

	b.Handle("Расписание", func(c tele.Context) error {
		schedule, err := services.GetSchedule(userState.group)
		if err != nil {
			logger.Error("ошибка в функции getSchedule: ", zap.Error(err))
		}
		c.Send(schedule)

		return nil
	})

	b.Handle("Предыдущее", func(c tele.Context) error {
		schedule, err := services.GetPrevSchedule(userState.group)
		if err != nil {
			logger.Error("ошибка в функции getPrevSchedule: ", zap.Error(err))
		}
		c.Send(schedule)

		return nil
	})

	b.Handle("Следующее", func(c tele.Context) error {
		schedule, err := services.GetNextSchedule(userState.group)
		if err != nil {
			logger.Error("ошибка в функции getNextSchedule: ", zap.Error(err))
		}
		c.Send(schedule)

		return nil
	})

	b.Handle("Настройки", func(c tele.Context) error {
		menu, btns := Keyboards(1)
		buttons.Settings1 = btns["changeRole"]
		buttons.Settings2 = btns["changeGroup"]
		buttons.Settings3 = btns["changePUSH"]
		c.Send("Выберите настройку:", menu)

		b.Handle(&buttons.Settings1, func(c tele.Context) error {
			menu, btns := Keyboards(2)
			for _, key := range roleButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerRole(nil, button.Text))
			}
			c.EditOrSend("Выберите роль:", menu)

			return nil
		})

		b.Handle(&buttons.Settings2, func(c tele.Context) error {
			menu, btns := Keyboards(3)
			for _, key := range typeButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerType(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)

			return nil
		})

		b.Handle(&buttons.Settings3, func(c tele.Context) error {
			handler := createHandlerPUSH()
			err := handler(c)
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}

			return nil
		})

		return nil
	})

	b.Handle("Техподдержка", func(c tele.Context) error {
		c.Send("Введите текст сообщения:")

		b.Handle(tele.OnText, func(c tele.Context) error {
			err := SendToAdmin(c.Text(), userState.userID, userState.name, userState.group, userState.role)
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}
			c.Send("Ваше обращение будет обязательно рассмотрено (если оно дойдет)!")

			return nil
		})
		return nil
	})

	b.Handle("/message", func(c tele.Context) error {
		if userState.userID != 1230045591 {
			return nil
		}
		c.Send("Введи userID пидораса")

		b.Handle(tele.OnText, func(c tele.Context) error {
			if userState.userID != 1230045591 {
				return nil
			}
			userIDstr := c.Text()
			userID, err := strconv.ParseInt(userIDstr, 10, 64)
			if err != nil {
				logger.Error("Ошибка преобразования userID в /message", zap.Error(err))
				return err
			}

			c.Send("Введи текст для пидораса")

			b.Handle(tele.OnText, func(c tele.Context) error {
				if userState.userID != 1230045591 {
					return nil
				}
				chel := &tele.User{ID: userID}
				_, err := b.Send(chel, "Сообщение от администратора: "+c.Text())
				if err != nil {
					logger.Error("ошибка при отправке сообщения админу: ", zap.Error(err))
				}

				return nil
			})
			return nil
		})

		return nil
	})

	b.Handle("/messageall", func(c tele.Context) error {
		if userState.userID != 1230045591 {
			return nil
		}
		c.Send("Введи текст для пидорасов")

		b.Handle(tele.OnText, func(c tele.Context) error {
			if userState.userID != 1230045591 {
				return nil
			}
			err := SendToAll(c.Text())
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}

			return nil
		})

		return nil
	})

	b.Start()
}

func createHandlerRole(typeButtons []ButtonOption, text string) func(c tele.Context) error {
	return func(c tele.Context) error {
		err := services.UpdateRole(text, userState.userID)
		if err != nil {
			logger.Error("ошибка при обновлении роли в функции createHandlerRole: ", zap.Error(err))
			c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
		}
		if typeButtons != nil {
			menu, btns := Keyboards(3)
			for _, key := range typeButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerType(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		} else {
			c.EditOrSend("Готово!")
		}

		return nil
	}
}

func createHandlerType(text string) func(c tele.Context) error {
	return func(c tele.Context) error {
		switch text {
		case "ОИТ":
			menu, btns := Keyboards(4)
			for _, key := range groupOITButtons {
				button := btns[key]
				b.Handle(&button, createHandlerGroup(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		case "ОАР":
			menu, btns := Keyboards(4)
			for _, key := range groupOARButtons {
				button := btns[key]
				b.Handle(&button, createHandlerGroup(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		case "СО":
			menu, btns := Keyboards(4)
			for _, key := range groupSOButtons {
				button := btns[key]
				b.Handle(&button, createHandlerGroup(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		case "ММО":
			menu, btns := Keyboards(4)
			for _, key := range groupMMOButtons {
				button := btns[key]
				b.Handle(&button, createHandlerGroup(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		case "ОЭП":
			menu, btns := Keyboards(4)
			for _, key := range groupOEPButtons {
				button := btns[key]
				b.Handle(&button, createHandlerGroup(button.Text))
			}
			c.EditOrSend("Выберите группу:", menu)
		}
		return nil
	}
}

func createHandlerGroup(text string) func(c tele.Context) error {
	splitted := strings.Split(text, "/")
	return func(c tele.Context) error {
		err := services.UpdateGroup(splitted[0], userState.userID)
		if err != nil {
			logger.Error("ошибка при обновлении группы в функции createHandlerGroup: ", zap.Error(err))
			c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
		} else {
			c.EditOrSend("Готово!")
		}
		return nil
	}
}

func createHandlerPUSH() func(c tele.Context) error {
	return func(c tele.Context) error {
		var err error

		if userState.push == 1 {
			err = services.UpdatePUSH(0, userState.userID)
		} else {
			err = services.UpdatePUSH(1, userState.userID)
		}

		if err != nil {
			logger.Error("ошибка при обновлении статуса PUSH в функции createHandlerPUSH: ", zap.Error(err))
			c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
		} else {
			c.EditOrSend("Готово!")
		}

		return nil
	}
}
