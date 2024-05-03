package services

import (
	"database/sql"
	"fmt"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
	"strconv"
	"time"

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

func GetUserData(user_id int64) (int, string, string, int, error) {
	logger.Info("Получение данных пользователя", zap.Int64("user_id", user_id))
	rows, err := db.Conn.Queryx(`SELECT id,"group",role,push FROM users WHERE userid = $1`, user_id)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы users в функции getUserData", zap.Error(err))
		return 0, "", "", 0, err
	}
	defer rows.Close()

	var id int
	var group sql.NullString
	var role sql.NullString
	var push sql.NullInt32
	for rows.Next() {
		err := rows.Scan(&id, &group, &role, &push)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы users в функции getUserData", zap.Error(err))
			return 0, "", "", 0, err
		}
	}

	logger.Debug("Данные пользователя", zap.Int("id", id), zap.String("group", group.String), zap.String("role", role.String), zap.Int("push", int(push.Int32)))
	return id, group.String, role.String, int(push.Int32), nil
}

// Переписать нахуй, фулл хуйня
func GetUserPUSH(idshift int, debugMode bool) ([][]string, error) {
	return nil, nil
}

/*func GetUserPUSH(idshift int, debugMode bool) ([][]string, error) {
	var rows *sqlx.Rows
	var err error
	if idshift != 0 {
		logger.Info("Получение данных о пользователях, у который включены PUSH-уведомления", zap.Int("idshift", idshift))
		rows, err = db.Conn.Queryx(`SELECT userid,"group" FROM users WHERE push = 1 AND idshift = $1`, idshift)
		if err != nil {
			logger.Error("ошибка при выборке данных из таблицы users в функции getUserPUSH", zap.Error(err))
			return nil, err
		}
	} else {
		logger.Info("Получение данных о пользователях, у который включены PUSH-уведомления")
		rows, err = db.Conn.Queryx(`SELECT userid,"group" FROM users WHERE push = 1`)
		if err != nil {
			logger.Error("ошибка при выборке данных из таблицы users в функции getUserPUSH", zap.Error(err))
			return nil, err
		}
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы users в функции getUserPUSH", zap.Error(err))
			return nil, err
		}

		data = append(data, r)
	}

	var dataGroup []string
	for _, newGroup := range parser.NewData_first {
		matchFound := false
		for _, oldGroup := range parser.OldData_first {
			if newGroup[1] == oldGroup[1] {
				if fmt.Sprintf("%v", newGroup) == fmt.Sprintf("%v", oldGroup) {
					matchFound = true
				}
			}
		}
		if !matchFound || debugMode {
			fmt.Println("есть")
			dataGroup = append(dataGroup, newGroup[1])
		}
	}
	logger.Info("Группы", zap.Any("dataGroup", dataGroup))

	var lastData [][]string
	if dataGroup != nil {
		for _, value := range data {
			matchFound := false
			for _, group := range dataGroup {
				if group == value["group"] {
					matchFound = true
				}
			}
			if !matchFound || debugMode {
				lastData = append(lastData, []string{fmt.Sprint(value["userid"]), fmt.Sprint(value["group"])})
			}
		}
	}

	logger.Debug("Все пользователи, у которых включены PUSH-уведомления", zap.Any("data", lastData))
	return lastData, nil
}*/

func AddUser(user_id int64) error {
	logger.Info("Добавление пользователя", zap.Int64("user_id", user_id))
	rows, err := db.Conn.Queryx(`INSERT INTO users (userid) VALUES ($1)`, user_id)
	if err != nil {
		logger.Error("ошибка при добавлении пользователя в таблицу users", zap.Error(err))
		return err
	}
	defer rows.Close()

	return nil
}

func UpdateRole(role string, user_id int64) error {
	logger.Info("Обновление роли пользователя", zap.String("role", role), zap.Int64("user_id", user_id))
	rows, err := db.Conn.Queryx(`UPDATE users SET role = $1 WHERE userid = $2`, role, user_id)
	if err != nil {
		logger.Error("ошибка при обновлении роли в таблице users", zap.Error(err))
		return err
	}
	defer rows.Close()

	return nil
}

