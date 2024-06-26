package server

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ursuldaniel/bank-api/internal/domain/models"
)

type Storage interface {
	Register(model *models.RegisterRequest) error
	Login(model *models.LoginRequest) (int, error)
	IsTokenValid(token string) error
	DisableToken(token string) error
	GetProfile(id int) (*models.ProfileResponse, error)
	UpdateProfile(id int, model *models.UpdateProfileRequest) error
	UpdatePassword(id int, model *models.UpdatePasswordRequest) error
	Deposit(id int, amount int) error
	Withdraw(id int, amount int) error
	Transfer(fromId int, toId int, amount int) error
	ListTransactions(id int) ([]*models.TransactionResponse, error)
	GetTransaction(id int, transactionId int) (*models.TransactionResponse, error)
}

type Server struct {
	listenAddr string
	storage    Storage
	validate   *validator.Validate
}

func NewServer(listenAddr string, storage Storage) *Server {
	return &Server{
		listenAddr: listenAddr,
		storage:    storage,
		validate:   validator.New(),
	}
}

func (s *Server) Run() error {
	app := gin.Default()

	auth := app.Group("/auth")
	auth.POST("/register", s.handleAuthRegister)
	auth.POST("/login", s.handleAuthLogin)
	auth.POST("/logout", jwtAuth(s), s.handleAuthLogout)

	accounts := app.Group("/accounts", jwtAuth(s))
	accounts.GET("/profile", s.handleGetProfile)
	accounts.PUT("/profile", s.handleUpdateProfile)
	accounts.PUT("/password", s.handleUpdatePassword)
	accounts.POST("/deposit", s.handleDeposit)
	accounts.POST("/withdraw", s.handleWithdraw)
	accounts.POST("/transfer/:id", s.handleTransfer)
	accounts.GET("/transactions", s.handleListTransactions)
	accounts.GET("/transaction/:id", s.handleGetTransaction)

	return app.Run(s.listenAddr)
}

func createToken(id int) (string, error) {
	claims := &jwt.MapClaims{
		"id":        id,
		"expiresAt": time.Now().Add(time.Hour * 72).Unix(),
	}

	secret := os.Getenv("SECRET_KEY")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func jwtAuth(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Request.Header["Authorization"]
		if tokenString == nil {
			c.JSON(http.StatusUnauthorized, models.Response{Message: "Authorization token is missing"})
			c.Abort()
			return
		}

		if err := s.storage.IsTokenValid(tokenString[0]); err != nil {
			c.JSON(http.StatusBadRequest, models.Response{Message: "Invalid authorization token"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString[0], func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("SECRET_KEY")), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, models.Response{Message: "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.Response{Message: "Invalid token claims"})
			c.Abort()
			return
		}

		id, ok := claims["id"].(float64)
		if !ok {
			c.JSON(http.StatusForbidden, models.Response{Message: "Unauthorized access to the account"})
			c.Abort()
			return
		}

		c.Set("id", int(id))
		c.Set("token", tokenString[0])

		c.Next()
	}
}
