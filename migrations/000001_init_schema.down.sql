-- Удаляем триггеры
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_bank_accounts_updated_at ON bank_accounts;
DROP TRIGGER IF EXISTS update_credits_updated_at ON credits;
DROP TRIGGER IF EXISTS update_credit_payments_updated_at ON credit_payments;

-- Удаляем функцию триггера
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаляем индексы
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_bank_accounts_holder_id;
DROP INDEX IF EXISTS idx_bank_accounts_number;
DROP INDEX IF EXISTS idx_transactions_account_id;
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_credits_account_id;
DROP INDEX IF EXISTS idx_credits_status;
DROP INDEX IF EXISTS idx_credit_payments_credit_id;
DROP INDEX IF EXISTS idx_credit_payments_due_date;
DROP INDEX IF EXISTS idx_credit_payments_status;

-- Удаляем таблицы
DROP TABLE IF EXISTS credit_payments;
DROP TABLE IF EXISTS credits;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS bank_accounts;
DROP TABLE IF EXISTS users;

-- Удаляем тип operation_type
DROP TYPE IF EXISTS operation_type; 