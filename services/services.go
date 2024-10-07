package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {

	dateTime, err := time.Parse("20060102", date)
	if err != nil {
		return "", fmt.Errorf("ошибка при разборе даты: %v", err)
	}
	if repeat == "" {
		return "", fmt.Errorf("правило повторения не задано")
	}

	val := strings.Split(repeat, " ")
	if len(val) < 1 {
		return "", fmt.Errorf("неправильный формат правила повторения")
	}

	switch val[0] {
	case "d":
		if len(val) < 2 {
			return "", fmt.Errorf("не указано количество дней")
		}
		dateAdd, err := strconv.Atoi(val[1])
		if err != nil || dateAdd < 1 || dateAdd > 400 {
			return "", fmt.Errorf("некорректное значение интервала для дней")
		}
		if dateTime.After(now) {
			dateTime = dateTime.AddDate(0, 0, dateAdd)
		}
		for dateTime.Before(now) {
			dateTime = dateTime.AddDate(0, 0, dateAdd)
		}

		return dateTime.Format("20060102"), nil

	case "y":

		if dateTime.After(now) {
			dateTime = dateTime.AddDate(1, 0, 0)
		}

		for dateTime.Before(now) {
			dateTime = dateTime.AddDate(1, 0, 0)
		}
		nextDateValue := dateTime.Format("20060102")
		return nextDateValue, nil

	case "w":
		if len(val) < 2 {
			return "", fmt.Errorf("не указаны дни недели")
		}
		weekdays := strings.Split(val[1], ",")
		weekdaysInt := make([]int, len(weekdays))
		for i, w := range weekdays {
			day, err := strconv.Atoi(w)
			if err != nil || day < 1 || day > 7 {
				return "", fmt.Errorf("некорректное значение для дней недели")
			}
			weekdaysInt[i] = day
		}
		for {
			weekday := int(dateTime.Weekday())
			if weekday == 0 {
				weekday = 1
			}
			for _, w := range weekdaysInt {
				if w == weekday && dateTime.After(now) {

					return dateTime.Format("20060102"), nil
				}
			}
			dateTime = dateTime.AddDate(0, 0, 1)
		}
	case "m":

		if len(val) < 2 {
			return "", fmt.Errorf("не указаны дни месяца")
		}
		days := strings.Split(val[1], ",")
		months := []int{}
		if len(val) > 2 {
			monthVals := strings.Split(val[2], ",")
			for _, m := range monthVals {
				month, err := strconv.Atoi(m)
				if err != nil || month < 1 || month > 12 {
					return "", fmt.Errorf("некорректное значение для месяцев")
				}
				months = append(months, month)
			}
		}

		for {
			day := dateTime.Day()
			month := int(dateTime.Month())

			for _, d := range days {
				dayInt, err := strconv.Atoi(d)
				if err != nil {
					return "", fmt.Errorf("некорректное значение для дней месяца")
				}
				if dayInt == -1 {
					lastDay := time.Date(dateTime.Year(), dateTime.Month()+1, 0, 0, 0, 0, 0, dateTime.Location()).Day()
					dayInt = lastDay
				} else if dayInt < 1 || dayInt > 31 {
					return "", fmt.Errorf("некорректное значение для дней месяца")
				}

				if dayInt == day && (len(months) == 0 || contains(months, month)) && dateTime.After(now) {
					return dateTime.Format("20060102"), nil
				}
			}
			dateTime = dateTime.AddDate(0, 0, 1)
		}

	default:
		return "", fmt.Errorf("неподдерживаемый формат правила повторения")
	}
}
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
