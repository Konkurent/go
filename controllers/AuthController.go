package controllers

import (
	"awesomeProject/config"
	"awesomeProject/database"
	"awesomeProject/services"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthController struct {
	userHandler *services.UserService
	validate    *validator.Validate
	config      *config.Config
}

type SignInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type SignInResponse struct {
	Token string `json:"token"`
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type SignUpRequest struct {
	FirstName string `json:"firstName" validate:"required,min=2,max=50,alpha"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50,alpha"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,password"`
}

type Token struct {
	Token  string `json:"token"`
	Email  string `json:"email"`
	UserID uint   `json:"userId"`
}

type AuthResponse struct {
	Token Token `json:"token"`
	User  struct {
		ID        uint   `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	} `json:"user"`
}

func NewAuthController(db *database.Database) *AuthController {
	validate := validator.New()

	// Регистрация кастомной валидации для пароля
	validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		// Проверка на наличие хотя бы одной цифры
		hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
		// Проверка на наличие хотя бы одной заглавной буквы
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
		// Проверка на наличие хотя бы одной строчной буквы
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
		// Проверка на наличие хотя бы одного специального символа
		hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)

		return hasNumber && hasUpper && hasLower && hasSpecial
	})

	// Получаем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	return &AuthController{
		userHandler: services.NewUserService(db),
		validate:    validate,
		config:      cfg,
	}
}

// SignIn обрабатывает вход пользователя
func (c *AuthController) SignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация запроса
	if err := c.validate.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		http.Error(w, validationErrors.Error(), http.StatusBadRequest)
		return
	}

	// Ищем пользователя по email
	user, err := c.userHandler.FindByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Создаем JWT токен
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(c.config.JWT.SecretKey))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := SignInResponse{
		Token: tokenString,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *AuthController) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация запроса
	if err := c.validate.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		http.Error(w, validationErrors.Error(), http.StatusBadRequest)
		return
	}

	// Конвертируем SignUpRequest в CreateUserRequest
	createUserReq := services.CreateUserRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	}

	// Создаем пользователя через UserService
	user, err := c.userHandler.CreateUserInternal(createUserReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Генерация JWT токена
	token, err := c.generateToken(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := AuthResponse{
		Token: *token,
		User: struct {
			ID        uint   `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Email     string `json:"email"`
		}{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetJWTKey возвращает ключ для JWT
func (c *AuthController) GetJWTKey() string {
	return c.config.JWT.SecretKey
}

// GetJWTExpiresIn возвращает время жизни JWT токена
func (c *AuthController) GetJWTExpiresIn() int {
	return c.config.JWT.ExpiresIn
}

// generateToken создает JWT токен
func (c *AuthController) generateToken(userID uint, email string) (*Token, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(c.config.JWT.SecretKey))
	if err != nil {
		return nil, err
	}

	return &Token{
		Token:  tokenString,
		Email:  email,
		UserID: userID,
	}, nil
}
