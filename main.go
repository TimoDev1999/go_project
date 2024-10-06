package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"todo/db"
	"todo/handlers"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	webDir := "./web"

	database, err := db.InitDB()
	if err != nil {
		log.Printf("Ошибка инициализации базы данных: %v", err)
		return
	}
	defer database.Close()

	http.Handle("/", http.FileServer(http.Dir(webDir)))

	http.HandleFunc("/api/nextdate", handlers.NextDateHandler)

	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		handlers.WorkWithTask(w, r, database)
	})

	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetTasksHandler(w, r, database)
	})

	http.HandleFunc("/api/task/done", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.DoneHandler(w, r, database)
		} else {
			http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Слушаем на порту " + port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
