package auth

import (
	"crypto/rand"
	"errors"
	"math/big"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var ( // Ошибки
	ErrEmptyPassword    = errors.New("password cannot be empty") // Пустой
	ErrPasswordTooShort = errors.New("password is too short")    // Короткий

)

// HashPassword хеширует пароль используя bcrypt
func HashPassword(password string) (string, error) {
	// Проверка, что пароль не пустой
	if password == "" {
		return "", ErrEmptyPassword
	}

	// Генерация хеша с помощью bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Возвращаем хеш как строку
	return string(hash), nil
}

// CheckPassword проверяет соответствие пароля и его хеша
// При логине мы не расшифровываем хеш, а проверяем соответствие
func CheckPassword(password, hash string) bool {
	// bcrypt сам проверяет соль и cost
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength проверяет надежность пароля
func ValidatePasswordStrength(password string) error {
	if len(password) == 0 { // Проверяет, что пароль не пустой
		return ErrEmptyPassword
	}
	if len(password) < 6 { // Проверяет минимальную длину
		return ErrPasswordTooShort
	}

	// Проверяет, есть ли хотя бы одна буква и одна цифра
	var hasLetter, hasNumber bool
	for _, c := range password {
		if unicode.IsLetter(c) {
			hasLetter = true
		} else if unicode.IsNumber(c) {
			hasNumber = true
		}
	}

	if !hasLetter || !hasNumber {
		return errors.New("password must contain letters and numbers")
	}

	return nil
}

// GenerateRandomPassword генерирует случайный пароль
func GenerateRandomPassword(length int) (string, error) {
	// Определяем набор допустимых символов: буквы, цифры, спецсимволы
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	if length <= 0 {
		return "", errors.New("length must be positive")
	}

	password := make([]byte, length) // Создаём массив
	for i := range password {        // Для каждого символа выбираем случайное число через crypto/rand
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()] // Формируем строку
	}

	return string(password), nil // Возвращаем строку
}
