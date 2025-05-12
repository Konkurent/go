package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	FirstName string    `gorm:"column:first_name;not null;size:50"`
	LastName  string    `gorm:"column:last_name;not null;size:50"`
	Email     string    `gorm:"column:email;unique;not null;size:100;index"`
	Password  string    `gorm:"column:password;not null;size:100"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
}

func (User) TableName() string {
	return "users"
}

// BeforeCreate хук для валидации перед созданием
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if len(u.FirstName) < 2 || len(u.FirstName) > 50 {
		return errors.New("first name must be between 2 and 50 characters")
	}
	if len(u.LastName) < 2 || len(u.LastName) > 50 {
		return errors.New("last name must be between 2 and 50 characters")
	}
	if len(u.Email) < 3 || len(u.Email) > 100 {
		return errors.New("email must be between 3 and 100 characters")
	}
	return nil
}
