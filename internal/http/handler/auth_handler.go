package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"studyroom/internal/models"
	"studyroom/internal/service"
)

type AuthHandler struct{ svc service.AuthService }

func NewAuthHandler(s service.AuthService) *AuthHandler { return &AuthHandler{svc: s} }

type creds struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in creds
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if err := h.svc.Register(in.Email, in.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "registered"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in creds
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	tok, exp, err := h.svc.Login(in.Email, in.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	maxAge := int(time.Until(exp).Seconds())
	c.SetCookie("session_token", tok, maxAge, "/", "", false, true) // set Secure:true behind HTTPS
	c.JSON(http.StatusOK, gin.H{"status": "logged_in"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if tok, err := c.Cookie("session_token"); err == nil {
		_ = h.svc.Logout(tok)
	}
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	u := c.MustGet("user").(*models.User)
	c.JSON(http.StatusOK, u)
}
