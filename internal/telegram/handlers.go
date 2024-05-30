package telegram

import (
	"fmt"
	"raspygk/internal/models"
	"raspygk/internal/services"
	"raspygk/pkg/logger"
	"strconv"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type State int

const (
	StateNone State = iota
	StateTextSupport
	StateGetIDMessage
	StateTextMessage
	StateAllMessage
)

var userStates = make(map[int64]State)
var SendUserIDMessage int64

func HandlerBot() {
	b.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			var err error
			userState.ChatID = c.Chat().ID
			userState.UserID = c.Sender().ID
			userState.Name = c.Sender().Username

			userState.Group, userState.Role, userState.Push, err = services.GetUserData(userState.UserID)
			if err != nil {
				logger.Error("ошибка при получении данных пользователя: ", zap.Error(err))
				return err
			}

			if (userState.Role == 0 || userState.Group == 0) && c.Text() != "/start" && c.Callback() == nil {
				err := c.Send("Чтобы начать пользоваться ботом, введите /start")
				if err != nil {
					return err
				}
			}

			return next(c)
		}
	})

	b.Handle("/start", func(c tele.Context) error {
		userStates[c.Sender().ID] = StateNone

		menu, _ := Keyboards(0, "")
		err := c.Send("Добро пожаловать! Это неофициальный бот для получения расписания в ЯГК.", menu)
		if err != nil {
			return err
		}

		if userState.Group == 0 && userState.Role == 0 {
			err := services.AddUser(userState.UserID)
			if err != nil {
				logger.Error("ошибка добавления пользователя: ", zap.Error(err))
			}
		}
		if userState.Role == 0 {
			menu, btns := Keyboards(2, "")
			for _, key := range models.RoleButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerRole(models.TypeButtons, button.Text))
			}
			err := c.EditOrSend("Выберите роль:", menu)
			if err != nil {
				return err
			}
			return nil
		}
		if userState.Group == 0 {
			menu, btns := Keyboards(3, "")
			for _, key := range models.TypeButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerType(button.Text))
			}
			err := c.EditOrSend("Выберите группу:", menu)
			if err != nil {
				return err
			}
		}

		return nil
	})

	b.Handle("Расписание", func(c tele.Context) error {
		userStates[c.Sender().ID] = StateNone

		schedule, err := services.GetSchedule(userState.Group)
		if err != nil {
			logger.Error("ошибка в функции getSchedule: ", zap.Error(err))
		}
		err = c.Send(schedule)
		if err != nil {
			return err
		}

		return nil
	})

	b.Handle("Предыдущее", func(c tele.Context) error {
		userStates[c.Sender().ID] = StateNone

		schedule, err := services.GetPrevSchedule(userState.Group)
		if err != nil {
			logger.Error("ошибка в функции getPrevSchedule: ", zap.Error(err))
		}
		err = c.Send(schedule)
		if err != nil {
			return err
		}

		return nil
	})

	b.Handle("Следующее", func(c tele.Context) error {
		userStates[c.Sender().ID] = StateNone

		schedule, err := services.GetNextSchedule(userState.Group)
		if err != nil {
			logger.Error("ошибка в функции getNextSchedule: ", zap.Error(err))
		}
		err = c.Send(schedule)
		if err != nil {
			return err
		}

		return nil
	})

	b.Handle("Настройки", func(c tele.Context) error {
		userStates[c.Sender().ID] = StateNone

		menu, btns := Keyboards(1, "")
		buttons.Settings1 = btns["changeRole"]
		buttons.Settings2 = btns["changeGroup"]
		buttons.Settings3 = btns["changePUSH"]
		err := c.Send("Выберите настройку:", menu)
		if err != nil {
			return err
		}

		b.Handle(&buttons.Settings1, func(c tele.Context) error {
			menu, btns := Keyboards(2, "")
			for _, key := range models.RoleButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerRole(nil, button.Text))
			}
			err := c.EditOrSend("Выберите роль:", menu)
			if err != nil {
				return err
			}

			return nil
		})

		b.Handle(&buttons.Settings2, func(c tele.Context) error {
			menu, btns := Keyboards(3, "")
			for _, key := range models.TypeButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerType(button.Text))
			}
			err := c.EditOrSend("Выберите группу:", menu)
			if err != nil {
				return err
			}

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
		userStates[c.Sender().ID] = StateTextSupport
		err := c.Send("Введите текст сообщения:")
		if err != nil {
			return err
		}

		return nil
	})

	fmt.Println(buttons.ReplyAdmin)

	b.Handle(&buttons.ReplyAdmin, func(c tele.Context) error {
		userStates[c.Sender().ID] = StateTextMessage

		logger.Info("Обработчик кнопки вызван")

		var err error
		SendUserIDMessage, err = strconv.ParseInt(c.Data(), 10, 64)
		if err != nil {
			logger.Error("Ошибка преобразования userID в /message", zap.Error(err))
		}

		err = c.Send("Введи текст для пидораса")
		if err != nil {
			return err
		}

		return nil
	})

	b.Handle("/message", func(c tele.Context) error {
		if userState.UserID != 1230045591 {
			return nil
		}

		userStates[c.Sender().ID] = StateGetIDMessage

		err := c.Send("Введи userID пидораса")
		if err != nil {
			return err
		}

		return nil
	})

	b.Handle("/messageall", func(c tele.Context) error {
		if userState.UserID != 1230045591 {
			return nil
		}

		userStates[c.Sender().ID] = StateAllMessage

		err := c.Send("Введи текст для пидорасов")
		if err != nil {
			return err
		}

		return nil
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		state := userStates[c.Sender().ID]

		switch state {
		case StateTextSupport:
			err := SendToAdmin(c.Text(), c.Sender().ID, userState.Name, userState.Group, userState.Role)
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}

			userStates[c.Sender().ID] = StateNone

			err = c.Send("Ваше обращение будет обязательно рассмотрено!")
			if err != nil {
				return err
			}
		case StateGetIDMessage:
			if userState.UserID != 1230045591 {
				break
			}

			var err error

			userIDstr := c.Text()
			SendUserIDMessage, err = strconv.ParseInt(userIDstr, 10, 64)
			if err != nil {
				logger.Error("Ошибка преобразования userID в /message", zap.Error(err))
				break
			}

			userStates[c.Sender().ID] = StateTextMessage

			err = c.Send("Введи текст для пидораса")
			if err != nil {
				return err
			}
		case StateTextMessage:
			if userState.UserID != 1230045591 {
				break
			}
			chel := &tele.User{ID: SendUserIDMessage}
			_, err := b.Send(chel, "Сообщение от администратора: "+c.Text())
			if err != nil {
				logger.Error("ошибка при отправке сообщения админу: ", zap.Error(err))
			}

			userStates[c.Sender().ID] = StateNone
		case StateAllMessage:
			if userState.UserID != 1230045591 {
				break
			}
			err := SendToAll(c.Text())
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}

			userStates[c.Sender().ID] = StateNone
		default:
			userStates[c.Sender().ID] = StateNone

			err := c.Send("Неизвестная команда")
			if err != nil {
				return err
			}
		}

		return nil
	})

	b.Start()
}

