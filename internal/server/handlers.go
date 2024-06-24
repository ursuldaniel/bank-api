package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ursuldaniel/bank-api/internal/domain/models"
)

func (s *Server) handleAuthRegister(c *gin.Context) {
	model := &models.RegisterRequest{}
	if err := c.ShouldBindBodyWithJSON(model); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Message: err.Error()})
		return
	}

	if err := s.storage.Register(model); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.Response{Message: "Account successfully registered"})
}

func (s *Server) handleAuthLogin(c *gin.Context) {
	model := &models.LoginRequest{}
	if err := c.ShouldBindBodyWithJSON(model); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Message: err.Error()})
		return
	}

	id, err := s.storage.Login(model)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Message: err.Error()})
		return
	}

	token, err := createToken(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.Response{Message: token})
}

func (s *Server) handleAuthLogout(c *gin.Context) {
	token := c.MustGet("token").(string)
	if err := s.storage.DisableToken(token); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.Response{Message: "Successfully logged out from account"})
}
