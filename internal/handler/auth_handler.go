package handler

import (
	"context"
	"encoding/json"
	"final_project/internal/middleware"
	"final_project/internal/model"
	"final_project/internal/service"
	"net/http"
)

// AuthHandler обрабатывает запросы аутентификации
type AuthHandler struct {
	userService *service.UserService
}

// ссылка на сервис пользователей (UserService), через который выполняется регистрация и логин
func NewAuthHandler(userService *service.UserService) *AuthHandler { // конструктор
	return &AuthHandler{
		userService: userService,
	}
}

// Register обрабатывает запрос на регистрацию нового пользователя
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { // Проверка HTTP метода
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем JSON тело запроса
	var req model.UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем сервис для регистрации
	resp, err := h.userService.Register(r.Context(), &req)

	// Обрабатываем ошибки сервиса
	if err != nil {
		switch err {
		case service.ErrUserAlreadyExists:
			writeError(w, err.Error(), http.StatusConflict)
		default:
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Отправка успешного ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

// Login обрабатывает запрос на вход пользователя
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { // Проверка метода
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем JSON
	var req model.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем сервис для логина
	resp, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			writeError(w, err.Error(), http.StatusUnauthorized)
		default:
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Отправка успешного ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// GetProfile возвращает профиль текущего пользователя
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context()) // Получаем userID из контекста
	if !ok {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем пользователя из базы по ID через сервис
	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем данные пользователя (без пароля) в JSON формате
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user.ToResponse())
}

// writeError отправляет JSON ответ с ошибкой (единый формат ошибок)
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}

// getUserIDFromContext извлекает ID пользователя из контекста
func getUserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(middleware.UserIDKey).(int)
	return id, ok
}
