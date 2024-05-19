package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"raspygk/pkg/cache"
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

func GetUserData(user_id int64) (int, int, bool, error) {
	data, err := cache.Rdb.Get(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id)).Result()
	if err == nil && data != "" {
		var result []interface{}
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			logger.Error("Ошибка при преобразовании данных из Redis в функции GetUserData()", zap.Error(err))
			return 0, 0, false, err
		}

		var push bool
		if v, ok := result[2].(bool); ok {
			push = v
		}

		return int(result[0].(float64)), int(result[1].(float64)), push, nil
	}

	// Данных нет в Redis, получаем их из базы данных
	rows, err := db.Conn.Queryx(`SELECT "group",role,push FROM users WHERE userid = $1`, user_id)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы users в функции getUserData", zap.Error(err))
		return 0, 0, false, err
	}
	defer rows.Close()

	var group sql.NullInt32
	var role sql.NullInt32
	var push sql.NullBool
	for rows.Next() {
		err := rows.Scan(&group, &role, &push)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы users в функции getUserData", zap.Error(err))
			return 0, 0, false, err
		}
	}

	// Сохраняем данные в Redis
	dataToStore, err := json.Marshal([]interface{}{int(group.Int32), int(role.Int32), push.Bool})
	if err != nil {
		logger.Error("Ошибка при преобразовании данных для сохранения в Redis в функции GetUserData()", zap.Error(err))
		return int(group.Int32), int(role.Int32), push.Bool, err
	}

	err = cache.Rdb.Set(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id), dataToStore, 1*time.Hour).Err()
	if err != nil {
		return int(group.Int32), int(role.Int32), push.Bool, err
	}

	logger.Info("Получение данных пользователя", zap.Int64("user_id", user_id))

	return int(group.Int32), int(role.Int32), push.Bool, nil
}

func GetUserPUSH(idshift int, ignoreSettings bool, dataGroup []string) ([][]string, error) {
	logger.Info("Получение данных о пользователях, у которых включены PUSH-уведомления", zap.Int("idshift", idshift))

	var query string
	if ignoreSettings {
		query = `SELECT userid, "group" FROM users`
	} else {
		query = `SELECT userid, "group" FROM users WHERE push = true`
	}
	rows, err := db.Conn.Queryx(query)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы users в функции GetUserPUSH", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы users в функции GetUserPUSH", zap.Error(err))
			return nil, err
		}

		data = append(data, r)
	}

	if dataGroup == nil {
		logger.Info("Получение групп по idshift", zap.Int("Тип знаменателя", idshift))
		if idshift == 0 {
			rows, err = db.Conn.Queryx(`SELECT id FROM groups`)
		} else {
			rows, err = db.Conn.Queryx(`SELECT id FROM groups WHERE idshift = $1`, idshift)
		}
		if err != nil {
			logger.Error("ошибка при выборке данных из таблицы groups в функции GetUserPUSH", zap.Error(err))
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			r := make(map[string]interface{})
			err := rows.MapScan(r)
			if err != nil {
				logger.Error("ошибка при обработке данных из таблицы groups в функции GetUserPUSH", zap.Error(err))
				return nil, err
			}

			for _, value := range r {
				dataValue := fmt.Sprintf("%v", value)
				dataGroup = append(dataGroup, dataValue)
			}
		}
	}

	var usersSlice [][]string
	for _, value := range data {
		matchFound := false
		for _, group := range dataGroup {
			group, err := strconv.Atoi(group)
			if err != nil {
				logger.Error("Ошибка преобразования lesson в функции InsertData()")
			}
			groupValue := value["group"].(int64)
			if int64(group) == groupValue {
				matchFound = true
				break
			}
		}
		if matchFound {
			userInfo := []string{
				fmt.Sprint(value["userid"]),
				fmt.Sprint(value["group"]),
			}
			usersSlice = append(usersSlice, userInfo)
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

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id))

	return nil
}

func UpdateRole(role int, user_id int64) error {
	logger.Info("Обновление роли пользователя", zap.Int("role", role), zap.Int64("user_id", user_id))

	rows, err := db.Conn.Queryx(`UPDATE users SET role = $1 WHERE userid = $2`, role, user_id)
	if err != nil {
		logger.Error("ошибка при обновлении роли в таблице users", zap.Error(err))
		return err
	}
	defer rows.Close()

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id))

	return nil
}

func UpdateGroup(group string, user_id int64) error {
	logger.Info("Обновление группы пользователя", zap.String("group", group), zap.Int64("user_id", user_id))

	var idGroup int
	err := db.Conn.QueryRowx(`SELECT id FROM groups WHERE name = $1`, group).Scan(&idGroup)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы groups в функции UpdateGroup", zap.Error(err))
		return err
	}

	_, err = db.Conn.Exec(`UPDATE users SET "group" = $1 WHERE userid = $2`, idGroup, user_id)
	if err != nil {
		logger.Error("ошибка при обновлении данных в таблице users в функции UpdateGroup", zap.Error(err))
		return err
	}

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id))

	return nil
}

func UpdatePUSH(push bool, user_id int64) error {
	logger.Info("Обновление PUSH-уведомлений пользователя", zap.Bool("push", push), zap.Int64("user_id", user_id))

	rows, err := db.Conn.Queryx(`UPDATE users SET push = $1 WHERE userid = $2`, push, user_id)
	if err != nil {
		logger.Error("ошибка при обновлении данных в таблице users в функции UpdatePUSH", zap.Error(err))
		return err
	}
	defer rows.Close()

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(user_id))

	return nil
}
