package services

import (
	"encoding/json"
	"fmt"
	"raspygk/pkg/cache"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func GetLessonTime(lesson int64, classroom string, idweek int) (string, error) {
	logger.Info("Получение времени занятия", zap.Int64("lesson", lesson), zap.String("classroom", classroom), zap.Int("idweek", idweek))

	if classroom == "" {
		return "", nil
	}
	code := []rune(classroom)[0]

	var lessonTime string
	var ok bool

	isValidCode := code == 'А' || code == 'Б' || code == 'В' || code == 'М' || classroom == "ДОТ"

	if isValidCode {
		switch idweek {
		case 1, 2, 3, 4, 5:
			lessonTime, ok = WeekdayTimeMap[lesson]
			if !ok {
				logger.Error("Время пары не найдено")
				return "", nil
			}
		case 6:
			lessonTime, ok = WeekendTimeMap[lesson]
			if !ok {
				logger.Error("Время пары не найдено")
				return "", nil
			}
		}
	}

	logger.Debug("Время занятия", zap.String("time", lessonTime))
	return lessonTime, nil
}

func GetTextSchedule(typeId int64, idGroup int, date string, typeweek int, idweek int, schedules []map[string]interface{}, replaces []map[string]interface{}) (string, error) {
	logger.Info("Получение текста расписания", zap.Int64("type_id", typeId), zap.Int("group", idGroup), zap.String("date", date), zap.Int("typeweek", typeweek), zap.Int("idweek", idweek))
	daysOfWeek := []string{"", "понедельник", "вторник", "среду", "четверг", "пятницу", "субботу"}
	if idweek < 1 || idweek > 6 {
		logger.Error("Неизвестная неделя")
		return "", nil
	}
	week := daysOfWeek[idweek]

	var group string
	err := db.Conn.QueryRowx(`SELECT name FROM groups WHERE id = $1`, idGroup).Scan(&group)
	if err != nil {
		logger.Error("ошибка при выборке данных из таблицы groups в функции UpdateGroup", zap.Error(err))
		return "", err
	}

	var textWeek string
	switch typeweek {
	case 1:
		textWeek = "Числитель"
	case 2:
		textWeek = "Знаменатель"
	}

	text := "Расписание " + group + " на " + week + ", " + date + " (" + textWeek + "):\n"

	mergedMap := mergeSchedules(schedules, replaces)
	for _, item := range mergedMap {
		fmt.Println(item)

		lesson := item["lesson"].(int64)
		discipline := fmt.Sprint(item["discipline"])
		disciplineReplace := fmt.Sprint(item["discipline_replace"])
		teacher := fmt.Sprint(item["teacher"])
		classroom := fmt.Sprint(item["classroom"])

		lessonTime, errTime := GetLessonTime(lesson, classroom, idweek)
		if errTime != nil {
			logger.Error("ошибка при вызове функции GetLessonTime, когда замена есть", zap.Error(errTime))
			return "", nil
		}

		if fmt.Sprint(item["source"]) == "schedules" {
			text += strconv.FormatInt(lesson, 10) + " пара [" + classroom + "] (" + lessonTime + ") " + discipline + " (" + teacher + ")\n"
		} else if item["lesson"].(int64) == -1 {
			text += disciplineReplace + " (❗замена)\n"
			break
		} else {
			if strings.ToLower(strings.TrimSpace(disciplineReplace)) != "снято" {
				text += strconv.FormatInt(lesson, 10) + " пара [" + classroom + "] (" + lessonTime + ") " + disciplineReplace + " (❗замена)\n"
			} else {
				text += strconv.FormatInt(lesson, 10) + " пара - снято (❗замена)\n"
			}
		}
	}

	if typeId == 0 || typeId == 2 {
		text += "\n✅ Замены обновлены"
	} else if typeId == 1 {
		text += "\n❌ Замены не обновлены"
	}

	logger.Debug("Текст расписания", zap.String("text", text))
	return text, nil
}

func GetSchedule(group int, userId int64) (string, error) {
	logger.Info("Получение расписания", zap.Int("group", group))

	idweek, typeweek, date, err := FetchLastWeekInfo()
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchLastWeekInfo()", zap.Error(err))
		return "", err
	}

	replaces, err := FetchReplaces(group, date)
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchReplaces()", zap.Error(err))
		return "", err
	}

	schedules, err := FetchSchedules(group, idweek, typeweek)
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchSchedules()", zap.Error(err))
		return "", err
	}

	var text, cacheText string

	// Функция для получения текста из кэша
	getTextFromCache := func(key string) (string, error) {
		text, err := cache.Rdb.Get(cache.Ctx, key).Result()
		if err == nil && text != "" {
			return text, nil
		}
		return "", err
	}

	// Попытка получить текст по userId, если userId не равен 0
	if userId != 0 {
		text, err := getTextFromCache("GetIDEditText_" + fmt.Sprint(userId))
		if err == nil {
			return text, nil
		}
	}

	// Попытка получить текст по group
	text, err = getTextFromCache("GetEditGroupText_" + fmt.Sprint(group))
	if err == nil {
		return text, nil
	}

	// Получение текста расписания, если текст не найден в кэше
	text, err = GetTextSchedule(0, group, date.Format("02.01.2006"), typeweek, idweek, schedules, replaces)
	if err != nil {
		logger.Error("Ошибка при выполнении функции GetTextSchedule()", zap.Error(err))
		return "", err
	}

	// Проверка, есть ли текст уже в кэше
	cacheText, err = getTextFromCache("GetTextSchedule_" + fmt.Sprint(group))
	if err == nil && cacheText == text {
		return cacheText, nil
	}

	// Сохранение текста в кэше
	err = cache.Rdb.Set(cache.Ctx, "GetTextSchedule_"+fmt.Sprint(group), text, 24*time.Hour).Err()
	if err != nil {
		return "", err
	}

	return text, nil
}

