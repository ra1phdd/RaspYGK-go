package models

import tele "gopkg.in/telebot.v3"

type UserState struct {
	ChatID int64
	UserID int64
	Name   string
	Group  int
	Role   int
	Push   bool
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
