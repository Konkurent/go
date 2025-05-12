package middleware

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"strconv"
	"time"
)

type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body = b
	return lrw.ResponseWriter.Write(b)
}

// LoggingMiddleware логирует информацию о запросе и ответе
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем обертку для ResponseWriter
		lrw := &LoggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Обрабатываем запрос
		next.ServeHTTP(lrw, r)

		// Логируем информацию
		duration := time.Since(start)
		log.Printf(
			"Method: %s, Path: %s, Status: %d, Duration: %v, Body: %s",
			r.Method,
			r.URL.Path,
			lrw.statusCode,
			duration,
			string(lrw.body),
		)
	})
}

// AuthMiddleware проверяет JWT токен и добавляет заголовок X-User-ID
func AuthMiddleware(jwtKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка
			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			// Убираем префикс "Bearer " если он есть
			if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
				tokenString = tokenString[7:]
			}

			// Парсим и проверяем токен
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return jwtKey, nil
			})

			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Проверяем claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// Получаем user_id из claims
				userID, ok := claims["user_id"].(float64)
				if !ok {
					http.Error(w, "Invalid user_id in token", http.StatusUnauthorized)
					return
				}

				// Добавляем заголовок X-User-ID
				r.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

				// Добавляем информацию о пользователе в контекст запроса
				ctx := r.Context()
				ctx = context.WithValue(ctx, "user_id", uint(userID))
				ctx = context.WithValue(ctx, "email", claims["email"].(string))
				r = r.WithContext(ctx)
			} else {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext получает информацию о пользователе из контекста
func GetUserFromContext(r *http.Request) (uint, string, error) {
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		return 0, "", fmt.Errorf("user_id not found in context")
	}

	email, ok := r.Context().Value("email").(string)
	if !ok {
		return 0, "", fmt.Errorf("email not found in context")
	}

	return userID, email, nil
}
