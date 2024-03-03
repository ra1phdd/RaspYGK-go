package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

const (
	URL_FIRST  = "https://menu.sttec.yar.ru/timetable/rasp_first.html"
	URL_SECOND = "https://menu.sttec.yar.ru/timetable/rasp_second.html"
)

var (
	newData_first  [][]string
	newData_second [][]string
	oldData_first  [][]string
	oldData_second [][]string
)

func Parser() error {
	urls := []string{URL_FIRST, URL_SECOND}
	checks := make([]bool, len(urls))

	for i, url := range urls {
		date, id_week, type_week, err := DataProcessing(url)
		logger.Debug("Дата в цикле", zap.String("date", date))
		if err != nil {
			return handleError("ошибка при обработке данных замен", err)
		}
		if (i == 0 && newData_first != nil) || (i == 1 && newData_second != nil) {
			checks[i], err = InsertData(date, id_week, type_week, i+1)
			if err != nil {
				return handleError("ошибка при добавлении данных замен", err)
			}
		}
	}

	if checks[0] && !checks[1] {
		logger.Info("Замены обновлены только у первой смены")
		SendToPush(1)
	} else if !checks[0] && checks[1] {
		logger.Info("Замены обновлены только у второй смены")
		SendToPush(2)
	} else if checks[0] && checks[1] {
		logger.Info("Замены обновлены у обоих смен")
		SendToPush(0)
	}

	return nil
}

