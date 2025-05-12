package utils

import (
	"sync"
	"time"
)

// RateLimiter реализует ограничение частоты запросов
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter создает новый RateLimiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow проверяет, разрешен ли запрос
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Очищаем старые запросы
	if requests, exists := rl.requests[key]; exists {
		var validRequests []time.Time
		for _, t := range requests {
			if t.After(windowStart) {
				validRequests = append(validRequests, t)
			}
		}
		rl.requests[key] = validRequests
	}

	// Проверяем лимит
	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	// Добавляем новый запрос
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// Reset сбрасывает счетчик для ключа
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.requests, key)
}

// GetRemaining возвращает количество оставшихся запросов
func (rl *RateLimiter) GetRemaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	var validRequests []time.Time
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	return rl.limit - len(validRequests)
}

// GetResetTime возвращает время до сброса лимита
func (rl *RateLimiter) GetResetTime(key string) time.Time {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.requests[key]) == 0 {
		return time.Now()
	}

	oldestRequest := rl.requests[key][0]
	return oldestRequest.Add(rl.window)
}
