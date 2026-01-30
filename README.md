# Chat API

Простой REST API для управления чатами и сообщениями.

## Быстрый старт

В корне проекта:

API будет доступно по адресу: `http://localhost:8080`

## Требования
- Docker
- Docker Compose

## Технологии
- Go 1.23+ - основной язык
- PostgreSQL 15 - база данных
- GORM - ORM для работы с БД
- Gorilla Mux - HTTP роутер
- Goose - миграции БД
- Testify - тестирование
- Docker & Docker Compose – контейнеризация

## Проверялся в POSTman

### GET /health
Проверка на жизнь сайта

### POST /chats
Создать чат
```json
{
  "title": "Название чата"
}
POST /chats/{id}/messages
Отправить новое сообщение
{
  "text": "Текст сообщения"
}
GET /chats/{id}
Перейти к чату по айди

DELETE /chats/{id}
Удалить чат
