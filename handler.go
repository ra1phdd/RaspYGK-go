package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var WeekdayTimeMap = map[int64]string{
	0: "08:00",
	1: "09:20",
	2: "11:00",
	3: "13:20",
	4: "15:05",
	5: "17:05",
	6: "18:45",
}

var WeekendTimeMap = map[int64]string{
	0: "08:00",
	1: "09:20",
	2: "11:00",
	3: "13:20",
	4: "15:05",
	5: "16:45",
}

func getUserData(user_id int64) (int, string, string, int, error) {
	logger.Info("Получение данных пользователя", zap.Int64("user_id", user_id))
	rows, err := Conn.Queryx(`SELECT id,"group",role,push FROM users WHERE userid = $1`, user_id)
	if err != nil {
		return 0, "", "", 0, handleError("ошибка при выборке данных из таблицы users в функции getUserData: %w", err)
	}
	defer rows.Close()

	var id int
	var group sql.NullString
	var role sql.NullString
	var push sql.NullInt32
	for rows.Next() {
		err := rows.Scan(&id, &group, &role, &push)
		if err != nil {
			return 0, "", "", 0, handleError("ошибка при обработке данных из таблицы users в функции getUserData: %w", err)
		}
	}

	logger.Debug("Данные пользователя", zap.Int("id", id), zap.String("group", group.String), zap.String("role", role.String), zap.Int("push", int(push.Int32)))
	return id, group.String, role.String, int(push.Int32), nil
}

func getUserPUSH(idshift int) ([][]string, error) {
	var rows *sqlx.Rows
	var err error
	if idshift != 0 {
		logger.Info("Получение данных о пользователях, у который включены PUSH-уведомления", zap.Int("idshift", idshift))
		rows, err = Conn.Queryx(`SELECT userid,"group" FROM users WHERE push = 1 AND idshift = $1`, idshift)
		if err != nil {
			return nil, handleError("ошибка при выборке данных из таблицы users в функции getUserPUSH: %w", err)
		}
	} else {
		logger.Info("Получение данных о пользователях, у который включены PUSH-уведомления")
		rows, err = Conn.Queryx(`SELECT userid,"group" FROM users WHERE push = 1`)
		if err != nil {
			return nil, handleError("ошибка при выборке данных из таблицы users в функции getUserPUSH: %w", err)
		}
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			return nil, handleError("ошибка при обработке данных из таблицы users в функции getUserPUSH: %w", err)
		}

		data = append(data, r)
	}

	logger.Info("newData_first", zap.Any("data", newData_first))
	logger.Info("oldData_first", zap.Any("data", oldData_first))

	var dataGroup []string
	for _, newGroup := range newData_first {
		matchFound := false
		for _, oldGroup := range oldData_first {
			if newGroup[1] == oldGroup[1] {
				if fmt.Sprintf("%v", newGroup) == fmt.Sprintf("%v", oldGroup) {
					matchFound = true
				}
			}
		}
		fmt.Println(matchFound)
		if !matchFound {
			dataGroup = append(dataGroup, newGroup[1])
		}
		logger.Info("Группы", zap.Any("dataGroup", dataGroup))
	}

	var lastData [][]string
	if dataGroup != nil {
		for _, value := range data {
			matchFound := false
			for _, group := range dataGroup {
				if group == value["group"] {
					matchFound = true
					break
				}
			}
			if !matchFound {
				lastData = append(lastData, []string{fmt.Sprint(value["userid"]), fmt.Sprint(value["group"])})
			}
		}
	}

	logger.Debug("Все пользователи, у которых включены PUSH-уведомления", zap.Any("data", lastData))
	return lastData, nil
}

func AddUser(user_id int64) error {
	logger.Info("Добавление пользователя", zap.Int64("user_id", user_id))
	rows, err := Conn.Queryx(`INSERT INTO users (userid) VALUES ($1)`, user_id)
	if err != nil {
		return handleError("ошибка при добавлении пользователя в таблицу users: %w", err)
	}
	defer rows.Close()

	return nil
}

func UpdateRole(role string, user_id int64) error {
	logger.Info("Обновление роли пользователя", zap.String("role", role), zap.Int64("user_id", user_id))
	rows, err := Conn.Queryx(`UPDATE users SET role = $1 WHERE userid = $2`, role, user_id)
	if err != nil {
		return handleError("ошибка при обновлении роли в таблице users: %w", err)
	}
	defer rows.Close()

	return nil
}

