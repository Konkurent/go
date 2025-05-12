package controllers

import (
	"awesomeProject/database"
	"awesomeProject/services"
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"time"
)

// BankController обрабатывает запросы, связанные с банковскими операциями
type BankController struct {
	bankService *services.BankService
	validator   *validator.Validate
}

// NewBankController создает новый экземпляр BankController
func NewBankController(db *database.Database, email *services.EmailService) *BankController {
	return &BankController{
		bankService: services.NewBankService(db.DB, email),
		validator:   validator.New(),
	}
}

// validateRequest валидирует DTO и возвращает ошибки валидации
func (c *BankController) validateRequest(dto interface{}) error {
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

// validateAccountOwnership проверяет, что счет принадлежит пользователю
func (c *BankController) validateAccountOwnership(accountID, userID uint) error {
	account, err := c.bankService.GetById(accountID)
	if err != nil {
		return errors.New("банковский счет не найден")
	}
	if account.HolderID != userID {
		return errors.New("нет доступа к данному счету")
	}
	return nil
}

// CreateBankAccount обрабатывает запрос на создание банковского счета
func (c *BankController) CreateBankAccount(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (установлен middleware)
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Создаем DTO для запроса
	var dto services.CreateBankAccountDTO
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

	// Создаем банковский счет
	account, err := c.bankService.CreateBankAccount(dto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
}

// Deposit обрабатывает запрос на пополнение банковского счета
func (c *BankController) Deposit(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (установлен middleware)
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Создаем DTO для запроса
	var dto services.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Устанавливаем тип транзакции
	dto.Type = services.TransactionTypeDeposit

	// Валидируем DTO
	if err := c.validateRequest(dto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверяем владельца счета
	if err := c.validateAccountOwnership(dto.AccountID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Пополняем счет
	updatedAccount, err := c.bankService.Deposit(dto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedAccount)
}

// Withdraw обрабатывает запрос на снятие средств с банковского счета
func (c *BankController) Withdraw(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (установлен middleware)
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Создаем DTO для запроса
	var dto services.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Устанавливаем тип транзакции
	dto.Type = services.TransactionTypeWithdraw

	// Валидируем DTO
	if err := c.validateRequest(dto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверяем владельца счета
	if err := c.validateAccountOwnership(dto.AccountID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Снимаем средства
	updatedAccount, err := c.bankService.Withdraw(dto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedAccount)
}

// Transfer обрабатывает запрос на перевод средств между счетами
func (c *BankController) Transfer(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (установлен middleware)
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Создаем DTO для запроса
	var dto services.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := c.validateAccountOwnership(dto.SourceID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// Валидируем DTO
	if err := c.validateRequest(dto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверяем владельца исходного счета
	if err := c.validateAccountOwnership(dto.SourceID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Выполняем перевод
	if err := c.bankService.Transfer(dto); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Перевод успешно выполнен",
	})
}

// GetAccounts обрабатывает запрос на получение списка банковских счетов пользователя
func (c *BankController) GetAccounts(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (установлен middleware)
	userID, ok := r.Context().Value("user_id").(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем список счетов пользователя
	accounts, err := c.bankService.GetAccountsByUserID(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Конвертируем BankAccount в BankAccountDTO
	var accountDTOs []services.BankAccountDTO
	for _, account := range accounts {
		// Получаем данные пользователя
		accountDTO := services.BankAccountDTO{
			ID: account.ID,
			Holder: services.UserDTO{
				ID:        account.Holder.ID,
				FirstName: account.Holder.FirstName,
				LastName:  account.Holder.LastName,
				Email:     account.Holder.Email,
			},
			Balance:   account.Balance,
			Title:     account.Title,
			Number:    account.Number,
			CreatedAt: account.CreatedAt.Format(time.RFC3339),
			UpdatedAt: account.UpdatedAt.Format(time.RFC3339),
		}
		accountDTOs = append(accountDTOs, accountDTO)
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(accountDTOs)
}

// RegisterRoutes регистрирует маршруты контроллера
func (c *BankController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/accounts", c.CreateBankAccount).Methods("POST")
	router.HandleFunc("/api/accounts/deposit", c.Deposit).Methods("PUT")
	router.HandleFunc("/api/accounts/withdraw", c.Withdraw).Methods("PUT")
	router.HandleFunc("/api/accounts/transfer", c.Transfer).Methods("POST")
	router.HandleFunc("/api/accounts", c.GetAccounts).Methods("GET")
}
