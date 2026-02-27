// Методы CRUD:
// Create — создать нового пользователя
// Get — получить пользователя по ID/email/username
// Exists — быстро проверить существование
// Update — изменить данные
// Delete — удалить
// Контекст для отмены/таймаута
// Репозиторий = SQL + mapping
// QueryRow — 1 запись
// Query — много записей
// RowsAffected — если записи не было, удалление SQL не считает ошибкой

package repository

import (
	"context"      // Контекст для управления временем выполнения и отменой запросов
	"database/sql" // Работа с SQL БД через стандартную библиотеку Go
	"errors"
	"final_project/internal/model" // Модель данных пользователя
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	ErrUserNotFound = errors.New("user not found")      // Пользователь не найден
	ErrUserExists   = errors.New("user already exists") // Пользователь существует
)

// UserRepo представляет репозиторий для работы с пользователями
type UserRepo struct {
	db *sql.DB // SQL соединение
}

// NewUserRepo создает новый репозиторий пользователей
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create создает нового пользователя
func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	now := time.Now()    // Текущее время
	user.CreatedAt = now // Сохраняем дату создания
	user.UpdatedAt = now // Сохраняем дату обновления

	// Выполняем запрос и получаем ID нового пользователя
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	// QueryRowContext выполняет запрос и возвращает одну строку
	// Мы используем RETURNING id чтобы сразу получить ID нового пользователя
	err := r.db.QueryRowContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)
	if err != nil {
		// Проверка на уникальность
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return ErrUserExists
			}
		}
		return fmt.Errorf("failed to create user: %w", err) // Любая другая ошибка
	}

	return nil
}

// GetByID получает пользователя по ID
func (r *UserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user model.User
	// Выполняем запрос, сканируем результат в структуру
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil { // Если пользователь не найден, возвращаем ErrUserNotFound
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// GetByEmail получает пользователя по email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetByUsername получает пользователя по username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// ExistsByEmail проверяет существование пользователя по email
func (r *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)` // возвращает true/false без загрузки всей строки
	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence by email: %w", err)
	}
	return exists, nil
}

// ExistsByUsername проверяет существование пользователя по username
func (r *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence by username: %w", err)
	}
	return exists, nil
}

// Update обновляет данные пользователя
func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now() // Время обновления
	query := `
		UPDATE users
		SET username = $1, email = $2, updated_at = $3
		WHERE id = $4
	`
	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected() // Проверяем, что была изменена строка
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete удаляет пользователя
func (r *UserRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected() // Проверяем, что была удалена строка
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