func DataProcessing(url string) (string, int, string, error) {
	logger.Info("Парсинг HTML-расписания", zap.String("url", url))

	c := colly.NewCollector()

	var date string
	var id_week int
	var type_week string

	// Нахождение даты
	c.OnHTML("div > div:nth-of-type(2)", func(e *colly.HTMLElement) {
		logger.Debug("Получение string с датой", zap.String("text", e.Text))
		text := e.Text

		search := []string{"в расписании на ", " года / ", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

		for _, s := range search {
			text = strings.Replace(text, s, "", -1)
		}

		logger.Debug("Обработанный string с датой", zap.String("text", text))

		months := map[string]string{
			"января":   "01",
			"февраля":  "02",
			"марта":    "03",
			"апреля":   "04",
			"мая":      "05",
			"июня":     "06",
			"июля":     "07",
			"августа":  "08",
			"сентября": "09",
			"октября":  "10",
			"ноября":   "11",
			"декабря":  "12",
		}

		for month, number := range months {
			text = strings.Replace(text, month, number, -1)
		}

		logger.Debug("Замена месяца в дате на числовой формат", zap.String("text", text))

		text = strings.Replace(text, " ", ".", -1)

		t, _ := time.Parse("2.01.2006", text)
		date = t.Format("2006-01-02")

		week := t.Weekday()
		id_week = int(week)

		logger.Debug("Дата", zap.String("date", date), zap.Int("id_week", id_week))
	})

	// Нахождение числителя и знаменателя
	c.OnHTML("div > div:nth-of-type(3)", func(e *colly.HTMLElement) {
		text := e.Text

		search := []string{"(", ") Первая смена", ") Вторая смена"}

		for _, s := range search {
			text = strings.Replace(text, s, "", -1)
		}

		type_week = text

		logger.Debug("Тип недели", zap.String("type_week", type_week))
	})

	data := make([]string, 0)

	skip := true
	c.OnHTML("tr", func(e *colly.HTMLElement) {
		if skip {
			skip = false
		} else {
			e.ForEach("td", func(_ int, el *colly.HTMLElement) {
				data = append(data, el.Text)
			})
		}
	})

	c.Visit(url)

	result := chunkArray(data, 6)

	var newData [][]string

	for _, item := range result {
		if item[2] != "" {
			item[3] = strings.Replace(item[3], "...", "", -1)

			values2 := strings.Split(item[2], ",")
			values3 := strings.Split(item[3], ",")
			values5 := strings.Split(item[5], ",")

			for i := 0; i < len(values2); i++ {
				newItem := make([]string, len(item))
				copy(newItem, item)

				newItem[2] = getValue(values2, i)
				newItem[3] = getValue(values3, i)
				newItem[5] = getValue(values5, i)

				if strings.Contains(newItem[2], "-") {
					rangeValues := strings.Split(newItem[2], "-")
					if len(rangeValues) > 1 {
						start, _ := strconv.Atoi(rangeValues[0])
						end, _ := strconv.Atoi(rangeValues[1])

						for j := start; j <= end; j++ {
							newItem[2] = strconv.Itoa(j)
							newData = append(newData, newItem)
						}
					}
				} else {
					newData = append(newData, newItem)
				}
			}
		}
	}
	if url == URL_FIRST {
		newData_first = newData
	} else {
		newData_second = newData
	}

	logger.Debug("Массив с расписанием", zap.Any("newData", newData))
	return date, id_week, type_week, nil
}

func InsertData(date string, id_week int, type_week string, id_shift int) (bool, error) {
	logger.Info("Добавление данных в БД", zap.String("date", date), zap.Int("id_week", id_week), zap.String("type_week", type_week), zap.Int("id_shift", id_shift))

	var column string
	switch id_shift {
	case 1:
		column = "result_first"
	case 2:
		column = "result_second"
	default:
		return false, handleError("неверный id_shift", nil)
	}

	var data [][]string
	if column == "result_first" {
		data = newData_first
	} else {
		data = newData_second
	}

	hashData, err := GetMD5Hash(data)
	if err != nil {
		logger.Error("ошибка при получении хеша массива данных", zap.Error(err))
	}
	logger.Debug("Получение хеша массива данных", zap.String("hashData", hashData))

	rows, err := Conn.Queryx(`SELECT result_first,result_second FROM arrays WHERE date = $1`, date)
	if err != nil && err != sql.ErrNoRows {
		return false, handleError("ошибка при обновлении к БД arrays", err)
	}
	defer rows.Close()

	var result_first sql.NullString
	var result_second sql.NullString
	for rows.Next() {
		rows.Scan(&result_first, &result_second)
	}

	logger.Debug("Полученные данные из БД", zap.String("result_first", result_first.String), zap.String("result_second", result_second.String))

	var query string
	if !result_first.Valid && !result_second.Valid {
		logger.Debug("Добавление хеша в таблицу arrays", zap.String("column", column))
		query = fmt.Sprintf(`INSERT INTO arrays (%s, idweek, typeweek, date) VALUES ($1, $2, $3, $4)`, column)
	} else if (column == "result_first" && result_first.String != hashData) || (column == "result_second" && result_second.String != hashData) {
		logger.Debug("Обновление существующего хеша в таблице arrays", zap.String("column", column))
		query = fmt.Sprintf(`UPDATE arrays SET %s = $1, idweek = $2, typeweek = $3 WHERE date = $4`, column)
	} else {
		logger.Debug("Хеш совпадает, пропуск")
		return false, nil
	}

	logger.Debug("Запрос к БД", zap.String("query", query))
	_, err = Conn.Exec(query, hashData, id_week, type_week, date)
	if err != nil {
		return false, handleError("ошибка при обновлении к БД arrays", err)
	}

	if result_first.Valid || result_second.Valid {
		logger.Debug("Удаление всех замен из таблицы replaces", zap.String("date", date), zap.Int("id_shift", id_shift))
		_, err = Conn.Exec(`DELETE FROM replaces WHERE date = $1 AND idshift = $2`, date, id_shift)
		if err != nil {
			return false, handleError("ошибка при удалении к БД replaces", err)
		}
	}

	logger.Debug("Добавление замен в таблицу replaces", zap.Any("data", data))
	for _, item := range data {
		var lesson any
		if item[2] == "07.40" {
			lesson = 10
		} else if item[2] == "08.00" {
			lesson = 0
		} else if item[2] == "08.25" {
			lesson = 11
		} else {
			lesson = item[2]
		}
		logger.Info("Добавление замены", zap.Any("item", lesson))
		_, err := Conn.Exec(`INSERT INTO replaces ("group", lesson, discipline_rasp, discipline_replace, classroom, date, idshift) VALUES ($1, $2, $3, $4, $5, $6, $7)`, item[1], lesson, item[3], item[4], item[5], date, id_shift)
		if err != nil {
			return false, handleError("ошибка при вставке в БД replaces", err)
		}
	}

	logger.Info("1")
	if column == "result_first" {
		oldData_first = data
		logger.Info("Новый массив с расписанием1", zap.Any("data", oldData_first))
	} else {
		oldData_second = data
		logger.Info("Новый массив с расписанием2", zap.Any("data", oldData_second))
	}

	return true, nil
}

func chunkArray(data []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func getValue(values []string, i int) string {
	if i < len(values) {
		return values[i]
	}
	return values[len(values)-1]
}

func GetMD5Hash(data [][]string) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:]), nil
}
