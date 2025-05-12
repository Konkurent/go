package models

import (
	"gorm.io/gorm"
	"time"
)

// Credit представляет кредит
type Credit struct {
	gorm.Model
	Rate      float64      `gorm:"not null"`
	Account   BankAccount  `gorm:"foreignKey:AccountID"`
	AccountID uint         `gorm:"not null"`
	Amount    float64      `gorm:"not null"`
	Status    CreditStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	Payments  []Payment    `gorm:"foreignKey:PaymentID"`
	StartDate time.Time    `gorm:"not null"`
	EndDate   time.Time    `gorm:"not null"`
}

// CreditStatus представляет статус кредита
type CreditStatus string

const (
	CreditStatusActive   CreditStatus = "ACTIVE"
	CreditStatusPaid     CreditStatus = "PAID"
	CreditStatusOverdue  CreditStatus = "OVERDUE"
	CreditStatusCanceled CreditStatus = "CANCELED"
)

// TableName возвращает имя таблицы для модели Credit
func (Credit) TableName() string {
	return "credits"
}
