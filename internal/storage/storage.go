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
