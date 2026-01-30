package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"chat-api/internal/models"
)

type Handler struct {
	DB *gorm.DB
}

func InitHandlers(r *mux.Router, db *gorm.DB) {
	h := &Handler{DB: db}

	r.HandleFunc("/chats", h.CreateChat).Methods("POST")
	r.HandleFunc("/chats/{id}/messages", h.CreateMessage).Methods("POST")
	r.HandleFunc("/chats/{id}", h.GetChat).Methods("GET")
	r.HandleFunc("/chats/{id}", h.DeleteChat).Methods("DELETE")
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request - body", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(request.Title)
	if title == "" || len(title) > 200 {
		http.Error(w, "Title must be between 1 and 200 characters", http.StatusBadRequest)
		return
	}

	chat := models.Chat{
		Title:     title,
		CreatedAt: time.Now(),
	}
	if err := h.DB.Create(&chat).Error; err != nil {
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chat)
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	// проверка существования чата
	var chat models.Chat
	if err := h.DB.First(&chat, chatID).Error; err != nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	var request struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	//проверка на длинну
	text := strings.TrimSpace(request.Text)
	if text == "" || len(text) > 5000 {
		http.Error(w, "Text must be between 1 and 5000 characters", http.StatusBadRequest)
		return
	}

	message := models.Message{
		ChatID:    uint(chatID),
		Text:      text,
		CreatedAt: time.Now(),
	}

	if err := h.DB.Create(&message).Error; err != nil {
		http.Error(w, "Failed to create message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

func (h *Handler) GetChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	//получение лимита из query параметров
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}

	//поиск чата по ид
	var chat models.Chat
	if err := h.DB.First(&chat, chatID).Error; err != nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	//последние сообщения
	var messages []models.Message
	h.DB.Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	//порядок для правильной сортировки
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	response := struct {
		models.Chat
		Messages []models.Message `json:"messages"`
	}{
		Chat:     chat,
		Messages: messages,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	//(сообщения удалятся каскадно из-за constraint)
	result := h.DB.Delete(&models.Chat{}, chatID)
	if result.Error != nil {
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") //проверка на жизнь сервера и отправка ответа при запросе
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