func UpdateGroup(group string, user_id int64) error {
	logger.Info("Обновление группы пользователя", zap.String("group", group), zap.Int64("user_id", user_id))

	rows, err := Conn.Queryx(`SELECT lesson FROM schedules WHERE "group" = $1 AND idweek = 1 AND typeweek = 'Числитель' LIMIT 1`, group)
	if err != nil {
		return handleError("ошибка при выборке данных из таблицы schedules в функции UpdateGroup: %w", err)
	}
	defer rows.Close()

	var lesson int
	for rows.Next() {
		err := rows.Scan(&lesson)
		if err != nil {
			return handleError("ошибка при обработке данных из таблицы schedules в функции UpdateGroup: %w", err)
		}
	}

	if lesson == 3 || lesson == 4 || lesson == 5 {
		logger.Debug("Вторая смена", zap.String("group", group))
		rows, err = Conn.Queryx(`UPDATE users SET "group" = $1, idshift = 2 WHERE userid = $2`, group, user_id)
		if err != nil {
			return handleError("ошибка при обновлении данных в таблице users в функции UpdateGroup: %w", err)
		}
		defer rows.Close()
	} else {
		logger.Debug("Первая смена", zap.String("group", group))
		rows, err = Conn.Queryx(`UPDATE users SET "group" = $1, idshift = 1 WHERE userid = $2`, group, user_id)
		if err != nil {
			return handleError("ошибка при обновлении данных в таблице users в функции UpdateGroup: %w", err)
		}
		defer rows.Close()
	}

	return nil
}

func UpdatePUSH(push int, user_id int64) error {
	logger.Info("Обновление PUSH-уведомлений пользователя", zap.Int("push", push), zap.Int64("user_id", user_id))
	rows, err := Conn.Queryx(`UPDATE users SET push = $1 WHERE userid = $2`, push, user_id)
	if err != nil {
		return handleError("ошибка при обновлении данных в таблице users в функции UpdatePUSH: %w", err)
	}
	defer rows.Close()

	return nil
}

func GetLessonTime(lesson int64, classroom string, idweek int64) (string, error) {
	logger.Info("Получение времени занятия", zap.Int64("lesson", lesson), zap.String("classroom", classroom), zap.Int64("idweek", idweek))

	code := []rune(classroom)[0]

	var time string
	var ok bool

	if idweek >= 1 && idweek <= 5 {
		if code == 'А' || code == 'Б' || code == 'В' || code == 'М' || classroom == "ДОТ" {
			time, ok = WeekdayTimeMap[lesson]
			if !ok {
				return "", handleError("Время пары не найдено", nil)
			}
		}
	} else if idweek == 6 {
		if code == 'А' || code == 'Б' || code == 'В' || code == 'М' || classroom == "ДОТ" {
			time, ok = WeekendTimeMap[lesson]
			if !ok {
				return "", handleError("Время пары не найдено", nil)
			}
		}
	}

	logger.Debug("Время занятия", zap.String("time", time))
	return time, nil
}

func GetTextSchedule(type_id int64, group string, date string, typeweek string, idweek int64, schedules []map[string]interface{}, replaces []map[string]interface{}) (string, error) {
	logger.Info("Получение текста расписания", zap.Int64("type_id", type_id), zap.String("group", group), zap.String("date", date), zap.String("typeweek", typeweek), zap.Int64("idweek", idweek))

	text := "Расписание " + group + " на " + date + " (" + typeweek + "):\n"

	for _, schedule := range schedules {
		lesson := schedule["lesson"].(int64)
		discipline := fmt.Sprint(schedule["discipline"])
		teacher := fmt.Sprint(schedule["teacher"])
		classroom_sched := fmt.Sprint(schedule["classroom"])

		if replaces != nil {
			for _, replace := range replaces {
				lesson_replace := replace["lesson"].(int64)
				discipline_replace := fmt.Sprint(replace["discipline_replace"])
				classroom_replace := fmt.Sprint(replace["classroom"])

				if lesson == lesson_replace {
					time, err := GetLessonTime(lesson_replace, classroom_replace, idweek)
					if err != nil {
						return "", handleError("ошибка при вызове функции GetLessonTime, когда замена есть: %w", err)
					}

					text += strconv.FormatInt(lesson_replace, 10) + " пара [" + classroom_replace + "] (" + time + ") " + discipline_replace + " (❗замена)\n"
					break
				} else {
					if discipline != "<nil>" {
						time, err := GetLessonTime(lesson, classroom_sched, idweek)
						if err != nil {
							return "", handleError("ошибка при вызове функции GetLessonTime, когда замены нет: %w", err)
						}

						text += strconv.FormatInt(lesson, 10) + " пара [" + classroom_sched + "] (" + time + ") " + discipline + " (" + teacher + ")\n"
					}
					break
				}
			}
		} else {
			if discipline != "<nil>" {
				time, err := GetLessonTime(lesson, classroom_sched, idweek)

				if err != nil {
					return "", handleError("ошибка при вызове GetLessonTime, когда замены не обновлены: %w", err)
				}

				text += strconv.FormatInt(lesson, 10) + " пара [" + classroom_sched + "] (" + time + ") " + discipline + " (" + teacher + ")\n"
			}
		}
	}

	if type_id == 0 || type_id == 2 {
		text += "\n✅ Замены обновлены"
	} else if type_id == 1 {
		text += "\n❌ Замены не обновлены"
	}

	logger.Debug("Текст расписания", zap.String("text", text))
	return text, nil
}

