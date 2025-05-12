package services

import (
	"awesomeProject/models"
	"errors"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// TransactionType представляет тип транзакции
type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "DEPOSIT"
	TransactionTypeWithdraw TransactionType = "WITHDRAW"
	TransactionTypeTransfer TransactionType = "TRANSFER"
)

type BankAccountDTO struct {
	ID        uint    `json:"id"`
	Holder    UserDTO `json:"holder"`
	Balance   float64 `json:"balance"`
	Title     string  `json:"title"`
	Number    string  `json:"number"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

// TransferRequest представляет данные для перевода средств
type TransferRequest struct {
	SourceID      uint    `json:"source_id" validate:"required"`
	DestinationID uint    `json:"destination_id" validate:"required"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
}

// TransactionRequest представляет данные для транзакции
type TransactionRequest struct {
	AccountID uint            `json:"account_id" validate:"required"`
	Amount    float64         `json:"amount" validate:"required,gt=0"`
	Type      TransactionType `json:"type" validate:"required,oneof=DEPOSIT WITHDRAW TRANSFER"`
}

// CreateBankAccountDTO представляет данные для создания банковского счета
type CreateBankAccountDTO struct {
	BankName string  `json:"bank_name" validate:"required,min=2,max=100"`
	Balance  float64 `json:"balance" validate:"gte=0"`
	Title    string  `json:"title" validate:"omitempty,min=2,max=100"`
	UserID   uint    `json:"user_id" validate:"required"`
}

// BankService предоставляет методы для работы с банковскими счетами
type BankService struct {
	db        *gorm.DB
	validator *validator.Validate
	email     *EmailService
}

// NewBankService создает новый экземпляр BankService
func NewBankService(db *gorm.DB, email *EmailService) *BankService {
	return &BankService{
		db:        db,
		validator: validator.New(),
		email:     email,
	}
}

// GetDB возвращает экземпляр базы данных
func (s *BankService) GetDB() *gorm.DB {
	return s.db
}

// GetById возвращает банковский счет по ID
func (s *BankService) GetById(id uint) (*models.BankAccount, error) {
	var account models.BankAccount

	// Ищем счет в базе данных
	if err := s.db.First(&account, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("банковский счет не найден")
		}
		return nil, errors.New("ошибка при поиске банковского счета")
	}

	return &account, nil
}

// GetAllByUserId возвращает все банковские счета пользователя
func (s *BankService) GetAllByUserId(userId uint) ([]models.BankAccount, error) {
	var accounts []models.BankAccount

	// Ищем все счета пользователя
	if err := s.db.Where("holder_id = ?", userId).Find(&accounts).Error; err != nil {
		return nil, errors.New("ошибка при поиске банковских счетов")
	}

	// Если счетов не найдено, возвращаем пустой слайс
	if len(accounts) == 0 {
		return []models.BankAccount{}, nil
	}

	return accounts, nil
}

// CreateBankAccount создает новый банковский счет
func (s *BankService) CreateBankAccount(dto CreateBankAccountDTO) (*BankAccountDTO, error) {
	// Валидируем DTO
	if err := s.validator.Struct(dto); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		var errorMessages []string
		for _, e := range validationErrors {
			switch e.Tag() {
			case "required":
				errorMessages = append(errorMessages, "поле "+e.Field()+" обязательно")
			case "min":
				errorMessages = append(errorMessages, "поле "+e.Field()+" должно содержать минимум "+e.Param()+" символов")
			case "max":
				errorMessages = append(errorMessages, "поле "+e.Field()+" должно содержать максимум "+e.Param()+" символов")
			case "gte":
				errorMessages = append(errorMessages, "поле "+e.Field()+" должно быть больше или равно "+e.Param())
			}
		}
		return nil, errors.New(strings.Join(errorMessages, "; "))
	}

	// Устанавливаем значения по умолчанию
	if dto.Title == "" {
		dto.Title = "Go White"
	}

	// Генерируем номер счета
	accountNumber := s.generateAccountNumber()

	// Создаем счет
	account := &models.BankAccount{
		Number:    accountNumber,
		Bank:      dto.BankName,
		Balance:   dto.Balance,
		Title:     dto.Title,
		HolderID:  dto.UserID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Сохраняем счет
	if err := s.db.Create(account).Error; err != nil {
		return nil, errors.New("не удалось создать банковский счет")
	}

	// Получаем данные пользователя
	var user models.User
	if err := s.db.First(&user, dto.UserID).Error; err != nil {
		return nil, errors.New("ошибка при получении данных пользователя")
	}

	// Конвертируем в DTO
	return &BankAccountDTO{
		ID: account.ID,
		Holder: UserDTO{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
		},
		Balance:   account.Balance,
		Title:     account.Title,
		Number:    account.Number,
		CreatedAt: account.CreatedAt.Format(time.RFC3339),
		UpdatedAt: account.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// generateAccountNumber генерирует номер банковского счета
func (s *BankService) generateAccountNumber() string {
	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())

	// Генерируем 20 случайных цифр
	var number strings.Builder
	for i := 0; i < 20; i++ {
		number.WriteString(strconv.Itoa(rand.Intn(10)))
	}

	return number.String()
}

// Deposit пополняет банковский счет
func (s *BankService) Deposit(request TransactionRequest) (*BankAccountDTO, error) {
	// Валидируем запрос
	if err := s.validator.Struct(request); err != nil {
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
		return nil, errors.New(strings.Join(errorMessages, "; "))
	}

	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.New("ошибка при начале транзакции")
	}

	// Получаем счет
	var account models.BankAccount
	if err := tx.First(&account, request.AccountID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("банковский счет не найден")
		}
		return nil, errors.New("ошибка при поиске банковского счета")
	}

	// Обновляем баланс
	account.Balance += request.Amount
	account.UpdatedAt = time.Now()

	// Сохраняем изменения в счете
	if err := tx.Save(&account).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при обновлении баланса")
	}

	if request.Type == TransactionTypeDeposit {
		// Создаем запись о транзакции
		transaction := &models.Transaction{
			AccountID:   request.AccountID,
			Amount:      request.Amount,
			Type:        string(TransactionTypeDeposit),
			Description: "ATM",
		}

		// Сохраняем транзакцию
		if err := tx.Create(transaction).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("ошибка при сохранении транзакции")
		}

		// Отправляем уведомление
		if err := s.email.SendTransactionNotification(account.Holder.Email, account.Number, request.Amount, "Пополнение"); err != nil {
			log.Printf("Ошибка отправки уведомления: %v", err)
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("ошибка при подтверждении транзакции")
	}

	return &BankAccountDTO{
		ID: account.ID,
		Holder: UserDTO{
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
	}, nil
}

