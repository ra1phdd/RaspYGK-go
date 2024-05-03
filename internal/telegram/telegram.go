package telegram

import (
	"raspygk/pkg/logger"
	"time"

	"go.uber.org/zap"

	tele "gopkg.in/telebot.v3"
)

type UserState struct {
	ID     int
	chatID int64
	userID int64
	name   string
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

type ButtonOption struct {
	Value   string
	Display string
}

var (
	b           *tele.Bot
	userState   UserState
	buttons     Buttons
	mainButtons = []ButtonOption{
		{Value: "Schedule", Display: "Расписание"},
		{Value: "PrevSchedule", Display: "Предыдущее"},
		{Value: "NextSchedule", Display: "Следующее"},
		{Value: "Settings", Display: "Настройки"},
		{Value: "Support", Display: "Техподдержка"},
	}
	roleButtons = []ButtonOption{
		{Value: "Student", Display: "Студент"},
		{Value: "Teacher", Display: "Преподаватель"},
	}
	typeButtons = []ButtonOption{
		{Value: "OIT", Display: "ОИТ"},
		{Value: "OAR", Display: "ОАР"},
		{Value: "SO", Display: "СО"},
		{Value: "MMO", Display: "ММО"},
		{Value: "OEP", Display: "ОЭП"},
	}
	groupOITButtons = []string{"IS1_11", "IS1_13", "IS1_15", "IS1_21",
		"IS1_23", "IS1_25", "IS1_31", "IS1_33",
		"IS1_35", "IS1_41", "IS1_43", "IS1_45",
		"SA1_11", "SA1_21", "SA1_31", "SA1_41",
		"IB1_11", "IB1_21", "IB1_41", "IB1_21"}
	groupOARButtons = []string{}
	groupSOButtons  = []string{}
	groupMMOButtons = []string{}
	groupOEPButtons = []string{}
)

func Init(TelegramAPI string) {
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
