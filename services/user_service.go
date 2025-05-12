package services

import (
	"awesomeProject/database"
	"awesomeProject/models"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *database.Database
}

type UserDTO struct {
	ID        uint   `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type CreateUserRequest struct {
	FirstName string `json:"firstName" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
}

type UserResponse struct {
	ID        uint   `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

func NewUserService(db *database.Database) *UserService {
	return &UserService{db: db}
}

// CreateUserInternal создает нового пользователя
func (h *UserService) CreateUserInternal(req CreateUserRequest) (*models.User, error) {
	// Проверяем, существует ли пользователь с таким email
	var existingUser models.User
	if err := h.db.DB.Where("LOWER(email) = LOWER(?)", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user with this email already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Создаем нового пользователя
	user := &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  string(hashedPassword),
	}

	if err := h.db.DB.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// findById ищет пользователя по ID
func (h *UserService) findById(id uint) (*models.User, error) {
	var user models.User
	if err := h.db.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// getById ищет пользователя по ID, возвращает nil если не найден
func (h *UserService) getById(id uint) (*models.User, error) {
	var user models.User
	if err := h.db.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail ищет пользователя по email (игнорируя регистр и пробелы)
func (h *UserService) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := h.db.DB.Where("LOWER(TRIM(email)) = LOWER(TRIM(?))", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}
