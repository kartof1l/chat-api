Простой REST API для управления чатами и сообщениями. 
В корне проекта: docker-compose up –build
API будет доступно по адресу: http://localhost:8080
Требования:
Docker
Docker Compose
 Технологии
Go 1.23+ - основной язык
PostgreSQL 15 - база данных
GORM - ORM для работы с БД
Gorilla Mux - HTTP роутер
Goose - миграции БД
Testify - тестирование
Docker & Docker Compose – контейнеризация
Проверялся в POSTman
GET /health – првоерка на жизнь сайта
POST /chats – создать чат 
{
  "title": "Название чата"
}
POST /chats/{id}/messages – отправить нвоое сообщение
{
  "text": "Текст сообщения"
}
GET /chats/{id} – перейти к чату по айди
DELETE /chats/{id} – удалить чат
