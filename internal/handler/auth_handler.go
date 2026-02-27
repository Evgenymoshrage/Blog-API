package handler

import (
	"encoding/json"
	"errors"
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
	var req model.UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			writeError(w, "User already exists", http.StatusConflict)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, resp, http.StatusCreated)
}

// Login обрабатывает запрос на вход пользователя
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, "Invalid credentials", http.StatusUnauthorized)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, resp, http.StatusOK)
}

// GetProfile возвращает профиль текущего пользователя
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, "User not found", http.StatusNotFound)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, user.ToResponse(), http.StatusOK)
}

// writeError отправляет JSON ответ с ошибкой (единый формат ошибок)
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}
