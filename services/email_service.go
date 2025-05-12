package services

import (
	"awesomeProject/config"
	"fmt"
	"gopkg.in/gomail.v2"
	"time"
)

// EmailService предоставляет методы для отправки email
type EmailService struct {
	dialer *gomail.Dialer
	from   string
	config *config.Config
}

// NewEmailService создает новый экземпляр EmailService
func NewEmailService(cfg *config.Config) *EmailService {
	dialer := gomail.NewDialer(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
	)

	return &EmailService{
		dialer: dialer,
		from:   cfg.SMTP.From,
		config: cfg,
	}
}

// SendEmail отправляет email
func (s *EmailService) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("ошибка отправки email: %v", err)
	}

	return nil
}

// SendTransactionNotification отправляет уведомление о транзакции
func (s *EmailService) SendTransactionNotification(to, accountNumber string, amount float64, transactionType string) error {
	subject := "Уведомление о транзакции"
	body := fmt.Sprintf(`
		<h2>Уведомление о транзакции</h2>
		<p>Счет: %s</p>
		<p>Тип операции: %s</p>
		<p>Сумма: %.2f</p>
		<p>Дата: %s</p>
	`, accountNumber, transactionType, amount, time.Now().Format("02.01.2006 15:04:05"))

	return s.SendEmail(to, subject, body)
}

// SendCreditNotification отправляет уведомление о кредите
func (s *EmailService) SendCreditNotification(to, accountNumber string, amount float64, term int) error {
	subject := "Уведомление о кредите"
	body := fmt.Sprintf(`
		<h2>Уведомление о кредите</h2>
		<p>Счет: %s</p>
		<p>Сумма кредита: %.2f</p>
		<p>Срок кредита: %d месяцев</p>
		<p>Дата: %s</p>
	`, accountNumber, amount, term, time.Now().Format("02.01.2006 15:04:05"))

	return s.SendEmail(to, subject, body)
}

// SendCreditPaidNotification отправляет уведомление о погашении кредита
func (s *EmailService) SendCreditPaidNotification(email string, creditID uint) error {
	// Формируем тему письма
	subject := "Поздравляем! Ваш кредит успешно погашен"

	// Формируем тело письма
	body := fmt.Sprintf(`
		<h2>Поздравляем!</h2>
		<p>Ваш кредит #%d был успешно погашен.</p>
		<p>Спасибо, что выбрали наш банк!</p>
		<p>Если у вас возникнут вопросы, пожалуйста, свяжитесь с нами.</p>
		<p>С уважением,<br>Команда банка</p>
	`, creditID)

	// Создаем сообщение
	message := gomail.NewMessage()
	message.SetHeader("From", s.from)
	message.SetHeader("To", email)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", body)

	// Отправляем письмо
	if err := s.dialer.DialAndSend(message); err != nil {
		return fmt.Errorf("ошибка при отправке уведомления о погашении кредита: %v", err)
	}

	return nil
}
