package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/ursuldaniel/bank-api/internal/domain/models"
	"golang.org/x/crypto/bcrypt"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := CreatePostgresDB(db); err != nil {
		return nil, err
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

func CreatePostgresDB(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS accounts (
		id SERIAL,
		login TEXT,
		first_name TEXT,
		second_name TEXT,
		surname TEXT,
		email TEXT,
		password TEXT,
		balance INT,
		created_at DATE
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id SERIAL,
		transaction_type TEXT,
		from_id TEXT,
		to_id TEXT,
		amount INT,
		transferred_at DATE
	);
	
	CREATE TABLE IF NOT EXISTS tokens (
		token TEXT
	)`

	_, err := db.Exec(query)
	return err
}

func (s *PostgresStorage) Register(model *models.RegisterRequest) error {
	query := `INSERT INTO accounts
	(login, first_name, second_name, surname, email, password, balance, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if err := isDataUnique(s.db, model.Login); err != nil {
		return err
	}

	hashedPassword, err := hashPassword(model.Password)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query, model.Login, model.FirstName, model.SecondName, model.Surname, model.Email, hashedPassword, 0, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Login(model *models.LoginRequest) (int, error) {
	query := `SELECT id, password FROM accounts WHERE login = $1`

	rows, err := s.db.Query(query, model.Login)
	if err != nil {
		return -1, err
	}

	var id int
	var password string
	for rows.Next() {
		err := rows.Scan(
			&id,
			&password,
		)

		if err != nil {
			return -1, err
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(model.Password)); err != nil {
		return -1, err
	}

	return id, nil
}

func (s *PostgresStorage) IsTokenValid(token string) error {
	query := `SELECT COUNT(*) FROM tokens WHERE token = $1`

	var count int
	if err := s.db.QueryRow(query, token).Scan(&count); err != nil {
		return err
	}

	if count != 0 {
		return fmt.Errorf("token is invalid")
	}

	return nil
}

func (s *PostgresStorage) DisableToken(token string) error {
	query := `INSERT INTO tokens (token) VALUES ($1)`

	_, err := s.db.Exec(query, token)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) GetProfile(id int) (*models.ProfileResponse, error) {
	query := `SELECT id, login, first_name, second_name, surname, email, balance, created_at FROM accounts WHERE id = $1`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	model := &models.ProfileResponse{}
	for rows.Next() {
		err := rows.Scan(
			&model.Id,
			&model.Login,
			&model.FirstName,
			&model.SecondName,
			&model.Surname,
			&model.Email,
			&model.Balance,
			&model.CreatedAt,
		)

		if err != nil {
			return nil, err
		}
	}

	return model, nil
}

func (s *PostgresStorage) UpdateProfile(id int, model *models.UpdateProfileRequest) error {
	query := `UPDATE accounts SET login = $1, first_name = $2, second_name = $3, surname = $4, email = $5 WHERE id = $6`

	if err := isDataUnique(s.db, model.Login); err != nil {
		return err
	}

	_, err := s.db.Exec(query, model.Login, model.FirstName, model.SecondName, model.Surname, model.Email, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) UpdatePassword(id int, model *models.UpdatePasswordRequest) error {
	query := `SELECT password FROM accounts WHERE id = $1`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return err
	}

	var password string
	for rows.Next() {
		err := rows.Scan(
			&password,
		)

		if err != nil {
			return err
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(model.OldPasssword)); err != nil {
		return err
	}

	newHashedPassword, err := hashPassword(model.NewPassword)
	if err != nil {
		return err
	}

	query = `UPDATE accounts SET password = $1 WHERE id = $2`

	_, err = s.db.Exec(query, newHashedPassword, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Deposit(id int, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	query := `SELECT balance FROM accounts WHERE id = $1`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return err
	}

	var balance int
	for rows.Next() {
		err := rows.Scan(
			&balance,
		)

		if err != nil {
			return err
		}
	}

	balance += amount

	query = `UPDATE accounts SET balance = $1 WHERE id = $2`

	_, err = s.db.Exec(query, balance, id)
	if err != nil {
		return err
	}

	err = addTransaction(s.db, "Deposit", id, id, amount)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Withdraw(id int, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	query := `SELECT balance FROM accounts WHERE id = $1`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return err
	}

	var balance int
	for rows.Next() {
		err := rows.Scan(
			&balance,
		)

		if err != nil {
			return err
		}
	}

	if balance < amount {
		return fmt.Errorf("invalid amount")
	}

	balance -= amount

	query = "UPDATE accounts SET balance = $1 WHERE id = $2"

	_, err = s.db.Exec(query, balance, id)
	if err != nil {
		return err
	}

	err = addTransaction(s.db, "Withdraw", id, id, amount)
	if err != nil {
		return err
	}

	return err
}

func (s *PostgresStorage) Transfer(fromId int, toId int, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	rows, err := s.db.Query(`SELECT balance FROM accounts WHERE id = $1`, fromId)
	if err != nil {
		return err
	}

	var fromBalance int
	for rows.Next() {
		err := rows.Scan(
			&fromBalance,
		)

		if err != nil {
			return err
		}
	}

	rows, err = s.db.Query(`SELECT balance FROM accounts WHERE id = $1`, toId)
	if err != nil {
		return err
	}

	var toBalance int
	for rows.Next() {
		err := rows.Scan(
			&toBalance,
		)

		if err != nil {
			return err
		}
	}

	if fromBalance-amount < 0 {
		return fmt.Errorf("invalid amount")
	}

	fromBalance -= amount
	toBalance += amount

	_, err = s.db.Exec("UPDATE accounts SET balance = $1 WHERE id = $2", fromBalance, fromId)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("UPDATE accounts SET balance = $1 WHERE id = $2", toBalance, toId)
	if err != nil {
		return err
	}

	err = addTransaction(s.db, "Deposit", fromId, toId, amount)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) ListTransactions(id int) ([]*models.TransactionResponse, error) {
	query := `SELECT * FROM transactions WHERE from_id = $1 OR to_id = $1`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	var fromId, toId int
	transactions := []*models.TransactionResponse{}
	for rows.Next() {
		transaction := &models.TransactionResponse{}
		err := rows.Scan(
			&transaction.Id,
			&transaction.TransactionType,
			&fromId,
			&toId,
			&transaction.Amount,
			&transaction.Transferred_at,
		)

		if err != nil {
			return nil, err
		}

		if fromId != toId {
			transaction.FromId = fromId
			transaction.ToId = toId
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (s *PostgresStorage) GetTransaction(id int, transactionId int) (*models.TransactionResponse, error) {
	query := `SELECT * FROM accounts WHERE id = $1`

	rows, err := s.db.Query(query, transactionId)
	if err != nil {
		return nil, err
	}

	var fromId, toId int
	transaction := &models.TransactionResponse{}
	for rows.Next() {
		err := rows.Scan(
			&transaction.Id,
			&transaction.TransactionType,
			&fromId,
			&toId,
			&transaction.Amount,
			&transaction.Transferred_at,
		)

		if err != nil {
			return nil, err
		}

		if fromId != id && toId != id {
			return nil, fmt.Errorf("access denied")
		}

		if fromId != toId {
			transaction.FromId = fromId
			transaction.ToId = toId
		}
	}

	return transaction, nil
}

func isDataUnique(db *sql.DB, login string) error {
	count := 0

	err := db.QueryRow("SELECT COUNT(*) FROM accounts WHERE login = $1", login).Scan(&count)
	if err != nil {
		return err
	}

	if count != 0 {
		return fmt.Errorf("non unique data")
	}

	return nil
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil
	}

	return string(hashedPassword), nil
}

func addTransaction(db *sql.DB, transactionType string, from int, to int, amount int) error {
	query := `INSERT INTO transactions
	(transaction_type, from_id, to_id, amount, transferred_at)
	VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, transactionType, from, to, amount, time.Now())
	if err != nil {
		return err
	}

	return err
}
