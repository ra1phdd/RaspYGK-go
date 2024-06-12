package parser

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"raspygk/internal/telegram"
	"raspygk/pkg/db"
	"raspygk/pkg/logger"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

const (
	UrlFirst  = "https://menu.sttec.yar.ru/timetable/rasp_first.html"
	UrlSecond = "https://menu.sttec.yar.ru/timetable/rasp_second.html"
)

var (
	dataFirst  [][]string
	dataSecond [][]string
)

func Init() error {
	urls := []string{UrlFirst, UrlSecond}
	checks := make([]bool, len(urls))

	for i, url := range urls {
		date, idWeek, typeWeek, err := HandleData(url)
		if err != nil {
			logger.Error("ошибка при обработке данных замен", zap.Error(err))
			return err
		}
		if (i == 0 && dataFirst != nil) || (i == 1 && dataSecond != nil) {
			checks[i], err = InsertData(date, idWeek, typeWeek, i+1)
			if err != nil {
				logger.Error("ошибка при добавлении данных замен", zap.Error(err))
				return err
			}
		}
	}

	var idShift int
	if checks[0] && !checks[1] {
		logger.Info("Замены обновлены только у первой смены")
		idShift = 1
	} else if !checks[0] && checks[1] {
		logger.Info("Замены обновлены только у второй смены")
		idShift = 2
	} else if checks[0] && checks[1] {
		logger.Info("Замены обновлены у обоих смен")
		idShift = 0
		return nil
	} else {
		return nil
	}

	err := telegram.SendToPush(idShift)
	if err != nil {
		return err
	}

	return nil
}

