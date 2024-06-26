package models

import "time"

type Response struct {
	Message string `json:"message"`
}

type RegisterRequest struct {
	Login      string `json:"login"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Surname    string `json:"surname"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ProfileResponse struct {
	Id         int       `json:"id"`
	Login      string    `json:"login"`
	FirstName  string    `json:"first_name"`
	SecondName string    `json:"second_name"`
	Surname    string    `json:"surname"`
	Email      string    `json:"email"`
	Balance    int       `json:"balance"`
	CreatedAt  time.Time `json:"created_at"`
}

type UpdateProfileRequest struct {
	Login      string `json:"login"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Surname    string `json:"surname"`
	Email      string `json:"email"`
}

type UpdatePasswordRequest struct {
	OldPasssword string `json:"old_password"`
	NewPassword  string `json:"new_password"`
}

type TransactionResponse struct {
	Id              int       `json:"id"`
	TransactionType string    `json:"transaction_type"`
	FromId          int       `json:"from_id,omitempty"`
	ToId            int       `json:"to_id,omitempty"`
	Amount          int       `json:"amount"`
	Transferred_at  time.Time `json:"transferred_at"`
}