func GetNextSchedule(group int) (string, error) {
	logger.Info("Получение расписания на следующий день", zap.Int("group", group))

	idweek, typeweek, date, err := FetchLastWeekInfo()
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchLastWeekInfo()", zap.Error(err))
		return "", err
	}

	var nextDate time.Time
	idweek++
	if idweek == 7 {
		idweek = 1
		if typeweek == 0 {
			typeweek = 1
		}
		nextDate = date.AddDate(0, 0, 2)
	} else {
		nextDate = date.AddDate(0, 0, 1)
	}

	schedules, err := FetchSchedules(group, idweek, typeweek)
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchSchedules()", zap.Error(err))
		return "", err
	}

	text, err := GetTextSchedule(1, group, nextDate.Format("02.01.2006"), typeweek, idweek, schedules, nil)
	if err != nil {
		logger.Error("Ошибка при выполнении функции GetTextSchedule()", zap.Error(err))
		return "", err
	}

	return text, nil
}

func GetPrevSchedule(group int) (string, error) {
	logger.Info("Получение расписания на предыдущий день", zap.Int("group", group))

	idweek, typeweek, date, err := FetchLastWeekInfo()
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchLastWeekInfo()", zap.Error(err))
		return "", err
	}

	var prevDate time.Time
	idweek--
	if idweek == 0 {
		idweek = 6
		if typeweek == 0 {
			typeweek = 1
		}
		prevDate = date.AddDate(0, 0, -2)
	} else {
		prevDate = date.AddDate(0, 0, -1)
	}

	replaces, err := FetchReplaces(group, prevDate)
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchReplaces()", zap.Error(err))
		return "", err
	}

	schedules, err := FetchSchedules(group, idweek, typeweek)
	if err != nil {
		logger.Error("Ошибка при выполнении функции FetchSchedules()", zap.Error(err))
		return "", err
	}

	text, err := GetTextSchedule(2, group, prevDate.Format("02.01.2006"), typeweek, idweek, schedules, replaces)
	if err != nil {
		logger.Error("Ошибка при выполнении функции GetTextSchedule()", zap.Error(err))
		return "", err
	}

	return text, nil
}

