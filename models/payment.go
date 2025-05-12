package models

import (
	"gorm.io/gorm"
	"time"
)

// PaymentStatus представляет статус платежа
type PaymentStatus string

const (
	PaymentStatusPlanned  PaymentStatus = "PLANNED"  // Запланированный платеж
	PaymentStatusPaid     PaymentStatus = "PAID"     // Оплаченный платеж
	PaymentStatusOverdue  PaymentStatus = "OVERDUE"  // Просроченный платеж
	PaymentStatusCanceled PaymentStatus = "CANCELED" // Отмененный платеж
)

// Payment представляет платеж по кредиту
type Payment struct {
	gorm.Model
	CreditID    uint          `gorm:"not null"`
	Credit      Credit        `gorm:"foreignKey:CreditID"`
	PayDate     time.Time     `gorm:"not null"` // Планируемая дата платежа
	Amount      float64       `gorm:"not null"` // Сумма платежа
	InitAmount  float64       `gorm:"not null"` // Начальная сумма платежа
	IsOverdue   bool          `gorm:"not null;default:false"`
	Status      PaymentStatus `gorm:"type:varchar(20);not null;default:'PLANNED'"`
	RealPayDate *time.Time    // Дата реального платежа
}

// TableName возвращает имя таблицы для модели Payment
func (Payment) TableName() string {
	return "payments"
}
