// Репозиторий = SQL + mapping
// context.Context
// QueryRow — 1 запись
// Query — много записей
// RowsAffected — если записи не было, удалление SQL не считает ошибкой

package repository

import (
	"context"      // Контекст для управления временем выполнения и отменой запросов
	"database/sql" // Работа с SQL БД через стандартную библиотеку Go
	"errors"
	"final_project/internal/model"
	"fmt"
	"time"
)

var ( // Ошибка, если пост не найден
	ErrPostNotFound = errors.New("post not found")
)

// PostRepo представляет репозиторий для работы с постами
type PostRepo struct {
	db *sql.DB // пул соединений к БД
}

// NewPostRepo создает новый репозиторий постов
func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

// Create создает новый пост
func (r *PostRepo) Create(ctx context.Context, post *model.Post) error {
	// Устанавливаем дату создания и обновления поста.
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	// SQL-запрос для вставки нового поста
	query := `
		INSERT INTO posts (title, content, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	// QueryRowContext для одной строки с RETURNING id
	if err := r.db.QueryRowContext(ctx, query,
		post.Title, post.Content, post.AuthorID, post.CreatedAt, post.UpdatedAt,
	).Scan(&post.ID); err != nil { // считываем сгенерированный ID в структуру post
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// GetByID получает пост по ID
func (r *PostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	// SQL-запрос на получение поста по ID
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		WHERE id = $1
	`

	var post model.Post
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID, &post.Title, &post.Content,
		&post.AuthorID, &post.CreatedAt, &post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post by ID: %w", err)
	}

	return &post, nil
}

// GetAll получает все посты с пагинацией (постранично)
func (r *PostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	// Получаем несколько постов
	// по убыванию даты создания (ORDER BY created_at DESC)
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}
	defer rows.Close() // Откладываем закрытие

	var posts []*model.Post
	for rows.Next() {
		var post model.Post // Считываем данные в структуру post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content,
			&post.AuthorID, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post) // Добавляем в срез posts
	}

	if err := rows.Err(); err != nil { // Проверяем ошибки итерации
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return posts, nil // Возвращаем срез постов
}

// GetTotalCount возвращает общее количество постов
func (r *PostRepo) GetTotalCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM posts`

	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}

// Update обновляет пост
func (r *PostRepo) Update(ctx context.Context, post *model.Post) error {
	post.UpdatedAt = time.Now() // При обновлении меняем поле UpdatedAt

	// Обновляем только нужные поля
	query := `
		UPDATE posts
		SET title = $1, content = $2, updated_at = $3
		WHERE id = $4
	`

	// SQL-запрос без возврата строк
	result, err := r.db.ExecContext(ctx, query, post.Title, post.Content, post.UpdatedAt, post.ID)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	rows, err := result.RowsAffected() // возвращает количество строк, затронутых SQL-запросом
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrPostNotFound
	}

	return nil
}

// Delete удаляет пост по ID
func (r *PostRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM posts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rows, err := result.RowsAffected() // возвращает количество строк, затронутых SQL-запросом
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrPostNotFound
	}

	return nil
}

// Exists проверяет существование поста
func (r *PostRepo) Exists(ctx context.Context, id int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1)`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check post existence: %w", err)
	}

	return exists, nil
}

// GetByAuthorID получает посты определенного автора
// Логика такая же, как GetAll, но добавлено WHERE author_id = $1
func (r *PostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		WHERE author_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, authorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts by author: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return posts, nil
}

func (r *PostRepo) GetTotalCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	query := `SELECT COUNT(*) FROM posts WHERE author_id = $1`
	var count int
	if err := r.db.QueryRowContext(ctx, query, authorID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count posts by author: %w", err)
	}
	return count, nil
}
