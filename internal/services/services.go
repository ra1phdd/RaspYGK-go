package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
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

func GetUserData(userId int64) (int, int, bool, error) {
	data, err := cache.Rdb.Get(cache.Ctx, "GetUserData_"+fmt.Sprint(userId)).Result()
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
	rows, err := db.Conn.Queryx(`SELECT "group",role,push FROM users WHERE userid = $1`, userId)
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

	err = cache.Rdb.Set(cache.Ctx, "GetUserData_"+fmt.Sprint(userId), dataToStore, 1*time.Hour).Err()
	if err != nil {
		return int(group.Int32), int(role.Int32), push.Bool, err
	}

	logger.Info("Получение данных пользователя", zap.Int64("user_id", userId))

	return int(group.Int32), int(role.Int32), push.Bool, nil
}

func GetUserPUSH(idShift int, dataGroup []string, ignoreSettings bool) ([][]string, error) {
	logger.Info("Получение данных о пользователях, у которых включены PUSH-уведомления", zap.Int("idShift", idShift))

	var rows *sqlx.Rows
	var err error
	if ignoreSettings {
		rows, err = db.Conn.Queryx(`SELECT userid, "group" FROM users`)
	} else {
		rows, err = db.Conn.Queryx(`SELECT userid, "group" FROM users WHERE push = true`)
	}
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
		logger.Info("Получение групп по idShift", zap.Int("Тип знаменателя", idShift))
		if idShift == 0 {
			rows, err = db.Conn.Queryx(`SELECT id FROM groups`)
		} else {
			rows, err = db.Conn.Queryx(`SELECT id FROM groups WHERE idshift = $1`, idShift)
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
			if value["group"] != nil {
				groupValue := value["group"].(int64)
				if int64(group) == groupValue {
					matchFound = true
					break
				}
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

	fmt.Println(usersSlice)

	return usersSlice, nil
}

func AddUser(userId int64) error {
	logger.Info("Добавление пользователя", zap.Int64("user_id", userId))

	rows, err := db.Conn.Queryx(`INSERT INTO users (userid) VALUES ($1)`, userId)
	if err != nil {
		logger.Error("ошибка при добавлении пользователя в таблицу users", zap.Error(err))
		return err
	}
	defer rows.Close()

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(userId))

	return nil
}

func UpdateRole(role int, userId int64) error {
	logger.Info("Обновление роли пользователя", zap.Int("role", role), zap.Int64("user_id", userId))

	rows, err := db.Conn.Queryx(`UPDATE users SET role = $1 WHERE userid = $2`, role, userId)
	if err != nil {
		logger.Error("ошибка при обновлении роли в таблице users", zap.Error(err))
		return err
	}
	defer rows.Close()

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(userId))

	return nil
}

func UpdateGroup(group string, userId int64) error {
	logger.Info("Обновление группы пользователя", zap.String("group", group), zap.Int64("user_id", userId))

	var idGroup int
	err := db.Conn.QueryRowx(`SELECT id FROM groups WHERE name = $1`, group).Scan(&idGroup)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы groups в функции UpdateGroup", zap.Error(err))
		return err
	}

	_, err = db.Conn.Exec(`UPDATE users SET "group" = $1 WHERE userid = $2`, idGroup, userId)
	if err != nil {
		logger.Error("ошибка при обновлении данных в таблице users в функции UpdateGroup", zap.Error(err))
		return err
	}

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(userId))

	return nil
}

func UpdatePUSH(push bool, userId int64) error {
	logger.Info("Обновление PUSH-уведомлений пользователя", zap.Bool("push", push), zap.Int64("user_id", userId))

	rows, err := db.Conn.Queryx(`UPDATE users SET push = $1 WHERE userid = $2`, push, userId)
	if err != nil {
		logger.Error("ошибка при обновлении данных в таблице users в функции UpdatePUSH", zap.Error(err))
		return err
	}
	defer rows.Close()

	cache.Rdb.Del(cache.Ctx, "GetUserData_"+fmt.Sprint(userId))

	return nil
}

func GetGroupID(group string) (int64, error) {
	rows, err := db.Conn.Queryx(`SELECT id FROM groups WHERE name = $1`, group)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы groups в функции GetGroupID", zap.Error(err))
		return 0, err
	}
	defer rows.Close()

	var groupID int64
	for rows.Next() {
		err := rows.Scan(&groupID)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы groups в функции GetGroupID", zap.Error(err))
			return 0, err
		}
	}

	return groupID, nil
}

func EditByUserID(userId int64, text string) error {
	data, err := cache.Rdb.Get(cache.Ctx, "GetIDEditText_"+fmt.Sprint(userId)).Result()
	if err == nil && data != "" {
		cache.Rdb.Del(cache.Ctx, "GetIDEditText_"+fmt.Sprint(userId))
	}

	err = cache.Rdb.Set(cache.Ctx, "GetIDEditText_"+fmt.Sprint(userId), text, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	return nil
}

func EditByGroup(groupId int64, text string) error {
	data, err := cache.Rdb.Get(cache.Ctx, "GetEditGroupText_"+fmt.Sprint(groupId)).Result()
	if err == nil && data != "" {
		cache.Rdb.Del(cache.Ctx, "GetEditGroupText_"+fmt.Sprint(groupId))
	}

	err = cache.Rdb.Set(cache.Ctx, "GetEditGroupText_"+fmt.Sprint(groupId), text, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	return nil
}

func RemoveEditByUserID(userId int64) {
	cache.Rdb.Del(cache.Ctx, "GetIDEditText_"+fmt.Sprint(userId))
}
func RemoveEditByGroup(groupId int64) {
	cache.Rdb.Del(cache.Ctx, "GetEditGroupText_"+fmt.Sprint(groupId))
}
