package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"todo/models"
	"todo/services"
)

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowParam := r.FormValue("now")
	dateParam := r.FormValue("date")
	repeatParam := r.FormValue("repeat")

	if nowParam == "" || dateParam == "" || repeatParam == "" {
		http.Error(w, "не заданы параметры", http.StatusBadRequest)
		return
	}
	now, err := time.Parse("20060102", nowParam)
	if err != nil {
		http.Error(w, "неверный формат времени", http.StatusBadRequest)
		return
	}
	nextDate, err := services.NextDate(now, dateParam, repeatParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, nextDate)
}

func WorkWithTask(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	switch r.Method {
	case http.MethodPost:
		AddTaskHandler(w, r, database)
	case http.MethodPut:
		UpdateTaskHandler(w, r, database)
	case http.MethodGet:
		GetTaskByIdHandler(w, r, database)
	case http.MethodDelete:
		DeleteTask(w, r, database)
	default:
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}

func AddTaskHandler(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	var task models.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error":"Ошибка десериализации JSON"}`, http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, `{"error":"Не указан заголовок задачи"}`, http.StatusBadRequest)
		return
	}

	if task.Date == "" || task.Date == "today" {
		task.Date = time.Now().Format("20060102")
	}

	if err := validateDateFormat(task.Date); err != nil {
		http.Error(w, `{"error":"Неверный формат даты"}`, http.StatusBadRequest)
		return
	}

	taskDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		http.Error(w, `{"error":"Неверная дата"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	taskDateTruncated := taskDate.Truncate(24 * time.Hour)
	if taskDateTruncated.Equal(today) {
		task.Date = today.Format("20060102")
	} else if taskDate.Before(now) {
		if task.Repeat == "" {
			task.Date = now.Format("20060102")
		} else {
			nextDate, err := services.NextDate(now, task.Date, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Ошибка при вычислении следующей даты"}`, http.StatusBadRequest)
				return
			}
			task.Date = nextDate
		}
	}

	query := "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)"
	res, err := database.Exec(query, task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		http.Error(w, `{"error":"Ошибка записи в базу данных"}`, http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, `{"error":"Ошибка получения ID задачи"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"id": fmt.Sprintf("%d", id),
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(response)
}
func GetTaskByIdHandler(w http.ResponseWriter, r *http.Request, database *sql.DB) {

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	var task models.Task
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?"
	row := database.QueryRow(query, id)
	var taskId int64

	err := row.Scan(&taskId, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Ошибка при поиске задачи"}`, http.StatusInternalServerError)
		}
		return
	}

	task.Id = fmt.Sprint(taskId)

	response := struct {
		Id      string `json:"id"`
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}{
		Id:      task.Id,
		Date:    task.Date,
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetTasksHandler(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	var tasks []models.Task

	err := getTasks(database, &tasks)
	if err != nil {
		logAndReturnError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	resp := map[string]interface{}{
		"tasks": tasks,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	var task models.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error":"Ошибка десериализации JSON"}`, http.StatusBadRequest)
		return
	}

	if task.Id == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, `{"error":"Не указан заголовок задачи"}`, http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format("20060102")
	}

	if err := validateDateFormat(task.Date); err != nil {
		http.Error(w, `{"error":"Неверный формат даты"}`, http.StatusBadRequest)
		return
	}

	taskDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		http.Error(w, `{"error":"Неверная дата"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	if taskDate.Before(now) {
		if task.Repeat != "" {
			nextDate, err := services.NextDate(now, task.Date, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Ошибка при вычислении следующей даты"}`, http.StatusBadRequest)
				return
			}
			task.Date = nextDate
		} else {
			task.Date = now.Format("20060102")
		}
	}

	query := "UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?"
	res, err := database.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		http.Error(w, `{"error":"Ошибка обновления в базе данных"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, `{"error":"Ошибка при получении результата обновления"}`, http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func DoneHandler(w http.ResponseWriter, r *http.Request, database *sql.DB) {

	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	var task models.Task

	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?"

	err := database.QueryRow(query, taskID).Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error":"Ошибка при получении данных задачи"}`, http.StatusInternalServerError)
		return
	}

	if task.Repeat != "" {

		now := time.Now()
		nextDate, err := services.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			http.Error(w, `{"error":"Ошибка при вычислении следующей даты"}`, http.StatusBadRequest)
			return
		}

		updateQuery := "UPDATE scheduler SET date = ? WHERE id = ?"
		_, err = database.Exec(updateQuery, nextDate, taskID)
		if err != nil {
			http.Error(w, `{"error":"Ошибка при обновлении задачи в базе данных"}`, http.StatusInternalServerError)
			return
		}
	} else {

		deleteQuery := "DELETE FROM scheduler WHERE id = ?"
		_, err := database.Exec(deleteQuery, taskID)
		if err != nil {
			http.Error(w, `{"error":"Ошибка при удалении задачи из базы данных"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func DeleteTask(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	query := "DELETE FROM scheduler WHERE id = ?"
	res, err := database.Exec(query, taskID)
	if err != nil {
		http.Error(w, `{"error":"Ошибка при удалении задачи"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, `{"error":"Ошибка получения информации об удалении"}`, http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func getTasks(database *sql.DB, tasks *[]models.Task) error {
	rows, err := database.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT 25")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var task models.Task
		var id int64

		err := rows.Scan(&id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return err
		}

		task.Id = fmt.Sprint(id)
		*tasks = append(*tasks, task)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func validateDateFormat(date string) error {
	_, err := time.Parse("20060102", date)
	return err
}

func logAndReturnError(w http.ResponseWriter, msg string, code int) {
	log.Printf("Ошибка: %s, код ответа: %d", msg, code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}
