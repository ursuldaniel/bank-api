package server

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/ursuldaniel/bank-api/internal/domain/models"
)

type Storage interface {
	Register(model *models.RegisterRequest) error
	Login(model *models.LoginRequest) (int, error)
	IsTokenValid(token string) error
	DisableToken(token string) error
}

type Server struct {
	listenAddr string
	storage    Storage
}

func NewServer(listenAddr string, storage Storage) *Server {
	return &Server{
		listenAddr: listenAddr,
		storage:    storage,
	}
}

func (s *Server) Run() error {
	app := gin.Default()

	auth := app.Group("/auth")
	auth.POST("/register", s.handleAuthRegister)
	auth.POST("/login", s.handleAuthLogin)
	auth.POST("/logout", jwtAuth(s), s.handleAuthLogout)

	accounts := app.Group("/accounts", jwtAuth(s))
	accounts.POST("/profile", nil)
	accounts.PUT("/profile", nil)
	accounts.POST("/deposit", nil)
	accounts.POST("/withdraw", nil)
	accounts.POST("/transfer/:id", nil)
	accounts.GET("/transactions", nil)
	accounts.GET("/transaction/:id", nil)

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
