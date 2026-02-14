// логирование запросов
// recovery после паники
// CORS
// request ID
// rate limit
// установку Content-Type
// утилиты (IP клиента, захват статус-кода)

package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid" // генерация уникального ID запроса
)

// LoggingMiddleware provides request logging, CORS, recovery and other utility middleware
type LoggingMiddleware struct {
	logger *log.Logger
}

// NewLoggingMiddleware принимает готовый логгер, создаёт структуру LoggingMiddleware, кладёт логгер внутрь, возвращает указатель
func NewLoggingMiddleware(logger *log.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Logger логирование HTTP-запросов
func (m *LoggingMiddleware) Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { // обёртка, которая выполнится вместо оригинального хендлера
		start := time.Now()

		rw := newResponseWriter(w)

		next(rw, r)

		duration := time.Since(start) // Считаем, сколько времени занял запрос

		m.logger.Printf( // Пишем лог
			"%s %s | IP=%s | Status=%d | Duration=%s",
			r.Method,
			r.URL.Path,
			getClientIP(r),
			rw.statusCode,
			duration,
		)
	}
}

// Recovery восстанавливается после паник
func (m *LoggingMiddleware) Recovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() { // гарантируем выполнение после выхода из функции, даже при panic
			if err := recover(); err != nil { // recover() ловит panic
				m.logger.Printf( // Логируем факт panic
					"PANIC recovered: %v | %s %s | IP=%s",
					err,
					r.Method,
					r.URL.Path,
					getClientIP(r),
				)

				http.Error( // Отправляем клиенту ответ
					w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
			}
		}()

		next(w, r) // Если panic не было, просто выполняем handler
	}
}

// CORS (Cross-Origin Resource Sharing) механизм безопасности браузера, который позволяет веб-странице с одного источника
// (домена, протокола, порта) безопасно запрашивать ресурсы с другого источника.
func (m *LoggingMiddleware) CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                                       // Разрешаем запросы с любых доменов
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS") // Разрешённые HTTP методы
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")            // Какие заголовки клиент может отправлять
		w.Header().Set("Access-Control-Max-Age", "86400")                                        // Браузер может кэшировать preflight 24 часа

		if r.Method == http.MethodOptions { // preflight запрос от браузера
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r) // Обычные запросы идут дальше
	}
}

// уникальный тип ключа для context
type requestIDKeyType struct{}

var requestIDKey = requestIDKeyType{}

// RequestID добавляет уникальный ID к каждому запросу
func (m *LoggingMiddleware) RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String() // Генерируется уникальный UUID

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)

		w.Header().Set("X-Request-ID", requestID)

		m.logger.Printf(
			"Request started | ID=%s | %s %s",
			requestID,
			r.Method,
			r.URL.Path,
		)

		next(w, r.WithContext(ctx)) // Передаём request дальше с новым context
	}
}

// RateLimiter ограничивает количество запросов от одного клиента
func (m *LoggingMiddleware) RateLimiter(maxRequests int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	type client struct { // Структура для хранения количества запросов и времени окончания окна
		count     int
		expiresAt time.Time
	}

	clients := make(map[string]*client) // Хранилище клиентов по IP

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			now := time.Now()

			c, exists := clients[ip]               // Проверяем, был ли этот IP раньше
			if !exists || now.After(c.expiresAt) { // если клиента нет или окно истекло, создаем нвое окно
				clients[ip] = &client{
					count:     1,
					expiresAt: now.Add(window),
				}
				next(w, r)
				return
			}

			if c.count >= maxRequests { // Если превышен лимит
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			c.count++
			next(w, r)
		}
	}
}

// ContentTypeJSON устанавливает Content-Type: application/json для всех ответов
func (m *LoggingMiddleware) ContentTypeJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}

// getClientIP извлекает IP адрес клиента
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	return strings.Split(r.RemoteAddr, ":")[0]
}

// responseWriter обертка для захвата статус кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader сохраняет статус код
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

// Write вызывает WriteHeader если еще не был вызван
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// newResponseWriter создает новую обертку
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}
}
