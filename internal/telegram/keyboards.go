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
		if userState.Push == 1 {
			buttonPush = "✅"
		} else {
			buttonPush = "❌"
		}

		// ААААААААААААААААААА
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
		btns := createGroupButtons(menu, models.GroupOITButtons)
		return menu, btns

	default:
		return nil, nil
	}
}

func createGroupButtons(menu *tele.ReplyMarkup, groups []string) map[string]tele.Btn {
	btns := make(map[string]tele.Btn)
	rows := make([][]tele.Btn, 0)
	row := make([]tele.Btn, 0)

	for i, group := range groups {
		btn := menu.Data(group, group)
		btns[group] = btn
		row = append(row, btn)

		if (i+1)%4 == 0 || i == len(groups)-1 {
			rows = append(rows, row)
			row = make([]tele.Btn, 0)
		}
	}

	for _, row := range rows {
		menu.Inline(menu.Row(row...))
	}

	return btns
}
