package models

import (
	"gorm.io/gorm"
)

// Card представляет банковскую карту
type Card struct {
	gorm.Model
	NumberEncrypted     string      `gorm:"not null"`
	NumberHMAC          string      `gorm:"not null"`
	ExpirationEncrypted string      `gorm:"not null"`
	ExpirationHMAC      string      `gorm:"not null"`
	CVV                 string      `gorm:"not null"`
	AccountID           uint        `gorm:"not null"`
	Account             BankAccount `gorm:"foreignKey:AccountID"`
}

// TableName возвращает имя таблицы для модели Card
func (Card) TableName() string {
	return "cards"
}
