// Register — регистрация пользователя + JWT.
// Login — вход пользователя + JWT.
// GetByID / GetByEmail — получение пользователя из базы.

package service

import (
	"context"
	"errors"
	"final_project/internal/model"
	"final_project/internal/repository"
	"final_project/pkg/auth"
	"fmt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

type UserService struct {
	userRepo   repository.UserRepository // репозиторий пользователей
	jwtManager *auth.JWTManager          // менеджер JWT токенов для аутентификации
}

func NewUserService(userRepo repository.UserRepository, jwtManager *auth.JWTManager) *UserService { // Конструктор сервиса
	return &UserService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (s *UserService) Register( // регистрация пользователя
	ctx context.Context,
	req *model.UserCreateRequest,
) (*model.TokenResponse, error) { // Возвращает TokenResponse с JWT токеном после регистрации

	// Валидация входных данных
	if err := validateUserCreateRequest(req); err != nil {
		return nil, err
	}

	// Проверка уникальности email
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email existence failed: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Проверка уникальности username
	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("check username existence failed: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Хеширование пароля
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("password hashing failed: %w", err)
	}

	// Создание модели пользователя
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
	}

	// Сохранение пользователя
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user failed: %w", err)
	}

	// Генерация JWT токена
	token, expiresAt, err := s.jwtManager.GenerateToken(
		user.ID,
		user.Email,
		user.Username,
	)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	// Формирование ответа
	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

func (s *UserService) Login( // вход пользователя
	ctx context.Context,
	req *model.UserLoginRequest,
) (*model.TokenResponse, error) {

	// Валидация входных данных
	if err := validateUserLoginRequest(req); err != nil {
		return nil, err
	}

	// Поиск пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email failed: %w", err)
	}

	// Проверка пароля
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Генерация JWT токена при успешной аутентификации
	token, expiresAt, err := s.jwtManager.GenerateToken(
		user.ID,
		user.Email,
		user.Username,
	)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	// Ответ
	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) { // Получение пользователя по ID
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id failed: %w", err)
	}
	return user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) { // Получение пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email failed: %w", err)
	}
	return user, nil
}

func validateUserCreateRequest(req *model.UserCreateRequest) error { // Проверка, что поля для регистрации корректны
	if req.Username == "" || len(req.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" || len(req.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}

func validateUserLoginRequest(req *model.UserLoginRequest) error { // Проверка, что email и пароль при входе заполнены
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	return nil
}
