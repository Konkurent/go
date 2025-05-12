package services

import (
	"awesomeProject/models"
	"errors"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"log"
	"math"
	"strings"
	"time"
)

// CreateCreditDTO представляет данные для создания кредита
type CreateCreditDTO struct {
	AccountID uint    `json:"account_id" validate:"required"`
	Amount    float64 `json:"amount" validate:"required,gt=0"`
	Months    int     `json:"months" validate:"required,gt=0"`
	UserID    uint    `json:"-" validate:"required"`
}

// PaymentDTO представляет данные платежа
type PaymentDTO struct {
	ID          uint       `json:"id"`
	PayDate     time.Time  `json:"pay_date"`
	Amount      float64    `json:"amount"`
	InitAmount  float64    `json:"init_amount"`
	IsOverdue   bool       `json:"is_overdue"`
	Status      string     `json:"status"`
	RealPayDate *time.Time `json:"real_pay_date,omitempty"`
}

// CreditResponseDTO представляет ответ с данными кредита
type CreditResponseDTO struct {
	ID              uint           `json:"id"`
	Rate            float64        `json:"rate"`
	Amount          float64        `json:"amount"`
	RemainingAmount float64        `json:"remaining_amount"`
	Status          string         `json:"status"`
	StartDate       time.Time      `json:"start_date"`
	EndDate         time.Time      `json:"end_date"`
	Payments        []PaymentDTO   `json:"payments"`
	NextPayment     *PaymentDTO    `json:"next_payment,omitempty"`
	User            UserDTO        `json:"user"`
	Account         BankAccountDTO `json:"account"`
}

// PaymentSchedule представляет график платежей
type PaymentSchedule struct {
	Payments []models.Payment
}

// PayCreditDTO представляет данные для погашения кредита
type PayCreditDTO struct {
	Amount    float64 `json:"amount" validate:"required,gt=0"`
	AccountID uint    `json:"account_id" validate:"required"`
	CreditID  uint    `json:"-"`
}

// CreditService предоставляет методы для работы с кредитами
type CreditService struct {
	db        *gorm.DB
	validator *validator.Validate
	email     *EmailService
}

// NewCreditService создает новый экземпляр CreditService
func NewCreditService(db *gorm.DB, email *EmailService) *CreditService {
	return &CreditService{
		db:        db,
		validator: validator.New(),
		email:     email,
	}
}

// calculateAnnuityPayment рассчитывает размер аннуитетного платежа
func (s *CreditService) calculateAnnuityPayment(amount float64, rate float64, months int) float64 {
	// Конвертируем годовую ставку в месячную (в долях)
	monthlyRate := rate / 12 / 100

	// Рассчитываем коэффициент аннуитета
	annuityCoefficient := (monthlyRate * math.Pow(1+monthlyRate, float64(months))) / (math.Pow(1+monthlyRate, float64(months)) - 1)

	// Рассчитываем размер платежа
	return amount * annuityCoefficient
}

// generatePaymentSchedule генерирует график платежей
func (s *CreditService) generatePaymentSchedule(credit *models.Credit) []models.Payment {
	// Рассчитываем количество месяцев между датами
	months := int(credit.EndDate.Sub(credit.StartDate).Hours() / 24 / 30)

	payments := make([]models.Payment, months)
	remainingAmount := credit.Amount
	monthlyRate := credit.Rate / 12 / 100

	// Рассчитываем размер аннуитетного платежа
	annuityPayment := s.calculateAnnuityPayment(credit.Amount, credit.Rate, months)

	for i := 0; i < months; i++ {
		// Рассчитываем проценты за текущий месяц
		interest := remainingAmount * monthlyRate

		// Рассчитываем основной долг
		principal := annuityPayment - interest

		// Обновляем оставшуюся сумму
		remainingAmount -= principal

		// Создаем платеж
		payDate := credit.StartDate.AddDate(0, i+1, 0)
		payments[i] = models.Payment{
			CreditID:    credit.ID,
			PayDate:     payDate,
			Amount:      annuityPayment,
			InitAmount:  annuityPayment,
			IsOverdue:   false,
			Status:      models.PaymentStatusPlanned,
			RealPayDate: nil,
		}
	}

	return payments
}

// toPaymentDTO конвертирует модель Payment в DTO
func (s *CreditService) toPaymentDTO(payment models.Payment) PaymentDTO {
	return PaymentDTO{
		ID:          payment.ID,
		PayDate:     payment.PayDate,
		Amount:      payment.Amount,
		InitAmount:  payment.InitAmount,
		IsOverdue:   payment.IsOverdue,
		Status:      string(payment.Status),
		RealPayDate: payment.RealPayDate,
	}
}

