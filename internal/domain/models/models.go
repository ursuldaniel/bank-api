package models

// type Account struct {
// 	Id         int
// 	Login      string
// 	FirstName  string
// 	SecondName string
// 	Surname    string
// 	Email      string
// 	Password   string
// 	Balance    int
// 	CreatedAt  time.Time
// }

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
