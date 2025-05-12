package models

import (
	"time"
)

type BankAccount struct {
	ID          uint          `gorm:"primaryKey;autoIncrement"`
	Bank        string        `gorm:"column:bank;not null"`
	Number      string        `gorm:"column:number;unique;not null"`
	Title       string        `gorm:"column:title;not null"`
	Balance     float64       `gorm:"column:balance;type:decimal(20,2);not null;default:0.0"`
	HolderID    uint          `gorm:"column:holder_id;not null"`
	Holder      User          `gorm:"foreignKey:HolderID;references:ID"`
	Transaction []Transaction `gorm:"foreignKey:AccountID"`
	Cards       []Card        `gorm:"foreignKey:AccountID"`
	CreatedAt   time.Time     `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time     `gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
}

func (BankAccount) TableName() string {
	return "bank_accounts"
}
