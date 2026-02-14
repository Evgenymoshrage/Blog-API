package database

import (
	"database/sql" // Для работы с БД
	"fmt"

	_ "github.com/lib/pq" // драйвер PostgreSQL. Подчёркивание - не используем функции пакета напрямую, регистрация в database/sql
)

// Конфигурация БД PostgreSQL
type Config struct {
	Host     string // адрес сервера БД
	Port     int    // порт PostgreSQL
	User     string // пользователь
	Password string // пароль
	DBName   string // имя базы
	SSLMode  string // SSL режим (безопасность)
}

// NewPostgresDB создает новое подключение к PostgreSQL
func NewPostgresDB(cfg Config) (*sql.DB, error) {
	dsn := GetDSN(cfg) //  строка подключения к БД

	// Создали подключение (не устанавливет соединение - проверяет драйвер, сохраняет параметры, готовит пул соединений)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Проверили подключение (реальное подключение к БД)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Настроили подключение (пул соединений)
	db.SetMaxOpenConns(25) // Максимум 25 одновременных соединений к БД
	db.SetMaxIdleConns(10) // 10 соединений держатся на готове

	return db, nil // Возвращаем подключение
}

// Migrate выполняет миграции базы данных (создают таблицы, индексы, схему БД)
// Пока все таблицы не создадутся, в БД ничего записывается (существует временно, внутри транзакции)
func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,

		`CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			author_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,

		`CREATE TABLE IF NOT EXISTS comments (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			post_id INT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			author_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,

		`CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);`,
	}

	tx, err := db.Begin() // Начало транзакции (Все следующие команды — временные)
	if err != nil {
		return err
	}

	for _, query := range queries { // Выполнение SQL внутри транзакции
		if _, err := tx.Exec(query); err != nil {
			_ = tx.Rollback() // Если ошибка, все отменяется и удаляется
			return err
		}
	}

	return tx.Commit() // Если ошибок не было, все сохраняется в БД
}

// CheckConnection проверяет соединение с базой данных
func CheckConnection(db *sql.DB) error {
	return db.Ping() // Запрос к базе данных: база запущена? доступ по сети? логин / пароль? база сейчас отвечает?
}

// GetDSN (Data Source Name) формирует строку подключения к PostgreSQL (как именно подключаться к базе данных)
func GetDSN(cfg Config) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", // Шаблон строки (формат PostgreSQL)
		cfg.Host,     // адрес сервера БД
		cfg.Port,     // порт PostgreSQL
		cfg.User,     // пользователь
		cfg.Password, // пароль
		cfg.DBName,   // имя базы
		cfg.SSLMode,  // SSL режим
	)
}

// Close закрывает соединение с базой данных
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// TestConnection выполняет тестовый запрос к БД
func TestConnection(db *sql.DB) error {
	var result int
	return db.QueryRow("SELECT 1").Scan(&result)
}
