package service

import (
	"context"
	"errors"
	"fmt"

	"final_project/internal/model"      // Comment, CommentCreateRequest, CommentUpdateRequest, методы CanBeEditedBy
	"final_project/internal/repository" // Интерфейсы доступа к БД
)

var ( // бизнес-ошибки
	ErrCommentNotFound = errors.New("comment not found")
	ErrPostNotExists   = errors.New("post does not exist")
)

// CommentService содержит бизнес-логику работы с комментариями
type CommentService struct {
	commentRepo repository.CommentRepository // Работает с комментариями в БД
	postRepo    repository.PostRepository    // проверка существования поста
	userRepo    repository.UserRepository    // пока непонятно
}

// NewCommentService создает новый CommentService
func NewCommentService( // Конструктор сервиса
	commentRepo repository.CommentRepository,
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

// Create создает новый комментарий к посту
func (s *CommentService) Create(
	ctx context.Context, // контекст запроса
	postID int, // контекст запроса
	userID int, // кто пишет комментарий
	req *model.CommentCreateRequest, // данные из HTTP-запроса
) (*model.Comment, error) {

	// Валидация входных данных
	if err := validateCommentCreateRequest(req); err != nil {
		return nil, err
	}

	// Проверяем, существует ли пост
	exists, err := s.postRepo.Exists(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to check post existence: %w", err)
	}
	if !exists {
		return nil, ErrPostNotExists
	}

	// Создаем модель комментария
	comment := &model.Comment{
		Content:  req.Content,
		PostID:   postID,
		AuthorID: userID,
	}

	// Сохраняем комментарий
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return comment, nil
}

// GetByID возвращает комментарий по ID
func (s *CommentService) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	return comment, nil
}

// GetByPost возвращает комментарии к посту с пагинацией
func (s *CommentService) GetByPost(
	ctx context.Context,
	postID int,
	limit,
	offset int,
) ([]*model.Comment, int, error) {

	// Валидация пагинации
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Проверяем существование поста
	exists, err := s.postRepo.Exists(ctx, postID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, ErrPostNotExists
	}

	// Получаем комментарии
	comments, err := s.commentRepo.GetByPostID(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Получаем общее количество
	total, err := s.commentRepo.GetCountByPostID(ctx, postID)
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// Update обновляет комментарий
func (s *CommentService) Update(
	ctx context.Context,
	id int,
	userID int,
	req *model.CommentUpdateRequest,
) (*model.Comment, error) {

	// Получаем существующий комментарий
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrCommentNotFound
	}

	// Проверка прав
	if !comment.CanBeEditedBy(userID) {
		return nil, ErrForbidden
	}

	// Валидация
	if err := validateCommentUpdateRequest(req); err != nil {
		return nil, err
	}

	// Обновляем данные
	comment.Content = req.Content

	// Сохраняем
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// Delete удаляет комментарий
func (s *CommentService) Delete(ctx context.Context, id int, userID int) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return ErrCommentNotFound
	}

	if !comment.CanBeDeletedBy(userID) {
		return ErrForbidden
	}

	if err := s.commentRepo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

// validateCommentCreateRequest проверяет данные создания комментария
func validateCommentCreateRequest(req *model.CommentCreateRequest) error {
	if req.Content == "" {
		return errors.New("content cannot be empty")
	}
	if len(req.Content) > 1000 {
		return errors.New("content too long")
	}
	return nil
}

// validateCommentUpdateRequest проверяет данные обновления комментария
func validateCommentUpdateRequest(req *model.CommentUpdateRequest) error {
	if req.Content == "" {
		return errors.New("content cannot be empty")
	}
	if len(req.Content) > 1000 {
		return errors.New("content too long")
	}
	return nil
}
