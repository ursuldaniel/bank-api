package storage

import (
	"context"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/ursuldaniel/bank-api/internal/domain/models"
	"golang.org/x/crypto/bcrypt"
)

type PostgresStorage struct {
	conn *pgx.Conn
}

func NewPostgresStorage(ctx context.Context, connStr string) (*PostgresStorage, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, err
	}
	// defer conn.Close(context.Background())

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	if err := CreatePostgresDB(ctx, conn); err != nil {
		return nil, err
	}

	return &PostgresStorage{
		conn: conn,
	}, nil
}

func CreatePostgresDB(ctx context.Context, conn *pgx.Conn) error {
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

	_, err := conn.Exec(ctx, query)
	return err
}

func (s *PostgresStorage) Register(model *models.RegisterRequest) error {
	if err := isDataUnique(s.conn, model.Login); err != nil {
		return err
	}

	hashedPassword, err := hashPassword(model.Password)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `INSERT INTO accounts
	(login, first_name, second_name, surname, email, password, balance, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = s.conn.Exec(ctx, query, model.Login, model.FirstName, model.SecondName, model.Surname, model.Email, hashedPassword, 0, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Login(model *models.LoginRequest) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT id, password FROM accounts WHERE login = $1`
	rows, err := s.conn.Query(ctx, query, model.Login)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var count int
	query := `SELECT COUNT(*) FROM tokens WHERE token = $1`
	if err := s.conn.QueryRow(ctx, query, token).Scan(&count); err != nil {
		return err
	}

	if count != 0 {
		return fmt.Errorf("token is invalid")
	}

	return nil
}

func (s *PostgresStorage) DisableToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `INSERT INTO tokens (token) VALUES ($1)`
	_, err := s.conn.Exec(ctx, query, token)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) GetProfile(id int) (*models.ProfileResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT id, login, first_name, second_name, surname, email, balance, created_at FROM accounts WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, id)
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
	if err := isDataUnique(s.conn, model.Login); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `UPDATE accounts SET login = $1, first_name = $2, second_name = $3, surname = $4, email = $5 WHERE id = $6`
	_, err := s.conn.Exec(ctx, query, model.Login, model.FirstName, model.SecondName, model.Surname, model.Email, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) UpdatePassword(id int, model *models.UpdatePasswordRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT password FROM accounts WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, id)
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
	_, err = s.conn.Exec(ctx, query, newHashedPassword, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Deposit(id int, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT balance FROM accounts WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, id)
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
	_, err = s.conn.Exec(ctx, query, balance, id)
	if err != nil {
		return err
	}

	err = addTransaction(s.conn, "Deposit", id, id, amount)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Withdraw(id int, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT balance FROM accounts WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, id)
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
	_, err = s.conn.Exec(ctx, query, balance, id)
	if err != nil {
		return err
	}

	err = addTransaction(s.conn, "Withdraw", id, id, amount)
	if err != nil {
		return err
	}

	return err
}

func (s *PostgresStorage) Transfer(fromId int, toId int, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT balance FROM accounts WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, fromId)
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

	query = `SELECT balance FROM accounts WHERE id = $1`
	rows, err = s.conn.Query(ctx, query, toId)
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

	query = `UPDATE accounts SET balance = $1 WHERE id = $2`
	_, err = s.conn.Exec(ctx, query, fromBalance, fromId)
	if err != nil {
		return err
	}

	query = `UPDATE accounts SET balance = $1 WHERE id = $2`
	_, err = s.conn.Exec(ctx, query, toBalance, toId)
	if err != nil {
		return err
	}

	err = addTransaction(s.conn, "Deposit", fromId, toId, amount)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) ListTransactions(id int) ([]*models.TransactionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT * FROM transactions WHERE from_id = $1 OR to_id = $1`
	rows, err := s.conn.Query(ctx, query, id)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `SELECT * FROM accounts WHERE id = $1`
	rows, err := s.conn.Query(ctx, query, transactionId)
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

func isDataUnique(conn *pgx.Conn, login string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var count int
	query := `SELECT COUNT(*) FROM accounts WHERE login = $1`
	err := conn.QueryRow(ctx, query, login).Scan(&count)
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

func addTransaction(conn *pgx.Conn, transactionType string, from int, to int, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	query := `INSERT INTO transactions
	(transaction_type, from_id, to_id, amount, transferred_at)
	VALUES ($1, $2, $3, $4, $5)`
	_, err := conn.Exec(ctx, query, transactionType, from, to, amount, time.Now())
	if err != nil {
		return err
	}

	return err
}
