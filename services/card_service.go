package services

import (
	"awesomeProject/config"
	"awesomeProject/models"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"gorm.io/gorm"
	"io"
	"strconv"
	"strings"
	"time"
)

// CardDTO представляет данные для создания карты
type CardDTO struct {
	UserID     uint   `json:"user_id" validate:"required"`
	AccountID  uint   `json:"account_id" validate:"required"`
	Number     string `json:"number" validate:"required,len=16"`
	CVV        string `json:"cvv" validate:"required,len=3"`
	Expiration string `json:"expiration" validate:"required,len=5"`
}

// CardResponseDTO представляет данные карты для ответа
type CardResponseDTO struct {
	ID         uint   `json:"id"`
	Number     string `json:"number"`
	CVV        string `json:"cvv"`
	Holder     string `json:"holder"`
	Expiration string `json:"expiration"`
	AccountID  uint   `json:"account_id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CardService предоставляет методы для работы с картами
type CardService struct {
	db          *gorm.DB
	config      *config.Config
	bankService *BankService
	userService *UserService
}

// NewCardService создает новый экземпляр CardService
func NewCardService(db *gorm.DB, bankService *BankService, userService *UserService) (*CardService, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, err
	}

	return &CardService{
		db:          db,
		config:      cfg,
		bankService: bankService,
		userService: userService,
	}, nil
}

// CreateCard создает новую карту
func (s *CardService) CreateCard(dto CardDTO) (*CardResponseDTO, error) {
	account, err := s.bankService.GetById(dto.AccountID)
	if err != nil {
		return nil, err
	}

	if account.HolderID != dto.UserID {
		return nil, errors.New("банковский счет не принадлежит пользователю")
	}

	cardNumber := s.generateCardNumber()

	if s.validateLuhn(cardNumber) {
		return nil, errors.New("Номер карты не проходит проверку по алгоритму Луна")
	}

	expirationDate := calculateExpirationDate()
	expirationStr := expirationDate.Format("01/06")

	hashedCVV, error := s.hashCVV(s.generateCVV())
	if error != nil {
		return nil, err
	}

	encryptedNumber, err := s.encryptData(cardNumber)
	if err != nil {
		return nil, errors.New("не удалось зашифровать номер карты")
	}

	encryptedExpiration, err := s.encryptData(expirationStr)
	if err != nil {
		return nil, errors.New("не удалось зашифровать дату истечения")
	}

	card := &models.Card{
		NumberEncrypted:     encryptedNumber,
		NumberHMAC:          s.calculateHMAC(cardNumber),
		ExpirationEncrypted: encryptedExpiration,
		ExpirationHMAC:      s.calculateHMAC(expirationStr),
		CVV:                 hashedCVV,
		AccountID:           dto.AccountID,
	}

	if err := s.db.Create(card).Error; err != nil {
		return nil, errors.New("не удалось создать карту")
	}

	return s.cardToResponseDTO(card)
}

// GetAllByUserID возвращает все карты пользователя
func (s *CardService) GetAllByUserID(userID uint) ([]CardResponseDTO, error) {
	accounts, err := s.bankService.GetAllByUserId(userID)
	if err != nil {
		return nil, err
	}

	var accountIDs []uint
	for _, account := range accounts {
		accountIDs = append(accountIDs, account.ID)
	}

	if len(accountIDs) == 0 {
		return []CardResponseDTO{}, nil
	}

	var cards []models.Card
	if err := s.db.Where("account_id IN ?", accountIDs).Find(&cards).Error; err != nil {
		return nil, errors.New("не удалось получить карты")
	}

	var response []CardResponseDTO
	for _, card := range cards {
		dto, err := s.cardToResponseDTO(&card)
		if err != nil {
			return nil, err
		}
		response = append(response, *dto)
	}

	return response, nil
}

// Вспомогательные методы

func (s *CardService) cardToResponseDTO(card *models.Card) (*CardResponseDTO, error) {
	number, err := s.decryptData(card.NumberEncrypted)
	if err != nil {
		return nil, errors.New("не удалось расшифровать номер карты")
	}

	expiration, err := s.decryptData(card.ExpirationEncrypted)
	if err != nil {
		return nil, errors.New("не удалось расшифровать дату истечения")
	}

	return &CardResponseDTO{
		ID:         card.ID,
		Number:     maskCardNumber(number),
		CVV:        "***",
		Holder:     card.Account.Holder.LastName + " " + card.Account.Holder.FirstName,
		Expiration: expiration,
		AccountID:  card.AccountID,
		CreatedAt:  card.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  card.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// decryptData расшифровывает данные с помощью PGP
func (s *CardService) decryptData(encryptedData string) (string, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(s.config.CardPrivateKey))
	if err != nil {
		return "", err
	}

	buf := strings.NewReader(encryptedData)
	md, err := openpgp.ReadMessage(buf, entityList, nil, &packet.Config{})
	if err != nil {
		return "", err
	}

	decrypted, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// maskCardNumber маскирует номер карты
func maskCardNumber(number string) string {
	if len(number) != 16 {
		return number
	}
	return number[:4] + " **** **** " + number[12:]
}

// calculateExpirationDate вычисляет дату истечения срока действия карты
// (текущий месяц/год + 10 лет)
func calculateExpirationDate() time.Time {
	now := time.Now()
	// Добавляем 10 лет к текущей дате
	expiration := now.AddDate(10, 0, 0)
	// Устанавливаем последний день месяца
	return time.Date(expiration.Year(), expiration.Month()+1, 0, 0, 0, 0, 0, time.UTC)
}

// generateCardNumber генерирует номер карты
func (s *CardService) generateCardNumber() string {
	// Генерируем первые 15 цифр
	number := ""
	for i := 0; i < 15; i++ {
		number += strconv.Itoa(int(time.Now().UnixNano() % 10))
	}

	// Вычисляем контрольную сумму
	sum := 0
	for i := 0; i < len(number); i++ {
		digit := int(number[i] - '0')
		if i%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}

	// Добавляем контрольную цифру
	checkDigit := (10 - (sum % 10)) % 10
	return number + strconv.Itoa(checkDigit)
}

// generateCVV генерирует номер карты
func (s *CardService) generateCVV() string {
	// Генерируем первые 3 цифр
	number := ""
	for i := 0; i < 3; i++ {
		number += strconv.Itoa(int(time.Now().UnixNano() % 10))
	}

	return number
}

// hashCVV хэширует CVV код
func (s *CardService) hashCVV(cvv string) (string, error) {
	hashedCVV, err := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedCVV), nil
}

// encryptData шифрует данные с помощью PGP
func (s *CardService) encryptData(data string) (string, error) {
	// Загружаем публичный ключ
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(s.config.CardPublicKey))
	if err != nil {
		return "", err
	}

	// Создаем буфер для зашифрованных данных
	var buf strings.Builder
	w, err := openpgp.Encrypt(&buf, entityList, nil, nil, &packet.Config{})
	if err != nil {
		return "", err
	}

	// Записываем данные
	if _, err := w.Write([]byte(data)); err != nil {
		return "", err
	}

	// Закрываем writer
	if err := w.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// calculateHMAC вычисляет HMAC для данных
func (s *CardService) calculateHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(s.config.CardHMACKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// validateLuhn проверяет номер карты по алгоритму Луна
func (s *CardService) validateLuhn(number string) bool {
	sum := 0
	for i := 0; i < len(number); i++ {
		digit := int(number[i] - '0')
		if i%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
