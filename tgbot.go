package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	tele "gopkg.in/telebot.v3"
)

type UserState struct {
	ID     int
	chatID int64
	userID int64
	group  string
	role   string
	push   int
	err    error
}

type Buttons struct {
	Settings1 tele.Btn
	Settings2 tele.Btn
	Settings3 tele.Btn
}

var (
	b               *tele.Bot
	TelegramAPI     string
	userState       UserState
	buttons         Buttons
	roleButtons     = []string{"Student", "Teacher"}
	typeButtons     = []string{"OIT", "OAR", "SO", "MMO", "OEP"}
	groupOITButtons = []string{"IS1_11", "IS1_13", "IS1_15", "IS1_21",
		"IS1_23", "IS1_25", "IS1_31", "IS1_33",
		"IS1_35", "IS1_41", "IS1_43", "IS1_45",
		"SA1_11", "SA1_21", "SA1_31", "SA1_41",
		"IB1_11", "IB1_21", "IB1_41", "IB1_21"}
	groupOARButtons         = []string{}
	groupSOButtons          = []string{}
	groupMMOButtons         = []string{}
	groupOEPButtons         = []string{}
	isHelpCommandSent       bool
	isAnswerTextCommandSent bool
	isAnswerCommandSent     bool
	isAnswerAllCommandSent  bool
)

func TelegramBot() {
	TelegramAPI = os.Getenv("TELEGRAM_API")

	pref := tele.Settings{
		Token:  TelegramAPI,
		Poller: &tele.LongPoller{Timeout: 1 * time.Second},
	}

	var err error
	b, err = tele.NewBot(pref)
	if err != nil {
		logger.Error("ошибка создания бота: ", zap.Error(err))
	}

	HandlerBot()
}