func createHandlerRole(typeButtons []models.ButtonOption, text string) func(c tele.Context) error {
	return func(c tele.Context) error {
		var role int
		switch text {
		case "Студент":
			role = 1
		case "Преподаватель":
			role = 2
		}
		err := services.UpdateRole(role, userState.UserID)
		if err != nil {
			logger.Error("ошибка при обновлении роли в функции createHandlerRole: ", zap.Error(err))
			err := c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
			if err != nil {
				return err
			}
		}
		if typeButtons != nil {
			menu, btns := Keyboards(3, "")
			for _, key := range typeButtons {
				button := btns[key.Value]
				b.Handle(&button, createHandlerType(button.Text))
			}
			err := c.EditOrSend("Выберите группу:", menu)
			if err != nil {
				return err
			}
		} else {
			err := c.EditOrSend("Готово!")
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func createHandlerType(text string) func(c tele.Context) error {
	return func(c tele.Context) error {
		menu, btns := Keyboards(4, "")
		var groupButtons []models.ButtonOption

		switch text {
		case "ОИТ":
			groupButtons = models.GroupOITButtons
		case "ОАР":
			groupButtons = models.GroupOARButtons
		case "СО":
			groupButtons = models.GroupSOButtons
		case "ММО":
			groupButtons = models.GroupMMOButtons
		case "ОЭП":
			groupButtons = models.GroupOEPButtons
		}

		for _, key := range groupButtons {
			button := btns[key.Value]
			b.Handle(&button, createHandlerGroup(button.Text))
		}

		err := c.EditOrSend("Выберите группу:", menu)
		if err != nil {
			return err
		}

		return nil
	}
}

func createHandlerGroup(text string) func(c tele.Context) error {
	return func(c tele.Context) error {
		err := services.UpdateGroup(text, userState.UserID)
		if err != nil {
			err := c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
			if err != nil {
				return err
			}
			logger.Error("ошибка при обновлении группы в функции createHandlerGroup: ", zap.Error(err))
		} else {
			err := c.EditOrSend("Готово!")
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func createHandlerPUSH() func(c tele.Context) error {
	return func(c tele.Context) error {
		var err error

		if userState.Push {
			err = services.UpdatePUSH(false, userState.UserID)
		} else {
			err = services.UpdatePUSH(true, userState.UserID)
		}

		if err != nil {
			err := c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
			if err != nil {
				return err
			}
			logger.Error("ошибка при обновлении статуса PUSH в функции createHandlerPUSH: ", zap.Error(err))
		} else {
			err := c.EditOrSend("Готово!")
			if err != nil {
				return err
			}
		}

		return nil
	}
}
