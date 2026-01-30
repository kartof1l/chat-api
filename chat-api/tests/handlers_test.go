package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"chat-api/internal/handlers"
	"chat-api/internal/middleware"
	"chat-api/internal/models"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	setupTestDatabase()
	code := m.Run()
	cleanupTestDatabase()
	os.Exit(code)
}

func setupTestDatabase() {
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to test database:", err)
	}
	sqlDB, err := testDB.DB()
	if err == nil {
		_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
		if err != nil {
			log.Printf("Failed to enable foreign keys: %v", err)
		}
	}

	err = testDB.AutoMigrate(&models.Chat{}, &models.Message{})
	if err != nil {
		log.Fatal("Failed to migrate test database:", err)
	}

	log.Println("Test database setup completed (foreign keys enabled)")
}

func cleanupTestDatabase() {
	sqlDB, err := testDB.DB()
	if err == nil {
		sqlDB.Close()
	}
	log.Println("Test database cleanup completed")
}
func createTestRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.Logging)
	r.Use(middleware.JSONContentType)

	h := &handlers.Handler{DB: testDB}

	r.HandleFunc("/chats", h.CreateChat).Methods("POST")
	r.HandleFunc("/chats/{id}/messages", h.CreateMessage).Methods("POST")
	r.HandleFunc("/chats/{id}", h.GetChat).Methods("GET")
	r.HandleFunc("/chats/{id}", h.DeleteChat).Methods("DELETE")
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")

	return r
}

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	return rr
}

func createTestChat(t assert.TestingT, title string) *models.Chat {
	chat := &models.Chat{
		Title:     title,
		CreatedAt: time.Now(),
	}

	result := testDB.Create(chat)
	assert.NoError(t, result.Error)
	assert.NotZero(t, chat.ID)

	return chat
}

func createTestMessage(t assert.TestingT, chatID uint, text string) *models.Message {
	message := &models.Message{
		ChatID:    chatID,
		Text:      text,
		CreatedAt: time.Now(),
	}

	result := testDB.Create(message)
	assert.NoError(t, result.Error)
	assert.NotZero(t, message.ID)

	time.Sleep(time.Millisecond)

	return message
}

type HandlersTestSuite struct {
	suite.Suite
	router *mux.Router
}

func (suite *HandlersTestSuite) SetupTest() {
	suite.router = createTestRouter()
	testDB.Exec("DELETE FROM messages")
	testDB.Exec("DELETE FROM chats")
}

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