// Create создает новый кредит
func (s *CreditService) Create(dto CreateCreditDTO) (*CreditResponseDTO, error) {
	// Валидируем DTO
	if err := s.validator.Struct(dto); err != nil {
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
		return nil, errors.New(strings.Join(errorMessages, "; "))
	}

	// Получаем ставку из центрального банка
	rate, err := GetCentralBankRate()
	if err != nil {
		return nil, errors.New("ошибка при получении ставки центрального банка")
	}

	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.New("ошибка при начале транзакции")
	}

	// Проверяем существование счета
	var account models.BankAccount
	if err := tx.Preload("Holder").First(&account, dto.AccountID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("банковский счет не найден")
		}
		return nil, errors.New("ошибка при поиске банковского счета")
	}

	if account.HolderID != dto.UserID {
		return nil, errors.New("Access denied")
	}

	// Проверяем, нет ли уже активного кредита
	var existingCredit models.Credit
	if err := tx.Where("account_id = ? AND status = ?", dto.AccountID, models.CreditStatusActive).First(&existingCredit).Error; err == nil {
		tx.Rollback()
		return nil, errors.New("у счета уже есть активный кредит")
	}

	// Рассчитываем даты
	startDate := time.Now()
	endDate := startDate.AddDate(0, dto.Months, 0)

	// Создаем кредит
	credit := &models.Credit{
		Rate:      rate,
		AccountID: dto.AccountID,
		Amount:    dto.Amount,
		Status:    models.CreditStatusActive,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Сохраняем кредит
	if err := tx.Create(credit).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при создании кредита")
	}

	// Генерируем график платежей
	payments := s.generatePaymentSchedule(credit)

	// Сохраняем платежи
	for _, payment := range payments {
		if err := tx.Create(&payment).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("ошибка при создании платежа")
		}
	}

	// Зачисляем средства на счет
	account.Balance += dto.Amount
	account.UpdatedAt = time.Now()

	// Сохраняем изменения в счете
	if err := tx.Save(&account).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при обновлении баланса")
	}

	// Создаем запись о транзакции
	transaction := &models.Transaction{
		AccountID:   dto.AccountID,
		Amount:      dto.Amount,
		Type:        string(TransactionTypeDeposit),
		Description: "Credit issuance",
	}

	// Сохраняем транзакцию
	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при сохранении транзакции")
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("ошибка при подтверждении транзакции")
	}

	// Конвертируем платежи в DTO
	paymentDTOs := make([]PaymentDTO, len(payments))
	for i, payment := range payments {
		paymentDTOs[i] = s.toPaymentDTO(payment)
	}

	// Формируем ответ
	response := &CreditResponseDTO{
		ID:        credit.ID,
		Rate:      credit.Rate,
		Amount:    credit.Amount,
		Status:    string(credit.Status),
		StartDate: credit.StartDate,
		EndDate:   credit.EndDate,
		Payments:  paymentDTOs,
		User: UserDTO{
			ID:        account.Holder.ID,
			FirstName: account.Holder.FirstName,
			LastName:  account.Holder.LastName,
			Email:     account.Holder.Email,
		},
		Account: BankAccountDTO{
			ID:      account.ID,
			Number:  account.Number,
			Balance: account.Balance,
			Holder: UserDTO{
				ID: account.HolderID,
			},
		},
	}

	return response, nil
}

// GetCreditsByUserID возвращает все кредиты пользователя
func (s *CreditService) GetCreditsByUserID(userID uint) ([]models.Credit, error) {
	var credits []models.Credit
	if err := s.db.Where("user_id = ?", userID).
		Preload("Account.Holder").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("payments.created_at DESC")
		}).
		Find(&credits).Error; err != nil {
		return nil, err
	}
	return credits, nil
}

// calculateNextPayment вычисляет следующий платеж по кредиту
func (s *CreditService) calculateNextPayment(credit models.Credit) *models.Payment {
	// Если кредит погашен, возвращаем nil
	if credit.Status != "ACTIVE" {
		return nil
	}

	// Вычисляем общую сумму платежа
	monthlyPayment := credit.Amount * (credit.Rate / 100 / 12) / (1 - 1/(1+credit.Rate/100/12))

	// Находим последний платеж
	var lastPayment models.Payment
	if err := s.db.Where("credit_id = ?", credit.ID).
		Preload("Credit").
		Order("created_at DESC").
		First(&lastPayment).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	// Если платежей еще не было, следующий платеж через месяц после создания кредита
	nextPaymentDate := credit.CreatedAt.AddDate(0, 1, 0)
	if lastPayment.ID != 0 {
		// Иначе следующий платеж через месяц после последнего платежа
		nextPaymentDate = lastPayment.CreatedAt.AddDate(0, 1, 0)
	}

	// Если дата следующего платежа в прошлом, устанавливаем на текущую дату
	if nextPaymentDate.Before(time.Now()) {
		nextPaymentDate = time.Now()
	}

	return &models.Payment{
		CreditID: credit.ID,
		Amount:   monthlyPayment,
		PayDate:  nextPaymentDate,
		Status:   models.PaymentStatusPlanned,
	}
}

