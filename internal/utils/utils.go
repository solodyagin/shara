package utils

import (
	"math/rand"
	"strings"
	"time"
)

var runes = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomString генерирует случайную строку заданной длины
func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

// Contains проверяет: входит ли строка в массив строк?
func Contains(arr []string, str string, caseInsensitive bool) bool {
	lowerStr := strings.ToLower(str)
	for _, v := range arr {
		if (v == str) || (caseInsensitive && (strings.ToLower(v) == lowerStr)) {
			return true
		}
	}
	return false
}

// Min
func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
