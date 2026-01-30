package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"chat-api/internal/config"
)

var DB *gorm.DB

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()
	log.Printf("Подключение к базе с DSN: %s", dsn)

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("Провал при соединении к бд: %v", err)
	}

	for i := 0; i < 10; i++ {
		err = sqlDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Ожидание дб... (попытка %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("Не вышло подключитсья к дб: %v", err)
	}

	err = runGooseMigrations(sqlDB)
	if err != nil {
		return nil, fmt.Errorf("Не вышло совершить миграцию: %v", err)
	}

	DB, err = gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("Не вышло открыть бд: %v", err)
	}

	log.Println("База данных успешно подключена и  мигрирована")
	return DB, nil
}

func runGooseMigrations(db *sql.DB) error {
	migrationDir := "./migrations"

	// Создаем папку migrations если ее нет
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		log.Printf("Создание директории дял мигарции(если  нет): %s", migrationDir)
		err := os.MkdirAll(migrationDir, 0755)
		if err != nil {
			return fmt.Errorf("Ошибка создания директории для   миграции: %v", err)
		}
		err = createInitialMigration(migrationDir)
		if err != nil {
			return fmt.Errorf("Не получилось инициировать первую мигарцию: %v", err)
		}
	}

	//goose настройка
	goose.SetBaseFS(nil) // нил потому что нужен дефолт
	goose.SetTableName("goose_migrations")

	//получение версии миграции
	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		log.Printf("Получение версии миграции: %v", err)
	} else {
		log.Printf("Текущая версия миграции: %d", currentVersion)
	}

	//применяем миграции вверх
	err = goose.Up(db, migrationDir)
	if err != nil {
		return fmt.Errorf("Не вышло применить goose migrations: %v", err)
	}

	//статус после миграций
	err = goose.Status(db, migrationDir)
	if err != nil {
		log.Printf("Не выходит получить статус: %v", err)
	}
	log.Println("Статус миграции получен успешно")
	return nil
}

func createInitialMigration(migrationDir string) error {
	//сооздаем файл первой миграции
	migrationFile := filepath.Join(migrationDir, "001_init.sql")

	migrationContent := `-- +goose Up
Создание таблицы чатов
CREATE TABLE chats (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

Создание таблицы сообщений
CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    chat_id INTEGER NOT NULL,
    text VARCHAR(5000) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
);

Создание индексов для оптимизации
CREATE INDEX idx_messages_chat_id ON messages(chat_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

-- +goose Down
-- Удаление таблиц в обратном порядке
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_chat_id;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats;
`

	err := os.WriteFile(migrationFile, []byte(migrationContent), 0644)
	if err != nil {
		return fmt.Errorf("Не вышло создать миграционный файл: %v", err)
	}

	log.Printf("Создан файл первичной миграции: %s", migrationFile)
	return nil
}