func FetchLastWeekInfo() (int, int, time.Time, error) {
	var idWeek int
	var typeWeek int
	var date string
	var dateForm time.Time
	var err error

	data, err := cache.Rdb.Get(cache.Ctx, "FetchLastWeekInfo").Result()
	if err == nil && data != "" {
		var result []interface{}
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			logger.Error("Ошибка при преобразовании данных из Redis", zap.Error(err))
			return 0, 0, time.Time{}, err
		}
		dateForm, err = time.Parse(time.RFC3339, result[2].(string))
		if err != nil {
			logger.Error("Ошибка при преобразовании даты из строки", zap.Error(err))
			return 0, 0, time.Time{}, err
		}
		return int(result[0].(float64)), int(result[1].(float64)), dateForm, nil
	}

	// Данных нет в Redis, получаем их из базы данных
	rows, err := db.Conn.Queryx(`SELECT idweek,typeweek,date FROM hashes ORDER BY id DESC LIMIT 1`)
	if err != nil {
		logger.Error("Ошибка при выполнении запроса к таблице arrays", zap.Error(err))
		return 0, 0, time.Time{}, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&idWeek, &typeWeek, &date)
		if err != nil {
			logger.Error("Ошибка при сканировании данных из таблицы arrays", zap.Error(err))
			return 0, 0, time.Time{}, err
		}

		dateForm, err = time.Parse("2006-01-02T15:04:05Z", date)
		if err != nil {
			logger.Error("Ошибка при преобразовании даты из таблицы arrays", zap.Error(err))
			return 0, 0, time.Time{}, err
		}
	}
	// Сохраняем данные в Redis
	dataToStore, err := json.Marshal([]interface{}{idWeek, typeWeek, dateForm})
	if err != nil {
		logger.Error("Ошибка при преобразовании данных для сохранения в Redis", zap.Error(err))
		return idWeek, typeWeek, dateForm, err
	}

	err = cache.Rdb.Set(cache.Ctx, "FetchLastWeekInfo", dataToStore, 1*time.Hour).Err()
	if err != nil {
		return idWeek, typeWeek, dateForm, err
	}

	return idWeek, typeWeek, dateForm, nil
}

func FetchReplaces(group int, date time.Time) ([]map[string]interface{}, error) {
	var replacesSlice []map[string]interface{}

	replaces, err := cache.Rdb.Get(cache.Ctx, "Replaces_"+fmt.Sprint(group)+"_"+fmt.Sprint(date)).Result()
	if err == nil && replaces != "" {
		// Данные найдены в Redis, возвращаем их
		err = json.Unmarshal([]byte(replaces), &replacesSlice)
		if err != nil {
			return replacesSlice, err
		}

		for i := range replacesSlice {
			replacesSlice[i]["id"] = int64(replacesSlice[i]["id"].(float64))
			replacesSlice[i]["lesson"] = int64(replacesSlice[i]["lesson"].(float64))
		}

		return replacesSlice, nil
	}

	// Данных нет в Redis, получаем их из базы данных
	rows, err := db.Conn.Queryx(`SELECT * FROM replaces WHERE "group" = $1 AND date = $2`, group, date)
	if err != nil {
		logger.Error("Ошибка при выполнении запроса к таблице replaces в функции FetchReplaces()", zap.Error(err))
		return replacesSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("Ошибка при сканировании данных из таблицы replaces в функции FetchSchedules()", zap.Error(err))
			return replacesSlice, err
		}

		replacesSlice = append(replacesSlice, r)
	}

	// Сохраняем данные в Redis
	replacesJSON, err := json.Marshal(replacesSlice)
	if err != nil {
		return replacesSlice, err
	}
	err = cache.Rdb.Set(cache.Ctx, "Replaces_"+fmt.Sprint(group)+"_"+fmt.Sprint(date), replacesJSON, 15*time.Minute).Err()
	if err != nil {
		return replacesSlice, err
	}

	return replacesSlice, nil
}

