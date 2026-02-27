// вытаскивает JWT из Authorization: Bearer
// проверяет токен через JWTManager
// кладёт данные пользователя в context.Context
// даёт хендлерам доступ к userID / email / username
// поддерживает: обязательную и опциональную авторизацию

package middleware

import (
	"context"
	"encoding/json"
	"final_project/pkg/auth"
	"net/http"
	"strings"
)

// contextKey — отдельный тип, чтобы избежать коллизий
type contextKey string

const ( // ключи
	UserIDKey    contextKey = "userID"
	UserEmailKey contextKey = "userEmail"
	UserNameKey  contextKey = "username"
)

// AuthMiddleware provides JWT authentication
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

// RequireAuth — middleware с обязательной авторизацией
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r) // извлечение токена
		if token == "" {         // Если токена нет
			WriteJSONError(w, "missing authorization token", http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtManager.ValidateToken(token) // Проверка токена (подпись, срок действия, парсится claims)
		if err != nil {
			WriteJSONError(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Кладём данные в context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, UserNameKey, claims.Username)

		next(w, r.WithContext(ctx)) // Передаём дальше
	}
}

// OptionalAuth — необязательная авторизация (токен НЕ обязателен. если он есть и валиден — используем, если нет — идём дальше)
func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r) // Пытаемся достать токен
		if token == "" {         // если токена нет, работаем как гост
			next(w, r)
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil { // если токен плохой игнорируем
			next(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, UserNameKey, claims.Username)

		next(w, r.WithContext(ctx))
	}
}

// extractToken извлекает Bearer токен
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization") // Берём заголовок
	if authHeader == "" {                       // Нет заголовка — нет токена
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)  // Разбиваем
	if len(parts) != 2 || parts[0] != "Bearer" { // Проверяем формат
		return ""
	}

	return parts[1] // Возвращаем сам JWT
}

// Helpers для контекста

func GetUserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(UserIDKey).(int)
	return id, ok
}

func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UserNameKey).(string)
	return username, ok
}

// JSON error response
type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

// Chain объединяет middleware
func Chain(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