func UpdateGroup(group string, user_id int64) error {
	logger.Info("Обновление группы пользователя", zap.String("group", group), zap.Int64("user_id", user_id))

	rows, err := db.Conn.Queryx(`SELECT lesson FROM schedules WHERE "group" = $1 AND idweek = 1 AND typeweek = 'Числитель' LIMIT 1`, group)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы schedules в функции UpdateGroup", zap.Error(err))
		return err
	}
	defer rows.Close()

	var lesson int
	for rows.Next() {
		err := rows.Scan(&lesson)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы schedules в функции UpdateGroup", zap.Error(err))
			return err
		}
	}

	if lesson == 3 || lesson == 4 || lesson == 5 {
		logger.Debug("Вторая смена", zap.String("group", group))
		rows, err = db.Conn.Queryx(`UPDATE users SET "group" = $1, idshift = 2 WHERE userid = $2`, group, user_id)
		if err != nil {
			logger.Error("ошибка при обновлении данных в таблице users в функции UpdateGroup", zap.Error(err))
			return err
		}
		defer rows.Close()
	} else {
		logger.Debug("Первая смена", zap.String("group", group))
		rows, err = db.Conn.Queryx(`UPDATE users SET "group" = $1, idshift = 1 WHERE userid = $2`, group, user_id)
		if err != nil {
			logger.Error("ошибка при обновлении данных в таблице users в функции UpdateGroup", zap.Error(err))
			return err
		}
		defer rows.Close()
	}

	return nil
}

func UpdatePUSH(push int, user_id int64) error {
	logger.Info("Обновление PUSH-уведомлений пользователя", zap.Int("push", push), zap.Int64("user_id", user_id))
	rows, err := db.Conn.Queryx(`UPDATE users SET push = $1 WHERE userid = $2`, push, user_id)
	if err != nil {
		logger.Error("ошибка при обновлении данных в таблице users в функции UpdatePUSH", zap.Error(err))
		return err
	}
	defer rows.Close()

	return nil
}

func GetLessonTime(lesson int64, classroom string, idweek int64) (string, error) {
	logger.Info("Получение времени занятия", zap.Int64("lesson", lesson), zap.String("classroom", classroom), zap.Int64("idweek", idweek))

	if classroom == "" {
		return "-", nil
	}
	code := []rune(classroom)[0]

	var time string
	var ok bool

	if idweek >= 1 && idweek <= 5 {
		if code == 'А' || code == 'Б' || code == 'В' || code == 'М' || classroom == "ДОТ" {
			time, ok = WeekdayTimeMap[lesson]
			if !ok {
				logger.Error("Время пары не найдено")
				return "", nil
			}
		}
	} else if idweek == 6 {
		if code == 'А' || code == 'Б' || code == 'В' || code == 'М' || classroom == "ДОТ" {
			time, ok = WeekendTimeMap[lesson]
			if !ok {
				logger.Error("Время пары не найдено")
				return "", nil
			}
		}
	}

	logger.Debug("Время занятия", zap.String("time", time))
	return time, nil
}

