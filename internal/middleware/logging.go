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
	"sync"
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
func (m *LoggingMiddleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		m.logger.Printf(
			"%s %s | IP=%s | Status=%d | Duration=%s",
			r.Method,
			r.URL.Path,
			getClientIP(r),
			rw.statusCode,
			time.Since(start),
		)
	})
}

func (m *LoggingMiddleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { // гарантируем выполнение после выхода из функции, даже при panic
			if err := recover(); err != nil {
				m.logger.Printf(
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

		next.ServeHTTP(w, r) // Если panic не было, просто выполняем handler
	})
}

// CORS (Cross-Origin Resource Sharing) механизм безопасности браузера, который позволяет веб-странице с одного источника
// (домена, протокола, порта) безопасно запрашивать ресурсы с другого источника.
func (m *LoggingMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                                       // Разрешаем запросы с любых доменов
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS") // Разрешённые HTTP методы
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")            // Какие заголовки клиент может отправлять
		w.Header().Set("Access-Control-Max-Age", "86400")                                        // Браузер может кэшировать preflight 24 часа

		if r.Method == http.MethodOptions { // preflight запрос от браузера
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r) // Обычные запросы идут дальше
	})
}

// уникальный тип ключа для context
type requestIDKeyType struct{}

var requestIDKey = requestIDKeyType{}

// RequestID добавляет уникальный ID к каждому запросу
func (m *LoggingMiddleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		m.logger.Printf(
			"Request started | ID=%s | %s %s",
			requestID,
			r.Method,
			r.URL.Path,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ContentTypeJSON устанавливает Content-Type: application/json для всех ответов
func (m *LoggingMiddleware) ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
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

type rateClient struct { // состояние одного клиента
	count     int
	expiresAt time.Time
}

type RateLimiter struct { // хранит состояние всех клиентов
	mu      sync.Mutex
	clients map[string]*rateClient
	max     int
	window  time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter { // создаёт один экземпляр rate limiter
	return &RateLimiter{
		clients: make(map[string]*rateClient),
		max:     max,
		window:  window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Определяем клиента
		ip := getClientIP(r)
		now := time.Now()

		// Блокируем доступ к map
		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Проверяем есть ли клиент и не истекло ли окно
		c, ok := rl.clients[ip]
		if !ok || now.After(c.expiresAt) {
			rl.clients[ip] = &rateClient{ // клиента нет -> начинаем новое окно
				count:     1,
				expiresAt: now.Add(rl.window),
			}
			next.ServeHTTP(w, r)
			return
		}

		if c.count >= rl.max {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		c.count++
		next.ServeHTTP(w, r)
	})
}