// Withdraw снимает средства с банковского счета
func (s *BankService) Withdraw(request TransactionRequest) (*BankAccountDTO, error) {
	// Валидируем запрос
	if err := s.validator.Struct(request); err != nil {
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
		return nil, errors.New(strings.Join(errorMessages, "; "))
	}

	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.New("ошибка при начале транзакции")
	}

	// Получаем счет
	var account models.BankAccount
	if err := tx.First(&account, request.AccountID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("банковский счет не найден")
		}
		return nil, errors.New("ошибка при поиске банковского счета")
	}

	// Проверяем достаточность средств
	if account.Balance < request.Amount {
		tx.Rollback()
		return nil, errors.New("недостаточно средств на счете")
	}

	// Обновляем баланс
	account.Balance -= request.Amount
	account.UpdatedAt = time.Now()

	// Сохраняем изменения в счете
	if err := tx.Save(&account).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при обновлении баланса")
	}

	if request.Type == TransactionTypeWithdraw {
		// Создаем запись о транзакции
		transaction := &models.Transaction{
			AccountID:   request.AccountID,
			Amount:      request.Amount,
			Type:        string(TransactionTypeWithdraw),
			Description: "ATM",
		}

		// Сохраняем транзакцию
		if err := tx.Create(transaction).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("ошибка при сохранении транзакции")
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("ошибка при подтверждении транзакции")
	}

	return &BankAccountDTO{
		ID: account.ID,
		Holder: UserDTO{
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
	}, nil
}

// Transfer переводит средства между счетами
func (s *BankService) Transfer(request TransferRequest) error {
	// Валидируем запрос
	if err := s.validator.Struct(request); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		var errorMessages []string
		for _, e := range validationErrors {
			switch e.Tag() {
			case "required":
				errorMessages = append(errorMessages, "поле "+e.Field()+" обязательно")
			case "gt":
				errorMessages = append(errorMessages, "поле "+e.Field()+" должно быть больше 0")
			}
		}
		return errors.New(strings.Join(errorMessages, "; "))
	}

	// Проверяем, что счета разные
	if request.SourceID == request.DestinationID {
		return errors.New("нельзя перевести средства на тот же счет")
	}

	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.New("ошибка при начале транзакции")
	}

	// Снимаем средства с исходного счета
	sourceAccount, err := s.Withdraw(TransactionRequest{
		AccountID: request.SourceID,
		Amount:    request.Amount,
		Type:      TransactionTypeTransfer,
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	// Зачисляем средства на целевой счет
	destinationAccount, err := s.Deposit(TransactionRequest{
		AccountID: request.DestinationID,
		Amount:    request.Amount,
		Type:      TransactionTypeTransfer,
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	// Создаем запись о транзакции перевода
	sourceTransaction := &models.Transaction{
		AccountID:   request.SourceID,
		Amount:      request.Amount,
		Type:        string(TransactionTypeTransfer),
		Description: "Transfer to account " + destinationAccount.Number,
	}

	// Создаем запись о транзакции перевода
	destinationTransaction := &models.Transaction{
		AccountID:   request.DestinationID,
		Amount:      request.Amount,
		Type:        string(TransactionTypeTransfer),
		Description: "Transfer from account " + sourceAccount.Number,
	}

	// Сохраняем транзакцию
	if err := tx.Create(sourceTransaction).Error; err != nil {
		tx.Rollback()
		return errors.New("ошибка при сохранении транзакции")
	}

	// Сохраняем транзакцию
	if err := tx.Create(destinationTransaction).Error; err != nil {
		tx.Rollback()
		return errors.New("ошибка при сохранении транзакции")
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return errors.New("ошибка при подтверждении транзакции")
	}

	return nil
}

// GetAccountsByUserID возвращает список банковских счетов пользователя
func (s *BankService) GetAccountsByUserID(userID uint) ([]models.BankAccount, error) {
	var accounts []models.BankAccount
	if err := s.db.Where("holder_id = ?", userID).
		Preload("Holder").
		Find(&accounts).Error; err != nil {
		return nil, errors.New("ошибка при получении списка счетов")
	}
	return accounts, nil
}