func GetTextSchedule(type_id int64, group string, date string, typeweek string, idweek int64, schedules []map[string]interface{}, replaces []map[string]interface{}) (string, error) {
	logger.Info("Получение текста расписания", zap.Int64("type_id", type_id), zap.String("group", group), zap.String("date", date), zap.String("typeweek", typeweek), zap.Int64("idweek", idweek))

	var week string
	switch idweek {
	case 1:
		week = "понедельник"
	case 2:
		week = "вторник"
	case 3:
		week = "среду"
	case 4:
		week = "четверг"
	case 5:
		week = "пятницу"
	case 6:
		week = "субботу"
	default:
		logger.Error("Неизвестная неделя")
		return "", nil
	}

	text := "Расписание " + group + " на " + week + ", " + date + " (" + typeweek + "):\n"

	for _, schedule := range schedules {
		lesson := schedule["lesson"].(int64)
		discipline := fmt.Sprint(schedule["discipline"])
		teacher := fmt.Sprint(schedule["teacher"])
		classroom_sched := fmt.Sprint(schedule["classroom"])

		if replaces != nil {
			replaceDone := 0
			for _, replace := range replaces {
				fmt.Println(replace)
				lesson_replace := replace["lesson"].(int64)
				discipline_replace := fmt.Sprint(replace["discipline_replace"])
				fmt.Println(discipline_replace)
				if discipline_replace == "по расписанию " || discipline_replace == "..." {
					discipline_replace = discipline
				}
				classroom_replace := fmt.Sprint(replace["classroom"])

				if lesson == lesson_replace {
					time, err := GetLessonTime(lesson_replace, classroom_replace, idweek)
					if err != nil {
						logger.Error("ошибка при вызове функции GetLessonTime, когда замена есть", zap.Error(err))
						return "", nil
					}

					text += strconv.FormatInt(lesson_replace, 10) + " пара [" + classroom_replace + "] (" + time + ") " + discipline_replace + " (❗замена)\n"
					replaceDone = 1
				}
			}
			if replaceDone == 0 {
				if discipline != "<nil>" {
					time, err := GetLessonTime(lesson, classroom_sched, idweek)
					if err != nil {
						logger.Error("ошибка при вызове функции GetLessonTime, когда замены нет", zap.Error(err))
						return "", nil
					}

					text += strconv.FormatInt(lesson, 10) + " пара [" + classroom_sched + "] (" + time + ") " + discipline + " (" + teacher + ")\n"
				}
			}
		} else {
			if discipline != "<nil>" {
				time, err := GetLessonTime(lesson, classroom_sched, idweek)

				if err != nil {
					logger.Error("ошибка при вызове GetLessonTime, когда замены не обновлены", zap.Error(err))
					return "", nil
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

func GetSchedule(group string) (string, error) {
	logger.Info("Получение расписания", zap.String("group", group))
	// Получение даты, номера и типа недели
	rows, err := db.Conn.Queryx(`SELECT idweek,typeweek,date FROM arrays ORDER BY id DESC LIMIT 1`)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы arrays в функции getSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var idweek int64
	var typeweek string
	var date string
	for rows.Next() {
		err := rows.Scan(&idweek, &typeweek, &date)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы arrays в функции getSchedule", zap.Error(err))
			return "", err
		}
	}
	dateForm, _ := time.Parse("2006-01-02T15:04:05Z", date)

	// Получение замен
	rows, err = db.Conn.Queryx(`SELECT * FROM replaces WHERE "group" = $1 AND date = $2`, group, date)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы replaces в функции getSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var replaces []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы replaces в функции getSchedule", zap.Error(err))
			return "", err
		}

		replaces = append(replaces, r)
	}

	// Получение расписания
	rows, err = db.Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND typeweek = $3`, group, idweek, typeweek)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы schedules в функции getSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы schedules в функции getSchedule", zap.Error(err))
			return "", err
		}

		schedules = append(schedules, r)
	}

	text, err := GetTextSchedule(0, group, dateForm.Format("02.01.2006"), typeweek, idweek, schedules, replaces)
	if err != nil {
		logger.Error("ошибка при вызове функции GetTextSchedule", zap.Error(err))
		return "", err
	}

	return text, nil
}

func GetNextSchedule(group string) (string, error) {
	logger.Info("Получение расписания на следующий день", zap.String("group", group))
	// Получение даты, номера и типа недели
	rows, err := db.Conn.Queryx(`SELECT idweek,typeweek,date FROM arrays ORDER BY id DESC LIMIT 1`)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы arrays в функции getNextSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var idweek int64
	var typeweek string
	var date string

	for rows.Next() {
		err := rows.Scan(&idweek, &typeweek, &date)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы arrays в функции getNextSchedule", zap.Error(err))
			return "", err
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
	rows, err = db.Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND typeweek = $3`, group, idweek, typeweek)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы arrays в функции getNextSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы schedules в функции getNextSchedule", zap.Error(err))
			return "", err
		}

		schedules = append(schedules, r)
	}

	text, err := GetTextSchedule(1, group, nextDate.Format("02.01.2006"), typeweek, idweek, schedules, nil)
	if err != nil {
		logger.Error("ошибка при вызове функции GetTextSchedule", zap.Error(err))
		return "", err
	}

	return text, nil
}

func GetPrevSchedule(group string) (string, error) {
	logger.Info("Получение расписания на предыдущий день", zap.String("group", group))
	// Получение даты, номера и типа недели
	rows, err := db.Conn.Queryx(`SELECT idweek,typeweek,date FROM arrays ORDER BY id DESC LIMIT 1`)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы arrays в функции getPrevSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var idweek int64
	var typeweek string
	var date string
	for rows.Next() {
		err := rows.Scan(&idweek, &typeweek, &date)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы arrays в функции getPrevSchedule", zap.Error(err))
			return "", err
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
	rows, err = db.Conn.Queryx(`SELECT * FROM replaces WHERE "group" = $1 AND date = $2`, group, prevDate.Format("2006-01-02T15:04:05Z"))
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы replaces в функции getPrevSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var replaces []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы replaces в функции getPrevSchedule", zap.Error(err))
			return "", err
		}

		replaces = append(replaces, r)
	}

	// Получение расписания
	rows, err = db.Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND typeweek = $3`, group, idweek, typeweek)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы schedules в функции getPrevSchedule", zap.Error(err))
		return "", err
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы schedules в функции getPrevSchedule", zap.Error(err))
			return "", err
		}

		schedules = append(schedules, r)
	}

	text, err := GetTextSchedule(2, group, prevDate.Format("02.01.2006"), typeweek, idweek, schedules, replaces)
	if err != nil {
		logger.Error("ошибка при вызове функции GetTextSchedule", zap.Error(err))
		return "", err
	}

	return text, nil
}