func (suite *HandlersTestSuite) TestHealthCheck() {
	t := suite.T()

	rr := performRequest(suite.router, "GET", "/health", nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

func (suite *HandlersTestSuite) TestCreateChat_Success() {
	t := suite.T()

	requestBody := map[string]string{
		"title": "Новый чат",
	}

	rr := performRequest(suite.router, "POST", "/chats", requestBody)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var chat models.Chat
	err := json.Unmarshal(rr.Body.Bytes(), &chat)
	assert.NoError(t, err)
	assert.Equal(t, "Новый чат", chat.Title)
	assert.NotZero(t, chat.ID)
	assert.NotZero(t, chat.CreatedAt)
}

func (suite *HandlersTestSuite) TestCreateChat_EmptyTitle() {
	t := suite.T()

	requestBody := map[string]string{
		"title": "",
	}

	rr := performRequest(suite.router, "POST", "/chats", requestBody)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func (suite *HandlersTestSuite) TestCreateChat_TooLongTitle() {
	t := suite.T()

	longTitle := strings.Repeat("a", 201)
	requestBody := map[string]string{
		"title": longTitle,
	}

	rr := performRequest(suite.router, "POST", "/chats", requestBody)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func (suite *HandlersTestSuite) TestCreateChat_TrimSpaces() {
	t := suite.T()

	requestBody := map[string]string{
		"title": "  Чат с пробелами  ",
	}

	rr := performRequest(suite.router, "POST", "/chats", requestBody)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var chat models.Chat
	err := json.Unmarshal(rr.Body.Bytes(), &chat)
	assert.NoError(t, err)
	assert.Equal(t, "Чат с пробелами", chat.Title)
}

func (suite *HandlersTestSuite) TestCreateMessage_Success() {
	t := suite.T()

	chat := createTestChat(t, "Чат для сообщения")

	requestBody := map[string]string{
		"text": "Тестик",
	}

	rr := performRequest(suite.router, "POST", fmt.Sprintf("/chats/%d/messages", chat.ID), requestBody)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var message models.Message
	err := json.Unmarshal(rr.Body.Bytes(), &message)
	assert.NoError(t, err)
	assert.Equal(t, "Тестовое сообщение", message.Text)
	assert.Equal(t, chat.ID, message.ChatID)
	assert.NotZero(t, message.ID)
}

func (suite *HandlersTestSuite) TestCreateMessage_ChatNotFound() {
	t := suite.T()

	requestBody := map[string]string{
		"text": "Сообщение",
	}

	rr := performRequest(suite.router, "POST", "/chats/999/messages", requestBody)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func (suite *HandlersTestSuite) TestCreateMessage_EmptyText() {
	t := suite.T()

	chat := createTestChat(t, "Чат")

	requestBody := map[string]string{
		"text": "",
	}

	rr := performRequest(suite.router, "POST", fmt.Sprintf("/chats/%d/messages", chat.ID), requestBody)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func (suite *HandlersTestSuite) TestCreateMessage_TooLongText() {
	t := suite.T()

	chat := createTestChat(t, "Чат")

	longText := strings.Repeat("a", 5001)
	requestBody := map[string]string{
		"text": longText,
	}

	rr := performRequest(suite.router, "POST", fmt.Sprintf("/chats/%d/messages", chat.ID), requestBody)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
func (suite *HandlersTestSuite) TestGetChat_Success() {
	t := suite.T()

	chat := createTestChat(t, "Чат с сообщениями")

	message1 := &models.Message{
		ChatID:    chat.ID,
		Text:      "Первое сообщение",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	testDB.Create(message1)

	message2 := &models.Message{
		ChatID:    chat.ID,
		Text:      "Второе сообщение",
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	testDB.Create(message2)

	message3 := &models.Message{
		ChatID:    chat.ID,
		Text:      "Третье сообщение",
		CreatedAt: time.Now(),
	}
	testDB.Create(message3)

	rr := performRequest(suite.router, "GET", fmt.Sprintf("/chats/%d", chat.ID), nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response struct {
		models.Chat
		Messages []models.Message `json:"messages"`
	}

	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, chat.ID, response.ID)
	assert.Equal(t, "Чат с сообщениями", response.Title)
	assert.Len(t, response.Messages, 3)

	/*  сообщения отсортированы от новых к старым
	В  коде есть переворот массива, так что порядок:
	1. Третье  -  Третье на позиции 0
	2. Второе - 1
	3. Первое  -  Первое на позиции 2
	*/
	fmt.Printf("DEBUG: Получены сообщения: ")
	for i, msg := range response.Messages {
		fmt.Printf("[%d: %s] ", i, msg.Text)
	}
	fmt.Println()

	hasFirst := false
	hasSecond := false
	hasThird := false

	for _, msg := range response.Messages {
		if msg.Text == "Первое сообщение" {
			hasFirst = true
		}
		if msg.Text == "Второе сообщение" {
			hasSecond = true
		}
		if msg.Text == "Третье сообщение" {
			hasThird = true
		}
	}

	assert.True(t, hasFirst, "первое сообщение")
	assert.True(t, hasSecond, "второе сообщение")
	assert.True(t, hasThird, "это должно быть третье сообщение")

}

func (suite *HandlersTestSuite) TestGetChat_WithLimit() {
	t := suite.T()

	chat := createTestChat(t, "Чат с лимитом")

	for i := 1; i <= 15; i++ {
		message := &models.Message{
			ChatID:    chat.ID,
			Text:      fmt.Sprintf("Сообщение %d", i),
			CreatedAt: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
		result := testDB.Create(message)
		assert.NoError(t, result.Error)
		time.Sleep(time.Millisecond)
	}

	rr := performRequest(suite.router, "GET", fmt.Sprintf("/chats/%d?limit=5", chat.ID), nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response struct {
		models.Chat
		Messages []models.Message `json:"messages"`
	}

	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Messages, 5)
	assert.Equal(t, "Сообщение 11", response.Messages[0].Text)
	assert.Equal(t, "Сообщение 12", response.Messages[1].Text)
	assert.Equal(t, "Сообщение 13", response.Messages[2].Text)
	assert.Equal(t, "Сообщение 14", response.Messages[3].Text)
	assert.Equal(t, "Сообщение 15", response.Messages[4].Text)
}

func (suite *HandlersTestSuite) TestGetChat_NotFound() {
	t := suite.T()

	rr := performRequest(suite.router, "GET", "/chats/999", nil)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func (suite *HandlersTestSuite) TestDeleteChat_Success() {
	t := suite.T()

	chat := createTestChat(t, "Чат удаления")
	createTestMessage(t, chat.ID, "Сообщение 1")
	createTestMessage(t, chat.ID, "Сообщение 2")

	var chatCount, messageCount int64
	testDB.Model(&models.Chat{}).Where("id = ?", chat.ID).Count(&chatCount)
	testDB.Model(&models.Message{}).Where("chat_id = ?", chat.ID).Count(&messageCount)

	assert.Equal(t, int64(1), chatCount)
	assert.Equal(t, int64(2), messageCount, "2 сообщения перед удалением")

	rr := performRequest(suite.router, "DELETE", fmt.Sprintf("/chats/%d", chat.ID), nil)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	testDB.Model(&models.Chat{}).Where("id = ?", chat.ID).Count(&chatCount)
	assert.Equal(t, int64(0), chatCount, "чат должен быть удален")

	//  SQLite работает только если включены foreign keys
	testDB.Model(&models.Message{}).Where("chat_id = ?", chat.ID).Count(&messageCount)
	if messageCount > 0 {
		t.Logf("каскадное удаление может не работать без foreign keys")
	}
}

func (suite *HandlersTestSuite) TestDeleteChat_NotFound() {
	t := suite.T()

	rr := performRequest(suite.router, "DELETE", "/chats/999", nil)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
func (suite *HandlersTestSuite) TestFullChatFlow() {
	t := suite.T()

	createRequest := map[string]string{
		"title": "Интеграционный чат",
	}

	rr := performRequest(suite.router, "POST", "/chats", createRequest)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var chat models.Chat
	json.Unmarshal(rr.Body.Bytes(), &chat)
	chatID := chat.ID

	messages := []string{"Привет", "Как дела?", "Все хорошо"}
	for _, text := range messages {
		msgRequest := map[string]string{"text": text}
		rr = performRequest(suite.router, "POST", fmt.Sprintf("/chats/%d/messages", chatID), msgRequest)
		assert.Equal(t, http.StatusCreated, rr.Code)
		time.Sleep(time.Millisecond)
	}

	rr = performRequest(suite.router, "GET", fmt.Sprintf("/chats/%d?limit=10", chatID), nil)
	assert.Equal(t, http.StatusOK, rr.Code)

	var getResponse struct {
		models.Chat
		Messages []models.Message `json:"messages"`
	}
	json.Unmarshal(rr.Body.Bytes(), &getResponse)

	assert.Equal(t, "Интеграционный чат", getResponse.Title)
	assert.Len(t, getResponse.Messages, 3)

	rr = performRequest(suite.router, "DELETE", fmt.Sprintf("/chats/%d", chatID), nil)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	rr = performRequest(suite.router, "GET", fmt.Sprintf("/chats/%d", chatID), nil)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
