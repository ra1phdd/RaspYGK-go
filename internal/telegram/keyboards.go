package telegram

import (
	"raspygk/internal/models"

	tele "gopkg.in/telebot.v3"
)

func Keyboards(keyboard int) (*tele.ReplyMarkup, map[string]tele.Btn) {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}

	switch keyboard {
	case 0:
		btns := make(map[string]tele.Btn, len(models.MainButtons))
		for _, item := range models.MainButtons {
			btns[item.Value] = menu.Text(item.Display)
		}

		menu.Reply(
			menu.Row(btns["Schedule"]),
			menu.Row(btns["PrevSchedule"], btns["NextSchedule"]),
			menu.Row(btns["Settings"], btns["Support"]),
		)

		return menu, nil

	case 1:
		var buttonPush string
		if userState.Push {
			buttonPush = "✅"
		} else {
			buttonPush = "❌"
		}

		btns := make(map[string]tele.Btn)
		btns["changeRole"] = menu.Data("Смена роли", "role")
		btns["changeGroup"] = menu.Data("Смена группы", "group")
		btns["changePUSH"] = menu.Data("Получение PUSH-уведомлений "+buttonPush, "push")

		menu.Inline(
			menu.Row(btns["changeRole"], btns["changeGroup"]),
			menu.Row(btns["changePUSH"]),
		)

		return menu, btns

	case 2:
		btns := make(map[string]tele.Btn, len(models.RoleButtons))
		for _, item := range models.RoleButtons {
			btns[item.Value] = menu.Data(item.Display, item.Value)
		}

		menu.Inline(
			menu.Row(btns["Student"], btns["Teacher"]),
		)

		return menu, btns

	case 3:
		btns := make(map[string]tele.Btn, len(models.TypeButtons))
		for _, item := range models.TypeButtons {
			btns[item.Value] = menu.Data(item.Display, item.Value)
		}

		menu.Inline(
			menu.Row(btns["OIT"], btns["OAR"]),
			menu.Row(btns["SO"], btns["MMO"]),
			menu.Row(btns["OEP"]),
		)

		return menu, btns

	case 4:
		btns := make(map[string]tele.Btn, len(models.GroupOITButtons))
		for _, item := range models.GroupOITButtons {
			btns[item.Value] = menu.Data(item.Display, item.Value)
		}

		menu.Inline(
			menu.Row(btns["IS1_11"], btns["IS1_13"], btns["IS1_15"]),
			menu.Row(btns["IS1_21"], btns["IS1_23"], btns["IS1_25"]),
			menu.Row(btns["IS1_31"], btns["IS1_33"], btns["IS1_35"]),
			menu.Row(btns["IS1_41"], btns["IS1_43"], btns["IS1_45"]),
		)

		return menu, btns

	default:
		return nil, nil
	}
}
