package utils

import (
	"sync"
	"time"
)

// Metrics содержит метрики приложения
type Metrics struct {
	mu sync.RWMutex

	// Метрики запросов
	TotalRequests     int64
	FailedRequests    int64
	RequestLatency    time.Duration
	AverageLatency    time.Duration
	LastRequestTime   time.Time
	RequestsPerMinute float64

	// Метрики карт
	TotalCards        int64
	ActiveCards       int64
	BlockedCards      int64
	ExpiredCards      int64
	LastCardOperation time.Time

	// Метрики ошибок
	ErrorCount     int64
	LastErrorTime  time.Time
	ErrorTypes     map[string]int64
	CriticalErrors int64
}

var (
	metrics     *Metrics
	metricsOnce sync.Once
)

// GetMetrics возвращает экземпляр метрик
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = &Metrics{
			ErrorTypes: make(map[string]int64),
		}
	})
	return metrics
}

// RecordRequest записывает метрики запроса
func (m *Metrics) RecordRequest(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.RequestLatency += duration
	m.AverageLatency = m.RequestLatency / time.Duration(m.TotalRequests)
	m.LastRequestTime = time.Now()

	if err != nil {
		m.FailedRequests++
		m.RecordError(err)
	}

	// Обновляем количество запросов в минуту
	if m.LastRequestTime.Sub(m.LastRequestTime.Add(-time.Minute)) >= time.Minute {
		m.RequestsPerMinute = float64(m.TotalRequests) / time.Since(m.LastRequestTime).Minutes()
	}
}

// RecordCardOperation записывает метрики операции с картой
func (m *Metrics) RecordCardOperation(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LastCardOperation = time.Now()

	switch operation {
	case "create":
		m.TotalCards++
		m.ActiveCards++
	case "delete":
		m.TotalCards--
		m.ActiveCards--
	case "block":
		m.ActiveCards--
		m.BlockedCards++
	case "unblock":
		m.ActiveCards++
		m.BlockedCards--
	case "expire":
		m.ActiveCards--
		m.ExpiredCards++
	}

	if err != nil {
		m.RecordError(err)
	}
}

// RecordError записывает метрики ошибки
func (m *Metrics) RecordError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ErrorCount++
	m.LastErrorTime = time.Now()

	errorType := "unknown"
	if err != nil {
		errorType = err.Error()
	}

	m.ErrorTypes[errorType]++
}

// RecordCriticalError записывает метрики критической ошибки
func (m *Metrics) RecordCriticalError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CriticalErrors++
	m.RecordError(err)
}

// GetMetricsSnapshot возвращает снимок текущих метрик
func (m *Metrics) GetMetricsSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":      m.TotalRequests,
		"failed_requests":     m.FailedRequests,
		"average_latency":     m.AverageLatency,
		"requests_per_minute": m.RequestsPerMinute,
		"total_cards":         m.TotalCards,
		"active_cards":        m.ActiveCards,
		"blocked_cards":       m.BlockedCards,
		"expired_cards":       m.ExpiredCards,
		"error_count":         m.ErrorCount,
		"critical_errors":     m.CriticalErrors,
		"last_error_time":     m.LastErrorTime,
		"error_types":         m.ErrorTypes,
	}
}

// ResetMetrics сбрасывает все метрики
func (m *Metrics) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests = 0
	m.FailedRequests = 0
	m.RequestLatency = 0
	m.AverageLatency = 0
	m.RequestsPerMinute = 0
	m.TotalCards = 0
	m.ActiveCards = 0
	m.BlockedCards = 0
	m.ExpiredCards = 0
	m.ErrorCount = 0
	m.CriticalErrors = 0
	m.ErrorTypes = make(map[string]int64)
}
