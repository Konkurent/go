package middleware

import (
	"net/http"
	"time"

	"awesomeProject/utils"
	"github.com/gin-gonic/gin"
)

var (
	// Глобальный rate limiter
	globalLimiter = utils.NewRateLimiter(100, time.Minute) // 100 запросов в минуту
)

// RateLimit middleware для ограничения частоты запросов
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем IP-адрес клиента
		clientIP := c.ClientIP()

		// Проверяем лимит
		if !globalLimiter.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
				"reset": globalLimiter.GetResetTime(clientIP),
			})
			c.Abort()
			return
		}

		// Добавляем заголовки с информацией о лимитах
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", string(globalLimiter.GetRemaining(clientIP)))
		c.Header("X-RateLimit-Reset", globalLimiter.GetResetTime(clientIP).String())

		c.Next()
	}
}

// Logger middleware для логирования запросов
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Начало запроса
		startTime := time.Now()

		// Обработка запроса
		c.Next()

		// Время выполнения
		duration := time.Since(startTime)

		// Логируем информацию о запросе
		utils.LogInfo("Request: %s %s - Status: %d - Duration: %v",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)

		// Логируем ошибки
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				utils.LogError("Error: %v", e)
			}
		}
	}
}

// Recovery middleware для обработки паник
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Логируем панику
				utils.LogError("Panic recovered: %v", err)

				// Отправляем ответ клиенту
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// Auth middleware для проверки аутентификации
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// TODO: Добавить проверку токена
		// Здесь должна быть ваша логика проверки токена

		c.Next()
	}
}

// CORSMiddleware middleware для CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
