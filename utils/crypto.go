package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	_ "errors"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"io"
	"strings"
	"time"
)

// PGPEncrypt шифрует данные с использованием PGP
func PGPEncrypt(data string, publicKey string) (string, error) {
	// Декодируем публичный ключ
	block, err := armor.Decode(strings.NewReader(publicKey))
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %v", err)
	}

	// Парсим публичный ключ
	entity, err := openpgp.ReadEntity(packet.NewReader(block.Body))
	if err != nil {
		return "", fmt.Errorf("failed to read entity: %v", err)
	}

	// Создаем буфер для зашифрованных данных
	var encryptedBuf strings.Builder
	armoredWriter, err := armor.Encode(&encryptedBuf, "PGP MESSAGE", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create armored writer: %v", err)
	}

	// Шифруем данные
	plaintext, err := openpgp.Encrypt(armoredWriter, []*openpgp.Entity{entity}, nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypt writer: %v", err)
	}

	_, err = plaintext.Write([]byte(data))
	if err != nil {
		return "", fmt.Errorf("failed to write data: %v", err)
	}

	err = plaintext.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close plaintext writer: %v", err)
	}

	err = armoredWriter.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close armored writer: %v", err)
	}

	return encryptedBuf.String(), nil
}

// PGPDecrypt расшифровывает данные с использованием PGP
func PGPDecrypt(encryptedData string, privateKey string) (string, error) {
	// Декодируем приватный ключ
	block, err := armor.Decode(strings.NewReader(privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %v", err)
	}

	// Парсим приватный ключ
	entity, err := openpgp.ReadEntity(packet.NewReader(block.Body))
	if err != nil {
		return "", fmt.Errorf("failed to read entity: %v", err)
	}

	// Создаем KeyRing
	keyRing := openpgp.EntityList{entity}

	// Декодируем зашифрованные данные
	encryptedBlock, err := armor.Decode(strings.NewReader(encryptedData))
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %v", err)
	}

	// Расшифровываем данные
	md, err := openpgp.ReadMessage(encryptedBlock.Body, keyRing, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to read message: %v", err)
	}

	// Читаем расшифрованные данные
	decryptedData, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted data: %v", err)
	}

	return string(decryptedData), nil
}

// GenerateHMAC создает HMAC для данных
func GenerateHMAC(data string, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GenerateRandomKey генерирует случайный ключ заданной длины
func GenerateRandomKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %v", err)
	}
	return key, nil
}

// ValidateHMAC проверяет HMAC
func ValidateHMAC(data string, hmac string, key []byte) bool {
	expectedHMAC := GenerateHMAC(data, key)
	return hmac == expectedHMAC
}

// GenerateSecureToken генерирует безопасный токен
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HashPassword создает хеш пароля
func HashPassword(password string) (string, error) {
	// Генерируем соль
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %v", err)
	}

	// Создаем хеш
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	hash := h.Sum(nil)

	// Объединяем соль и хеш
	result := make([]byte, len(salt)+len(hash))
	copy(result, salt)
	copy(result[len(salt):], hash)

	return base64.StdEncoding.EncodeToString(result), nil
}

// VerifyPassword проверяет пароль
func VerifyPassword(password, hashedPassword string) bool {
	// Декодируем хеш
	decoded, err := base64.StdEncoding.DecodeString(hashedPassword)
	if err != nil {
		return false
	}

	// Извлекаем соль и хеш
	salt := decoded[:16]
	hash := decoded[16:]

	// Создаем хеш для проверки
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	checkHash := h.Sum(nil)

	// Сравниваем хеши
	return hmac.Equal(hash, checkHash)
}

// GenerateExpirationTime генерирует время истечения срока действия
func GenerateExpirationTime(duration time.Duration) time.Time {
	return time.Now().Add(duration)
}

// IsExpired проверяет, истек ли срок действия
func IsExpired(expirationTime time.Time) bool {
	return time.Now().After(expirationTime)
}
