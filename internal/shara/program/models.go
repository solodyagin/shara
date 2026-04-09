package program

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Record struct {
	gorm.Model
	Link     string `gorm:"unique"`
	Name     string
	OrigName string
	Size     int64
}

func (r *Record) BeforeCreate(tx *gorm.DB) error {
	s, err := generateRandomString(44)
	if err != nil {
		return errors.New("can't save invalid data")
	}
	r.Link = s
	return nil
}

func generateRandomString(length int) (string, error) {
	// Проверка минимальной длины
	if length < 16 { // UUID занимает 16 байт, которые кодируются в 22 символа base64
		return "", fmt.Errorf("length must be at least 16")
	}

	u := uuid.New()

	// Вычисляем сколько дополнительных байт нужно
	extraBytes := max(length-16, 0)

	b := make([]byte, extraBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	combined := append(u[:], b...)

	s := base64.RawURLEncoding.EncodeToString(combined)

	// Удаляем нежелательные символы
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")

	// Обрезаем до нужной длины
	if len(s) > length {
		s = s[:length]
	}

	return s, nil
}
