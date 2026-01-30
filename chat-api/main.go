package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib" //докер ругается если не объявлять

	"chat-api/internal/config"
	"chat-api/internal/database"
	"chat-api/internal/handlers"
	"chat-api/internal/middleware"
)

func main() {
	// подставляем конфиг данные по бд
	cfg := config.Load()

	//инициация бд
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// от горилы маршрутизатор по url
	r := mux.NewRouter()

	// Мидлвары
	r.Use(middleware.Logging)
	r.Use(middleware.JSONContentType)

	//с пакета обработчиков инициализируется
	handlers.InitHandlers(r, db)

	// сервер запускается на порту из конфига
	srv := &http.Server{
		Handler:      r, // в качестве хендлера горилавские обработчики
		Addr:         ":" + cfg.ServerPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	log.Fatal(srv.ListenAndServe()) //при ошибке создать серв
}