func HandleData(url string) (string, int, int, error) {
	logger.Info("Парсинг HTML-расписания", zap.String("url", url))

	c := colly.NewCollector()

	var date string
	var idWeek, typeWeek int

	c.OnHTML("div > div:nth-of-type(2)", func(e *colly.HTMLElement) {
		date, idWeek = ParseDate(e)
	})

	c.OnHTML("div > div:nth-of-type(3)", func(e *colly.HTMLElement) {
		typeWeek = ParseTypeWeek(e)
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

	err := c.Visit(url)
	if err != nil {
		return "", 0, 0, err
	}

	c.Wait()

	result := chunkArray(data, 6)

	if url == UrlFirst {
		dataFirst = DataProccessing(result)
	} else {
		dataSecond = DataProccessing(result)
	}

	return date, idWeek, typeWeek, nil
}

func ParseDate(e *colly.HTMLElement) (string, int) {
	search := []string{"в расписании на ", " года / ", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

	text := e.Text
	for _, s := range search {
		text = strings.Replace(text, s, "", -1)
	}

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

	text = strings.Replace(text, " ", ".", -1)

	t, _ := time.Parse("2.01.2006", text)
	date := t.Format("2006-01-02")

	week := t.Weekday()
	idWeek := int(week)

	return date, idWeek
}

func ParseTypeWeek(e *colly.HTMLElement) int {
	search := []string{"(", ") Первая смена", ") Вторая смена"}

	text := e.Text
	for _, s := range search {
		text = strings.Replace(text, s, "", -1)
	}

	var typeWeek int
	switch text {
	case "Числитель":
		typeWeek = 1
	case "Знаменатель":
		typeWeek = 2
	}

	return typeWeek
}

func DataProccessing(result [][]string) [][]string {
	var data [][]string

	for _, item := range result {
		if item[1] == "" || item[1] == " " {
			continue
		}

		item[3] = strings.Replace(item[3], "...", "по расписанию", -1)

		values2 := strings.Split(item[2], ",") // Номер пары
		values3 := strings.Split(item[3], ",") // Дисциплина по расписанию
		values5 := strings.Split(item[5], ",") // Кабинет

		for i := 0; i < len(values2); i++ {
			newItem := make([]string, len(item))
			copy(newItem, item)

			newItem[2] = values2[i]

			if i < len(values3) {
				newItem[3] = values3[i]
			}
			if i < len(values5) {
				newItem[5] = values5[i]
			}

			if !strings.Contains(newItem[2], "-") {
				data = append(data, newItem)
				continue
			}

			rangeValues := strings.Split(newItem[2], "-")
			if len(rangeValues) != 2 {
				continue
			}

			start, _ := strconv.Atoi(rangeValues[0])
			end, _ := strconv.Atoi(rangeValues[1])

			for j := start; j <= end; j++ {
				newItemCopy := make([]string, len(newItem))
				copy(newItemCopy, newItem)
				newItemCopy[2] = strconv.Itoa(j)
				data = append(data, newItemCopy)
			}
		}
	}

	return data
}

func InsertData(date string, idWeek int, typeWeek int, idShift int) (bool, error) {
	var column string
	var data [][]string

	switch idShift {
	case 1:
		column = "result_first"
		data = dataFirst
	case 2:
		column = "result_second"
		data = dataSecond
	default:
		logger.Error("неверный id shift")
		return false, errors.New("неверный id shift")
	}

	hashData, err := GetMD5Hash(data)
	if err != nil {
		logger.Error("ошибка при получении хеша массива данных", zap.Error(err))
	}

	rows, err := db.Conn.Queryx(`SELECT id,result_first,result_second FROM hashes WHERE date = $1`, date)
	if err != nil {
		logger.Error("ошибка при обновлении2 к БД arrays", zap.Error(err))
		return false, err
	}
	defer rows.Close()

	var resultFirst, resultSecond sql.NullString
	var idArrays int

	// Проверяем, есть ли результаты
	if !rows.Next() {
		query := "INSERT INTO hashes (" + column + ", idweek, typeweek, date) VALUES ($1, $2, $3, $4) RETURNING id"
		err := db.Conn.QueryRow(query, column, idWeek, typeWeek, date).Scan(&idArrays)
		if err != nil {
			logger.Error("ошибка при обновлении1 к БД arrays", zap.Error(err))
			return false, err
		}
	} else {
		err := rows.Scan(&idArrays, &resultFirst, &resultSecond)
		if err != nil {
			logger.Error("ошибка при обработке данных из таблицы users в функции getUserData", zap.Error(err))
			return false, err
		}
	}

	var query string
	if (idShift == 1 && resultFirst.String != hashData) || (idShift == 2 && resultSecond.String != hashData) {
		logger.Debug("Обновление существующего хеша в таблице arrays", zap.String("column", column))
		query = "UPDATE hashes SET " + column + " = $1, idweek = $2, typeweek = $3 WHERE date = $4 RETURNING id"
	} else {
		logger.Debug("Хеш совпадает, пропуск")
		return false, nil
	}
	err = db.Conn.QueryRow(query, hashData, idWeek, typeWeek, date).Scan(&idArrays)
	if err != nil {
		logger.Error("Ошибка при получении id после вставки", zap.Error(err))
		return false, err
	}

	if resultFirst.Valid || resultSecond.Valid {
		_, err = db.Conn.Exec(`DELETE FROM replaces WHERE date = $1 AND "group" IN (SELECT id FROM groups WHERE idshift = $2)`, date, idShift)
		if err != nil {
			logger.Error("ошибка при удалении строки в БД arrays", zap.Error(err))
			return false, err
		}
	}

	for _, item := range data {
		var lesson int

		textLesson := strings.Replace(item[2], ".", "", -1)
		lesson, err := strconv.Atoi(textLesson)
		if err != nil {
			logger.Error("Ошибка преобразования lesson в функции InsertData()")
		}

		_, err = db.Conn.Exec(`INSERT INTO replaces ("group", lesson, discipline_rasp, discipline_replace, classroom, idarrays, date) VALUES ((SELECT id FROM groups WHERE name = $1), $2, $3, $4, $5, $6, $7)`, strings.TrimSpace(item[1]), lesson, item[3], item[4], item[5], idArrays, date)
		if err != nil {
			logger.Error("ошибка при вставке к БД arrays (name group - "+item[1]+")", zap.Error(err))
			return false, err
		}
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

func GetMD5Hash(data [][]string) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:]), nil
}