// GetCreditByID возвращает кредит по ID
func (s *CreditService) GetCreditByID(id uint) (*models.Credit, error) {
	var credit models.Credit
	if err := s.db.Preload("Account.Holder").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("payments.created_at DESC")
		}).
		First(&credit, id).Error; err != nil {
		return nil, err
	}
	return &credit, nil
}

// GetCreditsByAccountID возвращает все кредиты по ID счета
func (s *CreditService) GetCreditsByAccountID(accountID uint) ([]models.Credit, error) {
	var credits []models.Credit
	if err := s.db.Where("account_id = ?", accountID).
		Preload("Account.Holder").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("payments.created_at DESC")
		}).
		Find(&credits).Error; err != nil {
		return nil, err
	}
	return credits, nil
}

// PayCredit обрабатывает платеж по кредиту
func (s *CreditService) PayCredit(dto PayCreditDTO) (*PaymentDTO, error) {
	// Валидируем DTO
	if err := s.validator.Struct(dto); err != nil {
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
		return nil, errors.New(strings.Join(errorMessages, "; "))
	}

	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.New("ошибка при начале транзакции")
	}

	// Получаем кредит
	var credit models.Credit
	if err := tx.Preload("Account").
		Preload("Account.Holder").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_date ASC")
		}).
		First(&credit, dto.CreditID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("кредит не найден")
		}
		return nil, errors.New("ошибка при получении информации о кредите")
	}

	// Проверяем статус кредита
	if credit.Status != models.CreditStatusActive {
		tx.Rollback()
		return nil, errors.New("кредит не активен")
	}

	// Проверяем, что счет принадлежит владельцу кредита
	if credit.AccountID != dto.AccountID {
		tx.Rollback()
		return nil, errors.New("неверный номер счета")
	}

	// Получаем счет
	var account models.BankAccount
	if err := tx.Preload("Holder").First(&account, dto.AccountID).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("счет не найден")
	}

	// Проверяем достаточность средств
	if account.Balance < dto.Amount {
		tx.Rollback()
		return nil, errors.New("недостаточно средств на счете")
	}

	// Находим следующий платеж
	var nextPayment models.Payment
	if err := tx.Where("credit_id = ? AND status = ?", dto.CreditID, models.PaymentStatusPlanned).
		Preload("Credit").
		Order("pay_date ASC").
		First(&nextPayment).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("нет запланированных платежей")
	}

	// Проверяем сумму платежа
	if dto.Amount < nextPayment.Amount {
		tx.Rollback()
		return nil, errors.New("сумма платежа меньше минимальной")
	}

	// Обновляем статус платежа
	now := time.Now()
	nextPayment.Status = models.PaymentStatusPaid
	nextPayment.RealPayDate = &now
	nextPayment.Amount = dto.Amount

	if err := tx.Save(&nextPayment).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при обновлении платежа")
	}

	// Списываем средства со счета
	account.Balance -= dto.Amount
	if err := tx.Save(&account).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при обновлении баланса счета")
	}

	// Создаем запись о транзакции
	transaction := &models.Transaction{
		AccountID:   dto.AccountID,
		Amount:      -dto.Amount,
		Type:        string(TransactionTypeWithdraw),
		Description: "Credit payment",
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при создании транзакции")
	}

	// Проверяем, погашен ли кредит
	var remainingPayments int64
	if err := tx.Model(&models.Payment{}).
		Where("credit_id = ? AND status = ?", dto.CreditID, models.PaymentStatusPlanned).
		Count(&remainingPayments).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("ошибка при проверке оставшихся платежей")
	}

	// Если все платежи погашены, закрываем кредит
	if remainingPayments == 0 {
		credit.Status = models.CreditStatusPaid
		if err := tx.Save(&credit).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("ошибка при обновлении статуса кредита")
		}

		// Отправляем уведомление о погашении кредита
		if err := s.email.SendCreditPaidNotification(credit.Account.Holder.Email, credit.ID); err != nil {
			// Логируем ошибку, но не прерываем транзакцию
			log.Printf("Ошибка при отправке уведомления: %v", err)
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("ошибка при подтверждении транзакции")
	}

	// Возвращаем информацию о платеже
	return &PaymentDTO{
		ID:          nextPayment.ID,
		PayDate:     nextPayment.PayDate,
		Amount:      nextPayment.Amount,
		InitAmount:  nextPayment.InitAmount,
		IsOverdue:   nextPayment.IsOverdue,
		Status:      string(nextPayment.Status),
		RealPayDate: nextPayment.RealPayDate,
	}, nil
}