func HandlerBot() {
	b.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return tele.HandlerFunc(func(c tele.Context) error {
			userState.chatID = c.Chat().ID
			userState.userID = c.Sender().ID
			userState.ID, userState.group, userState.role, userState.push, userState.err = getUserData(userState.userID)

			if (userState.role == "" || userState.group == "") && c.Text() != "/start" && c.Callback() == nil {
				c.Send("Чтобы начать пользоваться ботом, введите /start")
			}

			return next(c)
		})
	})

	b.Handle("/start", func(c tele.Context) error {
		go func() {
			menu, _ := Keyboards(0)
			c.Send("Добро пожаловать! Это неофициальный бот для получения расписания в ЯГК.", menu)
			if userState.ID == 0 {
				err := AddUser(userState.userID)
				if err != nil {
					logger.Error("ошибка добавления пользователя: ", zap.Error(err))
				}
			}
			if userState.role == "" {
				menu, btns := Keyboards(2)
				for _, key := range roleButtons {
					button := btns[key]
					b.Handle(&button, createHandlerRole(typeButtons, button.Text))
				}
				c.EditOrSend("Выберите роль:", menu)
			}
			if userState.group == "" {
				menu, btns := Keyboards(3)
				for _, key := range typeButtons {
					button := btns[key]
					b.Handle(&button, createHandlerType(button.Text))
				}
				c.EditOrSend("Выберите группу:", menu)
			}
		}()
		return nil
	})

	b.Handle("Расписание", func(c tele.Context) error {
		go func() {
			schedule, err := getSchedule(userState.group)
			if err != nil {
				logger.Error("ошибка в функции getSchedule: ", zap.Error(err))
			}
			c.Send(schedule)
		}()
		return nil
	})

	b.Handle("Предыдущее", func(c tele.Context) error {
		go func() {
			schedule, err := getPrevSchedule(userState.group)
			if err != nil {
				logger.Error("ошибка в функции getPrevSchedule: ", zap.Error(err))
			}
			c.Send(schedule)
		}()
		return nil
	})

	b.Handle("Следующее", func(c tele.Context) error {
		go func() {
			schedule, err := getNextSchedule(userState.group)
			if err != nil {
				logger.Error("ошибка в функции getNextSchedule: ", zap.Error(err))
			}
			c.Send(schedule)
		}()
		return nil
	})

	b.Handle("Настройки", func(c tele.Context) error {
		go func() {
			menu, btns := Keyboards(1)
			buttons.Settings1 = btns["changeRole"]
			buttons.Settings2 = btns["changeGroup"]
			buttons.Settings3 = btns["changePUSH"]
			c.Send("Выберите настройку:", menu)

			b.Handle(&buttons.Settings1, func(c tele.Context) error {
				menu, btns := Keyboards(2)
				for _, key := range roleButtons {
					button := btns[key]
					b.Handle(&button, createHandlerRole(nil, button.Text))
				}
				c.EditOrSend("Выберите роль:", menu)

				return nil
			})

			b.Handle(&buttons.Settings2, func(c tele.Context) error {
				menu, btns := Keyboards(3)
				for _, key := range typeButtons {
					button := btns[key]
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
		}()
		return nil
	})

	b.Handle("Техподдержка", func(c tele.Context) error {
		go func() {
			c.Send("Введите текст сообщения: ")
			isHelpCommandSent = true
		}()
		return nil
	})

	b.Handle("/message", func(c tele.Context) error {
		c.Send("Введи userID пидораса")
		isAnswerTextCommandSent = true
		return nil
	})

	b.Handle("/messageall", func(c tele.Context) error {
		c.Send("Введи текст для пидорасов")
		isAnswerAllCommandSent = true
		return nil
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		var userID int64
		var text string
		var err error
		if isHelpCommandSent {
			err := SendToAdmin(c.Text(), userState.userID, userState.group, userState.role)
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}
			c.Send("Ваше обращение будет обязательно рассмотрено (если оно дойдет)!")
			isHelpCommandSent = false
		}
		if isAnswerTextCommandSent {
			userid := c.Text()
			userID, err = strconv.ParseInt(userid, 10, 64)
			if err != nil {
				// Обработка ошибки преобразования
				log.Printf("Ошибка преобразования userid: %v", err)
			}
			c.Send("Введи текст для пидораса")
			isAnswerTextCommandSent = false
			isAnswerCommandSent = true
			b.Handle(tele.OnText, func(c tele.Context) error {
				if isAnswerCommandSent {
					text = c.Text()
					chel := &tele.User{ID: userID}
					_, err := b.Send(chel, "Сообщение от администратора: "+text)
					if err != nil {
						logger.Error("ошибка при отправке сообщения админу: ", zap.Error(err))
					}
					isAnswerCommandSent = false
				}
				return nil
			})
		}
		if isAnswerAllCommandSent {
			text = c.Text()
			err := SendToAll(text)
			if err != nil {
				logger.Error("ошибка при обработке HTTP-запроса в обработчике PUSH: ", zap.Error(err))
			}
			isAnswerAllCommandSent = false
		}
		return nil
	})

	b.Start()
}

func Keyboards(keyboard int) (*tele.ReplyMarkup, map[string]tele.Btn) {
	btns := make(map[string]tele.Btn)

	switch keyboard {
	case 0:
		menu := &tele.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}
		btnSchedule := menu.Text("Расписание")
		btnPrev := menu.Text("Предыдущее")
		btnNext := menu.Text("Следующее")
		btnSettings := menu.Text("Настройки")
		btnSupport := menu.Text("Техподдержка")

		menu.Reply(
			menu.Row(btnSchedule),
			menu.Row(btnPrev, btnNext),
			menu.Row(btnSettings, btnSupport),
		)

		return menu, nil

	case 1:
		var buttonPush string
		if userState.push == 1 {
			buttonPush = "✅"
		} else {
			buttonPush = "❌"
		}
		menu := &tele.ReplyMarkup{}
		btns["changeRole"] = menu.Data("Смена роли", "role")
		btns["changeGroup"] = menu.Data("Смена группы", "group")
		btns["changePUSH"] = menu.Data("Получение PUSH-уведомлений "+buttonPush, "push")

		menu.Inline(
			menu.Row(btns["changeRole"], btns["changeGroup"]),
			menu.Row(btns["changePUSH"]),
		)

		return menu, btns

	case 2:
		menu := &tele.ReplyMarkup{}
		btns["Student"] = menu.Data("Студент", "Student")
		btns["Teacher"] = menu.Data("Преподаватель", "Teacher")

		menu.Inline(
			menu.Row(btns["Student"], btns["Teacher"]),
		)

		return menu, btns

	case 3:
		menu := &tele.ReplyMarkup{}
		btns["OIT"] = menu.Data("ОИТ", "OIT")
		btns["OAR"] = menu.Data("ОАР", "OAR")
		btns["SO"] = menu.Data("СО", "SO")
		btns["MMO"] = menu.Data("ММО", "MMO")
		btns["OEP"] = menu.Data("ОЭП", "OEP")

		menu.Inline(
			menu.Row(btns["OIT"], btns["OAR"]),
			menu.Row(btns["SO"], btns["MMO"]),
			menu.Row(btns["OEP"]),
		)

		return menu, btns

	case 4:
		menu := &tele.ReplyMarkup{}

		btns["IS1_11"] = menu.Data("ИС1-11/12", "IS1_11")
		btns["IS1_13"] = menu.Data("ИС1-13/14", "IS1_13")
		btns["IS1_15"] = menu.Data("ИС1-15/16", "IS1_15")
		btns["IS1_21"] = menu.Data("ИС1-21/22/ИС2-11", "IS1_21")
		btns["IS1_23"] = menu.Data("ИС1-23/24/ИС2-12", "IS1_23")
		btns["IS1_25"] = menu.Data("ИС1-25/26/ИС2-13", "IS1_25")
		btns["IS1_31"] = menu.Data("ИС1-31/32/ИС2-21", "IS1_31")
		btns["IS1_33"] = menu.Data("ИС1-33/34/ИС2-22", "IS1_33")
		btns["IS1_35"] = menu.Data("ИС1-35/ИС2-23", "IS1_35")
		btns["IS1_41"] = menu.Data("ИС1-41/42/ИС2-31", "IS1_41")
		btns["IS1_43"] = menu.Data("ИС1-43/ИС2-32", "IS1_43")
		btns["IS1_45"] = menu.Data("ИС1-45/ИС2-33", "IS1_45")

		btns["SA1_11"] = menu.Data("СА1-11/12", "SA1_11")
		btns["SA1_21"] = menu.Data("СА1-21/22/СА2-11", "SA1_21")
		btns["SA1_31"] = menu.Data("СА1-31/32/СА2-21", "SA1_31")
		btns["SA1_41"] = menu.Data("СА1-41/42/СА2-31", "SA1_41")

		btns["IB1_11"] = menu.Data("ИБ1-11/12", "IB1_11")
		btns["IB1_21"] = menu.Data("ИБ1-21/22/ИБ2-11", "IB1_21")
		btns["IB1_31"] = menu.Data("ИБ1-31/32/ИБ2-21", "IB1_31")
		btns["IB1_41"] = menu.Data("ИБ1-41/42/ИБ2-31", "IB1_41")

		menu.Inline(
			menu.Row(btns["IS1_11"], btns["IS1_13"]),
			menu.Row(btns["IS1_15"], btns["IS1_21"]),
			menu.Row(btns["IS1_23"], btns["IS1_25"]),
			menu.Row(btns["IS1_31"], btns["IS1_33"]),
			menu.Row(btns["IS1_35"], btns["IS1_41"]),
			menu.Row(btns["IS1_43"], btns["IS1_45"]),
			menu.Row(btns["SA1_11"], btns["SA1_21"]),
			menu.Row(btns["SA1_31"], btns["SA1_41"]),
			menu.Row(btns["IB1_11"], btns["IB1_21"]),
			menu.Row(btns["IB1_31"], btns["IB1_41"]),
		)

		return menu, btns

	default:
		return nil, nil
	}
}