func FetchSchedules(group int, idweek int, typeweek int) ([]map[string]interface{}, error) {
	var schedulesSlice []map[string]interface{}

	schedules, err := cache.Rdb.Get(cache.Ctx, "Schedules_"+fmt.Sprint(group)+"_"+fmt.Sprint(idweek)+"_"+fmt.Sprint(typeweek)).Result()
	if err == nil && schedules != "" {
		// Данные найдены в Redis, возвращаем их
		err = json.Unmarshal([]byte(schedules), &schedulesSlice)
		if err != nil {
			return schedulesSlice, err
		}

		for i := range schedulesSlice {
			schedulesSlice[i]["id"] = int64(schedulesSlice[i]["id"].(float64))
			schedulesSlice[i]["lesson"] = int64(schedulesSlice[i]["lesson"].(float64))
			schedulesSlice[i]["idweek"] = int64(schedulesSlice[i]["idweek"].(float64))
			schedulesSlice[i]["typeweek"] = int64(schedulesSlice[i]["typeweek"].(float64))
		}

		return schedulesSlice, nil
	}

	// Данных нет в Redis, получаем их из базы данных
	rows, err := db.Conn.Queryx(`SELECT * FROM schedules WHERE "group" = $1 AND idweek = $2 AND (typeweek = $3 OR typeweek = 0)`, group, idweek, typeweek)
	if err != nil {
		logger.Error("Ошибка при выполнении запроса к таблице schedules в функции FetchReplaces()", zap.Error(err))
		return schedulesSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		r := make(map[string]interface{})
		err := rows.MapScan(r)
		if err != nil {
			logger.Error("Ошибка при сканировании данных из таблицы replaces в функции FetchSchedules()", zap.Error(err))
			return schedulesSlice, err
		}

		schedulesSlice = append(schedulesSlice, r)
	}

	// Сохраняем данные в Redis
	schedulesJSON, err := json.Marshal(schedulesSlice)
	if err != nil {
		return schedulesSlice, err
	}
	err = cache.Rdb.Set(cache.Ctx, "Schedules_"+fmt.Sprint(group)+"_"+fmt.Sprint(idweek)+"_"+fmt.Sprint(typeweek), schedulesJSON, 1*time.Hour).Err()
	if err != nil {
		return schedulesSlice, err
	}

	return schedulesSlice, nil
}

func mergeSchedules(schedules, replaces []map[string]interface{}) []map[string]interface{} {
	// Создаем новую мапу для объединения данных
	mergedMap := make(map[int64]map[string]interface{})

	if len(replaces) > 0 {
		if replaces[0]["lesson"].(int64) == -1 {
			lesson := replaces[0]["lesson"].(int64)
			mergedMap[lesson] = replaces[0]

			result := make([]map[string]interface{}, 0, 1)
			result = append(result, mergedMap[lesson])

			return result
		}
	}

	// Заполняем мапу значениями из schedules
	for _, schedule := range schedules {
		lesson := schedule["lesson"].(int64)
		schedule["source"] = "schedules"
		mergedMap[lesson] = schedule
	}

	// Обновляем/добавляем значения из replaces
	for _, replace := range replaces {
		lesson := replace["lesson"].(int64)
		replace["source"] = "replaces"
		mergedMap[lesson] = replace
	}

	// Извлекаем ключи (уроки) и сортируем их
	lessons := make([]int64, 0, len(mergedMap))
	for lesson := range mergedMap {
		lessons = append(lessons, lesson)
	}
	sort.Slice(lessons, func(i, j int) bool {
		return lessons[i] < lessons[j]
	})

	// Создаем результирующий слайс
	result := make([]map[string]interface{}, 0, len(lessons))
	for _, lesson := range lessons {
		result = append(result, mergedMap[lesson])
	}

	return result
}
