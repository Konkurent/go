package database

import (
	"awesomeProject/config"
	"awesomeProject/models"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

// Database представляет подключение к базе данных
type Database struct {
	DB *gorm.DB
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase(cfg *config.Config) (*Database, error) {
	// Формируем строку подключения
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.DBName,
	)

	// Открываем подключение
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	return &Database{DB: db}, nil
}

// GetDB возвращает экземпляр GORM
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// Close закрывает подключение к базе данных
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Connect устанавливает соединение с базой данных и выполняет миграции
func Connect(cfg *config.Config) (*gorm.DB, error) {
	// Формируем строку подключения
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.DBName,
	)

	// Настраиваем логгер
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Устанавливаем соединение
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	// Настраиваем пул соединений
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пула соединений: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Выполняем SQL миграции
	if err := runMigrations(cfg); err != nil {
		return nil, fmt.Errorf("ошибка выполнения SQL миграций: %v", err)
	}

	// Выполняем автоматическую миграцию моделей
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("ошибка автоматической миграции моделей: %v", err)
	}

	return db, nil
}

// runMigrations выполняет SQL миграции
func runMigrations(cfg *config.Config) error {
	// Формируем URL для миграций
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.DBName,
	)

	// Создаем экземпляр миграции
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		return fmt.Errorf("ошибка создания миграции: %v", err)
	}

	// Выполняем миграции
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка выполнения миграций: %v", err)
	}

	return nil
}

// autoMigrate выполняет автоматическую миграцию моделей
func autoMigrate(db *gorm.DB) error {
	// Автоматическая миграция моделей
	err := db.AutoMigrate(
		&models.User{},
		&models.BankAccount{},
		&models.Transaction{},
		&models.Credit{},
		&models.Payment{},
	)
	if err != nil {
		return fmt.Errorf("ошибка автоматической миграции: %v", err)
	}

	return nil
}

// Методы для работы с пользователями
func (d *Database) CreateUser(user *models.User) error {
	return d.DB.Create(user).Error
}

func (d *Database) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := d.DB.First(&user, id).Error
	return &user, err
}

func (d *Database) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := d.DB.Where("email = ?", email).First(&user).Error
	return &user, err
}

// Методы для работы с банковскими счетами
func (d *Database) CreateBankAccount(account *models.BankAccount) error {
	return d.DB.Create(account).Error
}

func (d *Database) GetBankAccountByID(id uint) (*models.BankAccount, error) {
	var account models.BankAccount
	err := d.DB.First(&account, id).Error
	return &account, err
}

// Методы для работы с транзакциями
func (d *Database) CreateTransaction(transaction *models.Transaction) error {
	return d.DB.Create(transaction).Error
}

func (d *Database) GetTransactionByID(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	err := d.DB.First(&transaction, id).Error
	return &transaction, err
}

// Методы для работы с кредитами
func (d *Database) CreateCredit(credit *models.Credit) error {
	return d.DB.Create(credit).Error
}

func (d *Database) GetCreditByID(id uint) (*models.Credit, error) {
	var credit models.Credit
	err := d.DB.First(&credit, id).Error
	return &credit, err
}

// Методы для работы с платежами
func (d *Database) CreatePayment(payment *models.Payment) error {
	return d.DB.Create(payment).Error
}

func (d *Database) GetPaymentByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	err := d.DB.First(&payment, id).Error
	return &payment, err
}
