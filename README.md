# Банковское API

## Запуск приложения

### Предварительные требования
- Go 1.21 или выше
- PostgreSQL 14 или выше
- Make (опционально)

### Установка зависимостей
```bash
go mod download
```

### Настройка окружения
1. Создайте файл `.env` в корневой директории проекта:
```env
# Настройки сервера
SERVER_PORT=8080
SERVER_HOST=localhost

# Настройки базы данных
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=bank_db
DB_SSL_MODE=disable

# Настройки JWT
JWT_SECRET=your_jwt_secret
JWT_EXPIRES_IN=24

# Настройки SMTP
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASSWORD=your_app_password

# Настройки карт
CARD_PRIVATE_KEY=your_private_key
CARD_PUBLIC_KEY=your_public_key
CARD_HMAC_KEY=your_hmac_key
```

### Запуск базы данных
```bash
docker run --name bank-postgres \
    -e POSTGRES_DB=bank_db \
    -e POSTGRES_USER=postgres \
    -e POSTGRES_PASSWORD=your_password \
    -p 5432:5432 \
    -d postgres:14
```

### Миграции базы данных
```bash
go run migrations/migrate.go
```

### Запуск приложения
```bash
go run main.go
```


### Сборка приложения
```bash
go build -o bank-api
```

### Запуск через Make (если установлен)
```bash
# Установка зависимостей
make deps

# Запуск миграций
make migrate

# Запуск приложения
make run

# Запуск тестов
make test

# Сборка приложения
make build
```

## Аутентификация

### POST /api/auth/signup
Регистрация нового пользователя
```json
{
    "firstName": "string",
    "lastName": "string",
    "email": "string",
    "password": "string"
}
```

### POST /api/auth/signin
Вход в систему
```json
{
    "email": "string",
    "password": "string"
}
```

## Банковские счета

### POST /api/accounts
Создание нового банковского счета
```json
{
    "bankName": "string",
    "balance": "number",
    "title": "string"
}
```

### GET /api/accounts
Получение списка банковских счетов пользователя

### POST /api/accounts/{id}/deposit
Пополнение счета
```json
{
    "amount": "number"
}
```

### POST /api/accounts/{id}/withdraw
Снятие средств со счета
```json
{
    "amount": "number"
}
```

### POST /api/accounts/transfer
Перевод средств между счетами
```json
{
    "sourceId": "number",
    "destinationId": "number",
    "amount": "number"
}
```

## Банковские карты

### POST /api/cards
Создание новой банковской карты
```json
{
    "accountId": "number"
}
```

### GET /api/cards
Получение списка банковских карт пользователя

## Кредиты

### POST /api/credits
Создание нового кредита
```json
{
    "amount": "number",
    "term": "number",
    "accountId": "number"
}
```

### GET /api/credits
Получение списка кредитов пользователя

### POST /api/credits/{id}/pay
Погашение кредита
```json
{
    "amount": "number"
}
```

## Требования к паролю
- Минимум 8 символов
- Минимум 1 цифра
- Минимум 1 заглавная буква
- Минимум 1 строчная буква
- Минимум 1 специальный символ

## Формат ответов

### Успешный ответ
```json
{
    "data": {},
    "message": "string"
}
```

### Ошибка
```json
{
    "error": "string"
}
```

## Аутентификация
Все защищенные эндпоинты требуют заголовок Authorization с JWT токеном:
```
Authorization: Bearer <token>
``` 