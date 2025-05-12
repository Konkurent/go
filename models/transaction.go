package models

import (
	"time"
)

type Transaction struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	AccountID     uint      `gorm:"column:account_id;not null;index"`
	Amount        float64   `gorm:"column:amount;not null"`
	Type          string    `gorm:"column:type;not null;size:20"` // deposit, withdraw, transfer_in, transfer_out
	BalanceBefore float64   `gorm:"column:balance_before;not null"`
	BalanceAfter  float64   `gorm:"column:balance_after;not null"`
	Description   string    `gorm:"column:description;size:255"`
	CreatedAt     time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
}

func (Transaction) TableName() string {
	return "transactions"
}