func createHandlerRole(typeButtons []string, text string) func(c tele.Context) error {
	return func(c tele.Context) error {
		err := UpdateRole(text, userState.userID)
		if err != nil {
			logger.Error("ошибка при обновлении роли в функции createHandlerRole: ", zap.Error(err))
			c.EditOrSend("Произошла ошибка! Повторите попытку позже или обратитесь в техподдержку")
		}
		if typeButtons != nil {
			menu, btns := Keyboards(3)
			for _, key := range typeButtons {
				button := btns[key]
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
		err := UpdateGroup(splitted[0], userState.userID)
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
			err = UpdatePUSH(0, userState.userID)
		} else {
			err = UpdatePUSH(1, userState.userID)
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

func SendToPush(idshift int) error {
	data, err := getUserPUSH(idshift, false)
	if err != nil {
		logger.Error("ошибка при получении пользователей с включенными PUSH в функции SendToPush: ", zap.Error(err))
	}

	for _, item := range data {
		schedule, err := getSchedule(fmt.Sprint(item[1]))
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
		if err != nil {
			logger.Error("ошибка при отправке PUSH-уведомления пользователю: ", zap.Error(err))
		}
	}

	return nil
}

func SendToAdmin(text string, userID int64, group string, role string) error {
	chel := &tele.User{ID: 1230045591}
	_, err := b.Send(chel, "Сообщение от пидораса с ID "+fmt.Sprint(userID)+" из группы "+group+": "+text)
	if err != nil {
		logger.Error("ошибка при отправке сообщения админу: ", zap.Error(err))
	}

	return nil
}

func SendToAll(text string) error {
	data, err := getUserPUSH(0, true)
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
		go func() {
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
		}()
		if err != nil {
			logger.Error("ошибка при отправке PUSH-уведомления пользователю: ", zap.Error(err))
		}
	}

	return nil
}