func getSchedule(group string) (string, error) {
	logger.Info("Получение расписания", zap.String("group", group))
	// Получение даты, номера и типа недели
	rows, err := Conn.Queryx(`SELECT idweek,typeweek,date FROM arrays ORDER BY id DESC LIMIT 1`)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы arrays в функции getSchedule: %w", err)
	}
	defer rows.Close()

	var idweek int64
	var typeweek string
	var date string
	for rows.Next() {
		err := rows.Scan(&idweek, &typeweek, &date)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы arrays в функции getSchedule: %w", err)
		}
	}
	dateForm, _ := time.Parse("2006-01-02T15:04:05Z", date)

	// Получение замен
	rows, err = Conn.Queryx(`SELECT * FROM replaces WHERE "group" = $1 AND date = $2`, group, date)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы replaces в функции getSchedule: %w", err)
	}
	defer rows.Close()

	var replaces []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы replaces в функции getSchedule: %w", err)
		}

		replaces = append(replaces, r)
	}

	// Получение расписания
	rows, err = Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND typeweek = $3`, group, idweek, typeweek)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы schedules в функции getSchedule: %w", err)
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы schedules в функции getSchedule: %w", err)
		}

		schedules = append(schedules, r)
	}

	text, err := GetTextSchedule(0, group, dateForm.Format("02.01.2006"), typeweek, idweek, schedules, replaces)
	if err != nil {
		return "", handleError("ошибка при вызове функции GetTextSchedule: %w", err)
	}

	return text, nil
}

func getNextSchedule(group string) (string, error) {
	logger.Info("Получение расписания на следующий день", zap.String("group", group))
	// Получение даты, номера и типа недели
	rows, err := Conn.Queryx(`SELECT idweek,typeweek,date FROM arrays ORDER BY id DESC LIMIT 1`)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы arrays в функции getNextSchedule: %w", err)
	}
	defer rows.Close()

	var idweek int64
	var typeweek string
	var date string

	for rows.Next() {
		err := rows.Scan(&idweek, &typeweek, &date)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы arrays в функции getNextSchedule: %w", err)
		}
	}

	dateForm, _ := time.Parse("2006-01-02T15:04:05Z", date)
	var nextDate time.Time

	idweek++
	if idweek == 7 {
		idweek = 1
		if typeweek == "Числитель" {
			typeweek = "Знаменатель"
		}
		nextDate = dateForm.AddDate(0, 0, 2)
	} else {
		nextDate = dateForm.AddDate(0, 0, 1)
	}

	// Получение расписания
	rows, err = Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND typeweek = $3`, group, idweek, typeweek)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы schedules в функции getNextSchedule: %w", err)
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы schedules в функции getNextSchedule: %w", err)
		}

		schedules = append(schedules, r)
	}

	text, err := GetTextSchedule(1, group, nextDate.Format("02.01.2006"), typeweek, idweek, schedules, nil)
	if err != nil {
		return "", handleError("ошибка при вызове функции GetTextSchedule: %w", err)
	}

	return text, nil
}

func getPrevSchedule(group string) (string, error) {
	logger.Info("Получение расписания на предыдущий день", zap.String("group", group))
	// Получение даты, номера и типа недели
	rows, err := Conn.Queryx(`SELECT idweek,typeweek,date FROM arrays ORDER BY id DESC LIMIT 1`)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы arrays в функции getPrevSchedule: %w", err)
	}
	defer rows.Close()

	var idweek int64
	var typeweek string
	var date string
	for rows.Next() {
		err := rows.Scan(&idweek, &typeweek, &date)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы arrays в функции getPrevSchedule: %w", err)
		}
	}
	dateForm, _ := time.Parse("2006-01-02T15:04:05Z", date)
	var prevDate time.Time

	idweek--
	if idweek == 0 {
		idweek = 6
		if typeweek == "Числитель" {
			typeweek = "Знаменатель"
		}
		prevDate = dateForm.AddDate(0, 0, -2)
	} else {
		prevDate = dateForm.AddDate(0, 0, -1)
	}

	// Получение замен
	rows, err = Conn.Queryx(`SELECT * FROM replaces WHERE "group" = $1 AND date = $2`, group, prevDate.Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы replaces в функции getPrevSchedule: %w", err)
	}
	defer rows.Close()

	var replaces []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы replaces в функции getPrevSchedule: %w", err)
		}

		replaces = append(replaces, r)
	}

	// Получение расписания
	rows, err = Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND typeweek = $3`, group, idweek, typeweek)
	if err != nil {
		return "", handleError("ошибка при выборке данных из таблицы schedules в функции getPrevSchedule: %w", err)
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			return "", handleError("ошибка при обработке данных из таблицы schedules в функции getPrevSchedule: %w", err)
		}

		schedules = append(schedules, r)
	}

	text, err := GetTextSchedule(2, group, prevDate.Format("02.01.2006"), typeweek, idweek, schedules, replaces)
	if err != nil {
		return "", handleError("ошибка при вызове функции GetTextSchedule: %w", err)
	}

	return text, nil
}
