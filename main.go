package main

import (
	"awesomeProject/config"
	"awesomeProject/controllers"
	"awesomeProject/database"
	"awesomeProject/middleware"
	"awesomeProject/services"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func initPaymentScheduler(db *database.Database, emailService *services.EmailService) {
	// Создаем сервис кредитов
	creditService := services.NewCreditService(db.DB, emailService)

	// Создаем планировщик платежей
	scheduler := services.NewPaymentSchedulerService(db.DB, creditService)

	// Запускаем планировщик
	scheduler.Start()
	log.Println("Планировщик платежей запущен")
}

func main() {
	// Инициализируем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем подключение к базе данных
	db, err := database.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	// Инициализируем сервис email
	emailService := services.NewEmailService(cfg)

	// Запускаем планировщик платежей
	initPaymentScheduler(db, emailService)

	// Создаем роутер
	router := mux.NewRouter()

	// Инициализируем контроллеры
	authController := controllers.NewAuthController(db)
	bankController := controllers.NewBankController(db, emailService)
	creditController := controllers.NewCreditController(db, emailService)

	// Публичные маршруты для аутентификации
	router.HandleFunc("/api/auth/signUp", authController.SignUp).Methods("POST")
	router.HandleFunc("/api/auth/signIn", authController.SignIn).Methods("POST")

	// Защищенные маршруты
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(middleware.AuthMiddleware([]byte(authController.GetJWTKey())))
	protected.Use(middleware.LoggingMiddleware)

	// Маршруты для работы с банковскими счетами
	protected.HandleFunc("/bank/accounts", bankController.CreateBankAccount).Methods("POST")
	protected.HandleFunc("/bank/accounts", bankController.GetAccounts).Methods("GET")
	protected.HandleFunc("/bank/accounts/{id}/deposit", bankController.Deposit).Methods("POST")
	protected.HandleFunc("/bank/accounts/{id}/withdraw", bankController.Withdraw).Methods("POST")
	protected.HandleFunc("/bank/accounts/{id}/transfer", bankController.Transfer).Methods("POST")

	// Маршруты для работы с кредитами
	protected.HandleFunc("/bank/credits", creditController.CreateCredit).Methods("POST")
	protected.HandleFunc("/bank/credits", creditController.GetCredits).Methods("GET")
	protected.HandleFunc("/bank/credits/{id}", creditController.GetCredit).Methods("GET")
	protected.HandleFunc("/bank/credits/{id}/pay", creditController.PayCredit).Methods("POST")

	// Запускаем сервер
	port := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Сервер запущен на порту %s", port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
