package controllers

import (
	"awesomeProject/database"
	"awesomeProject/services"
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
)

// CreditController обрабатывает запросы, связанные с кредитами
type CreditController struct {
	creditService *services.CreditService
	validator     *validator.Validate
}

// NewCreditController создает новый экземпляр CreditController
func NewCreditController(db *database.Database, email *services.EmailService) *CreditController {
	return &CreditController{
		creditService: services.NewCreditService(db.DB, email),
		validator:     validator.New(),
	}
}

// CreateCredit обрабатывает запрос на создание кредита
func (c *CreditController) CreateCredit(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Создаем DTO для запроса
	var dto services.CreateCreditDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Устанавливаем ID пользователя
	dto.UserID = userID

	// Валидируем DTO
	if err := c.validateRequest(dto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создаем кредит
	credit, err := c.creditService.Create(dto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(credit)
}

// GetCredits обрабатывает запрос на получение списка кредитов пользователя
func (c *CreditController) GetCredits(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем список кредитов
	credits, err := c.creditService.GetCreditsByUserID(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(credits)
}

// GetCredit обрабатывает запрос на получение информации о кредите
func (c *CreditController) GetCredit(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем ID кредита из URL
	vars := mux.Vars(r)
	creditID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid credit ID", http.StatusBadRequest)
		return
	}

	// Получаем информацию о кредите
	credit, err := c.creditService.GetCreditByID(uint(creditID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Проверяем, что кредит принадлежит пользователю
	if credit.Account.Holder.ID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(credit)
}

// PayCredit обрабатывает запрос на погашение кредита
func (c *CreditController) PayCredit(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем ID кредита из URL
	vars := mux.Vars(r)
	creditID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid credit ID", http.StatusBadRequest)
		return
	}

	// Создаем DTO для запроса
	var dto services.PayCreditDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Устанавливаем ID кредита
	dto.CreditID = uint(creditID)

	// Валидируем DTO
	if err := c.validateRequest(dto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверяем, что кредит принадлежит пользователю
	credit, err := c.creditService.GetCreditByID(uint(creditID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if credit.Account.Holder.ID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Погашаем кредит
	payment, err := c.creditService.PayCredit(dto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payment)
}

// validateRequest валидирует DTO и возвращает ошибки валидации
func (c *CreditController) validateRequest(dto interface{}) error {
	if err := c.validator.Struct(dto); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		var errorMessages []string
		for _, e := range validationErrors {
			switch e.Tag() {
			case "required":
				errorMessages = append(errorMessages, "поле "+e.Field()+" обязательно")
			case "gt":
				errorMessages = append(errorMessages, "поле "+e.Field()+" должно быть больше 0")
			case "oneof":
				errorMessages = append(errorMessages, "поле "+e.Field()+" должно быть одним из: "+e.Param())
			}
		}
		return errors.New(strings.Join(errorMessages, "; "))
	}
	return nil
}
