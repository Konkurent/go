package services

import (
	"awesomeProject/models"
	"errors"
	"gorm.io/gorm"
	"log"
	"time"
)

// PaymentSchedulerService предоставляет методы для автоматической обработки платежей
type PaymentSchedulerService struct {
	db            *gorm.DB
	creditService *CreditService
}

// NewPaymentSchedulerService создает новый экземпляр PaymentSchedulerService
func NewPaymentSchedulerService(db *gorm.DB, creditService *CreditService) *PaymentSchedulerService {
	return &PaymentSchedulerService{
		db:            db,
		creditService: creditService,
	}
}

// Start запускает планировщик платежей
func (s *PaymentSchedulerService) Start() {
	// Запускаем обработку регулярных платежей каждые 8 часов
	regularTicker := time.NewTicker(8 * time.Hour)
	go func() {
		for {
			select {
			case <-regularTicker.C:
				if err := s.processPayments(); err != nil {
					log.Printf("Ошибка при обработке регулярных платежей: %v", err)
				}
			}
		}
	}()

	// Запускаем обработку просроченных платежей каждый час
	overdueTicker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-overdueTicker.C:
				if err := s.processOverduePayments(); err != nil {
					log.Printf("Ошибка при обработке просроченных платежей: %v", err)
				}
			}
		}
	}()
}

// processOverduePayments обрабатывает просроченные платежи
func (s *PaymentSchedulerService) processOverduePayments() error {
	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.New("ошибка при начале транзакции")
	}

	// Получаем все просроченные платежи
	var payments []models.Payment
	if err := tx.Where("is_overdue = ? AND status = ?", true, models.PaymentStatusOverdue).
		Preload("Credit").
		Preload("Credit.Account").
		Find(&payments).Error; err != nil {
		tx.Rollback()
		return errors.New("ошибка при получении просроченных платежей")
	}

	for _, payment := range payments {
		if err := s.processPayment(tx, &payment); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return errors.New("ошибка при подтверждении транзакции")
	}

	return nil
}

// processPayments обрабатывает платежи, срок которых наступил
func (s *PaymentSchedulerService) processPayments() error {
	// Начинаем транзакцию
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.New("ошибка при начале транзакции")
	}

	// Получаем все платежи, срок которых наступил
	var payments []models.Payment
	if err := tx.Where("pay_date <= ? AND status = ?", time.Now(), models.PaymentStatusPlanned).
		Preload("Credit").
		Preload("Credit.Account").
		Find(&payments).Error; err != nil {
		tx.Rollback()
		return errors.New("ошибка при получении платежей")
	}

	for _, payment := range payments {
		if err := s.processPayment(tx, &payment); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return errors.New("ошибка при подтверждении транзакции")
	}

	return nil
}

// processPayment обрабатывает один платеж
func (s *PaymentSchedulerService) processPayment(tx *gorm.DB, payment *models.Payment) error {
	// Проверяем достаточно ли средств на счете
	if payment.Credit.Account.Balance < payment.Amount {
		if payment.IsOverdue {
			return nil
		}
		// Если средств не хватает, помечаем как просроченный
		payment.IsOverdue = true
		payment.Status = models.PaymentStatusOverdue
		// Увеличиваем сумму на 10%
		payment.Amount *= 1.1

		// Обновляем платеж
		if err := tx.Save(payment).Error; err != nil {
			return errors.New("ошибка при обновлении просроченного платежа")
		}

		// Обновляем статус кредита
		payment.Credit.Status = models.CreditStatusOverdue
		if err := tx.Save(&payment.Credit).Error; err != nil {
			return errors.New("ошибка при обновлении статуса кредита")
		}

		return nil
	}

	// Списываем средства со счета
	payment.Credit.Account.Balance -= payment.Amount
	if err := tx.Save(&payment.Credit.Account).Error; err != nil {
		return errors.New("ошибка при списании средств")
	}

	// Обновляем статус платежа
	now := time.Now()
	payment.Status = models.PaymentStatusPaid
	payment.RealPayDate = &now

	// Сохраняем платеж
	if err := tx.Save(payment).Error; err != nil {
		return errors.New("ошибка при обновлении платежа")
	}

	// Создаем запись о транзакции
	transaction := &models.Transaction{
		AccountID:   payment.Credit.AccountID,
		Amount:      -payment.Amount,
		Type:        string(TransactionTypeWithdraw),
		Description: "Credit payment",
	}

	// Сохраняем транзакцию
	if err := tx.Create(transaction).Error; err != nil {
		return errors.New("ошибка при сохранении транзакции")
	}

	return nil
}
