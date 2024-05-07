package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"raspygk/pkg/cache"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
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
	data, err := cache.Rdb.Get(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id)).Result()
	if err == nil && data != "" {
		var result []interface{}
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			logger.Error("Ошибка при преобразовании данных из Redis в функции GetUserData()", zap.Error(err))
			return 0, "", "", 0, err
		}
		return int(result[0].(float64)), result[1].(string), result[2].(string), int(result[3].(float64)), nil
	}

	// Данных нет в Redis, получаем их из базы данныхror": "parsing time \"\\xd0\\xa1\\xd1\\x82\\xd1\\x83\\xd0\\xb4\\xd0\\xb5\\xd0\\xbd\\xd1\\x82\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"\\xd0\\xa1\\xd1\\x82\\xd1\\x83\\xd0\\xb4\\xd0\\xb5\\xd0\\xbd\\xd1\\x82\" as \"20
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

	// Сохраняем данные в Redis
	dataToStore, err := json.Marshal([]interface{}{id, group.String, role.String, int(push.Int32)})
	if err != nil {
		logger.Error("Ошибка при преобразовании данных для сохранения в Redis в функции GetUserData()", zap.Error(err))
		return id, group.String, role.String, int(push.Int32), err
	}

	err = cache.Rdb.Set(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id), dataToStore, 1*time.Hour).Err()
	if err != nil {
		return id, group.String, role.String, int(push.Int32), err
	}

	logger.Info("Получение данных пользователя", zap.Int64("user_id", user_id))

	return id, group.String, role.String, int(push.Int32), nil
}

// Переписать нахуй, фулл хуйня
func GetUserPUSH(idshift int, debugMode bool) ([][]string, error) {
	return nil, nil
}

func GetUsersPUSH(idshift int, dataGroup []string) ([]string, error) {
	logger.Info("Получение данных о пользователях, у который включены PUSH-уведомления", zap.Int("idshift", idshift))
	rows, err := db.Conn.Queryx(`SELECT userid, "group" FROM users WHERE push = 1 AND (CASE WHEN $1 <> 0 THEN idshift = $1 ELSE TRUE END)`, idshift)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы users в функции getUserPUSH", zap.Error(err))
		return nil, err
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

	if dataGroup == nil {
		logger.Info("Получение групп по idshoft", zap.Int("idshift", idshift))
		rows, err = db.Conn.Queryx(`SELECT name FROM groups WHERE idshift = $1`, idshift)
		if err != nil {
			logger.Error("ошибка при выборке данных из таблицы users в функции getUserPUSH", zap.Error(err))
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			r := make(map[string]interface{})
			err := rows.MapScan(r)
			if err != nil {
				logger.Error("ошибка при обработке данных из таблицы users в функции getUserPUSH", zap.Error(err))
				return nil, err
			}

			// Проходим по всем ключам в map
			for _, value := range r {
				// Преобразуем значение в строку и добавляем его в слайс
				dataValue := fmt.Sprintf("%v", value)
				dataGroup = append(dataGroup, dataValue)
			}
		}
	}

	var usersSlice []string
	for _, value := range data {
		matchFound := false
		for _, group := range dataGroup {
			if group == value["group"] {
				matchFound = true
			}
		}
		if !matchFound {
			usersSlice = append(usersSlice, fmt.Sprint(value["userid"]))
		}
	}

	return usersSlice, nil
}

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

	var idshift int
	switch lesson {
	case 3, 4, 5:
		idshift = 2
	default:
		idshift = 1
	}

	rows, err = db.Conn.Queryx(`UPDATE users SET "group" = $1, idshift = $2 WHERE userid = $3`, group, idshift, user_id)
	if err != nil {
		logger.Error("ошибка при обновлении данных в таблице users в функции UpdateGroup", zap.Error(err))
		return err
	}
	defer rows.Close()

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
