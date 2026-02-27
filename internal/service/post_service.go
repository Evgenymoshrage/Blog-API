package service

import (
	"context"
	"errors"
	"final_project/internal/model"
	"final_project/internal/repository"
	"fmt"
)

var ( // бизнес ошибки
	ErrPostNotFound = errors.New("post not found")
)

type PostService struct {
	postRepo repository.PostRepository // работа с постами
	userRepo repository.UserRepository // проверять существование пользователя
}

func NewPostService(postRepo repository.PostRepository, userRepo repository.UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

// Create создает новый пост
func (s *PostService) Create(
	ctx context.Context,
	userID int,
	req *model.PostCreateRequest,
) (*model.Post, error) {

	// Валидация данных
	if err := validatePostCreateRequest(req); err != nil {
		return nil, err
	}

	// Создание модели поста
	post := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: userID,
	}

	// Сохранение через репозиторий
	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Возврат созданного поста
	return post, nil
}

// GetByID получает пост по ID
func (s *PostService) GetByID(ctx context.Context, id int) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	return post, nil
}

// GetAll получает список постов с пагинацией
func (s *PostService) GetAll(
	ctx context.Context,
	limit, offset int,
) ([]*model.Post, int, error) {

	// Нормализация пагинации
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Получение постов
	posts, err := s.postRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Получение общего количества
	total, err := s.postRepo.GetTotalCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// Update обновляет пост
func (s *PostService) Update(
	ctx context.Context,
	id int,
	userID int,
	req *model.PostUpdateRequest,
) (*model.Post, error) {

	// Получение существующего поста
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	// Проверка прав
	if !post.CanBeEditedBy(userID) {
		return nil, ErrForbidden
	}

	// Валидация данных
	if err := validatePostUpdateRequest(req); err != nil {
		return nil, err
	}

	// Обновление полей
	post.Title = req.Title
	post.Content = req.Content

	// Сохранение
	if err := s.postRepo.Update(ctx, post); err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	return post, nil
}

// Delete удаляет пост
func (s *PostService) Delete(ctx context.Context, id int, userID int) error {

	// Получение поста
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrPostNotFound
		}
		return err
	}

	// Проверка прав
	if !post.CanBeDeletedBy(userID) {
		return ErrForbidden
	}

	// Удаление
	if err := s.postRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrPostNotFound
		}
		return fmt.Errorf("failed to delete post: %w", err)
	}
	return nil
}

// GetByAuthor получает посты конкретного автора
func (s *PostService) GetByAuthor(
	ctx context.Context,
	authorID int,
	limit, offset int,
) ([]*model.Post, int, error) {

	// Нормализация пагинации
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Получение постов
	posts, err := s.postRepo.GetByAuthorID(ctx, authorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Получение общего количества
	total, err := s.postRepo.GetTotalCountByAuthorID(ctx, authorID)
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// validatePostCreateRequest проверяет данные для создания поста
func validatePostCreateRequest(req *model.PostCreateRequest) error {
	if req.Title == "" {
		return fmt.Errorf("%w: title cannot be empty", ErrValidation)
	}
	if len(req.Title) > 200 {
		return fmt.Errorf("%w: title too long", ErrValidation)
	}
	if req.Content == "" {
		return fmt.Errorf("%w: content cannot be empty", ErrValidation)
	}
	return nil
}

// validatePostUpdateRequest проверяет данные для обновления поста
func validatePostUpdateRequest(req *model.PostUpdateRequest) error {
	if req.Title == "" {
		return fmt.Errorf("%w: title cannot be empty", ErrValidation)
	}
	if len(req.Title) > 200 {
		return fmt.Errorf("%w: title too long", ErrValidation)
	}
	if req.Content == "" {
		return fmt.Errorf("%w: content cannot be empty", ErrValidation)
	}
	return nil
}
