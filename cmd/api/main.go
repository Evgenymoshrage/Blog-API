package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"final_project/internal/handler"
	"final_project/internal/logger"
	"final_project/internal/middleware"
	"final_project/internal/repository"
	"final_project/internal/service"
	"final_project/pkg/auth"
	"final_project/pkg/database"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Загружаем конфигурацию
	cfg := loadConfig()

	// Подключаемся к БД
	db, err := database.NewPostgresDB(database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Создаём JWT менеджер
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours)

	// Инициализируем репозитории
	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	commentRepo := repository.NewCommentRepo(db)

	// Инициализируем сервисы
	userService := service.NewUserService(userRepo, jwtManager)
	postService := service.NewPostService(postRepo, userRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)

	// Инициализируем новый логер
	eventLogger, err := logger.NewEventLogger("logs.txt")
	if err != nil {
		log.Fatalf("Failed to create event logger: %v", err)
	}
	defer eventLogger.Close()

	// Инициализируем хендлеры
	authHandler := handler.NewAuthHandler(userService)
	postHandler := handler.NewPostHandler(postService, eventLogger)
	commentHandler := handler.NewCommentHandler(commentService, eventLogger)

	// Настраиваем роутер
	router := chi.NewRouter()

	// Middleware
	logger := log.New(
		os.Stdout,     // куда писать
		"[BLOG-API] ", // префикс
		log.LstdFlags|log.Lshortfile,
	)

	lm := middleware.NewLoggingMiddleware(logger)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	router.Use(lm.Recovery)            // ловим panic
	router.Use(lm.RequestID)           // создаём request-id
	router.Use(lm.CORS)                // CORS + OPTIONS
	router.Use(lm.ContentTypeJSON)     // JSON заголовок
	router.Use(rateLimiter.Middleware) // rate limit
	router.Use(lm.Logger)              // логируем после выполнения

	// Публичные маршруты
	router.Post("/api/register", authHandler.Register)
	router.Post("/api/login", authHandler.Login)
	router.Get("/api/posts", postHandler.GetAll)
	router.Get("/api/posts/{id}", postHandler.GetByID)
	router.Get("/api/posts/{id}/comments", commentHandler.GetByPost)

	// Защищённые маршруты
	router.Group(func(r chi.Router) {
		r.Post("/api/posts", authMiddleware.RequireAuth(postHandler.Create))
		r.Put("/api/posts/{id}", authMiddleware.RequireAuth(postHandler.Update))
		r.Delete("/api/posts/{id}", authMiddleware.RequireAuth(postHandler.Delete))
		r.Post("/api/posts/{id}/comments", authMiddleware.RequireAuth(commentHandler.Create))
	})

	// Health check
	router.Get("/api/health", handler.Health)

	// Запуск сервера
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Starting server at %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// ----------------- Конфигурация -----------------
type Config struct {
	ServerHost      string
	ServerPort      int
	DBHost          string
	DBPort          int
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	JWTSecret       string
	JWTExpiryHours  int
	CacheTTLMinutes int
}

func loadConfig() *Config {
	return &Config{
		ServerHost:      getEnv("SERVER_HOST", "localhost"),
		ServerPort:      getEnvAsInt("SERVER_PORT", 8080),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnvAsInt("DB_PORT", 5432),
		DBUser:          getEnv("DB_USER", "bloguser"),
		DBPassword:      getEnv("DB_PASSWORD", "blogpassword"),
		DBName:          getEnv("DB_NAME", "blogdb"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		JWTExpiryHours:  getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		CacheTTLMinutes: getEnvAsInt("CACHE_TTL_MINUTES", 5),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}
