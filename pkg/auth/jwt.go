package auth

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ( // Ошибки
	ErrInvalidToken = errors.New("invalid token") // токен неправильный, повреждён или подпись неверная
	ErrExpiredToken = errors.New("token expired") // токен просрочен
)

// Claims представляет данные, хранимые в JWT токене
type Claims struct { // «паспортные данные токена»
	UserID               int    `json:"user_id"` // ID пользователя
	Email                string `json:"email"`   // для идентификации пользователя
	Username             string `json:"username"`
	jwt.RegisteredClaims        // стандартные поля JWT, обязательны для безопасности: ExpiresAt, IssuedAt, NotBefore
}

// JWTManager управляет созданием и валидацией JWT токенов
type JWTManager struct {
	secretKey []byte        // секрет для подписи токена (HS256)
	ttl       time.Duration // время жизни токена
}

// NewJWTManager создает новый экземпляр JWT менеджера
func NewJWTManager(secretKey string, ttlHours int) *JWTManager {
	if secretKey == "" {
		log.Fatal("JWT secret must be set via environment variable JWT_SECRET") // Проверка, что секрет задан
	}
	return &JWTManager{
		secretKey: []byte(secretKey),                   // преобразует секрет из строки в []byte
		ttl:       time.Duration(ttlHours) * time.Hour, // преобразует TTL из часов в time.Duration для работы с time.Now().Add
	}
}

// GenerateToken создает новый JWT токен для пользователя
func (m *JWTManager) GenerateToken(userID int, email, username string) (string, time.Time, error) {
	expiration := time.Now().Add(m.ttl) // Считаем время окончания действия токена

	claims := &Claims{ // Создаём Claims, которые будут храниться в токене
		UserID:   userID,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{ // безопасность
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Создаём токен с алгоритмом HS256
	signedToken, err := token.SignedString(m.secretKey)        // Подписываем токен секретным ключом
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiration, nil // Возвращаем строку токена и время истечения
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	// Парсим токен и проверяем подпись
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil { // Если токен невалидный или просрочен возвращаем нужную ошибку
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims) // Проверяем, что токен валидный и claims корректные
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken обновляет существующий токен
func (m *JWTManager) RefreshToken(tokenString string) (string, time.Time, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil { // Проверяем старый токен
		return "", time.Time{}, err
	}

	// Генерируем новый токен с теми же данными
	return m.GenerateToken(claims.UserID, claims.Email, claims.Username)
}

// GetUserIDFromToken быстро извлекает ID пользователя из токена без полной валидации
// Для логирования, быстрого определения пользователя в middleware
func (m *JWTManager) GetUserIDFromToken(tokenString string) (int, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return 0, ErrInvalidToken
	}

	return claims.UserID, nil
}
