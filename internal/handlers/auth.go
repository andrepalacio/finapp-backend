package handlers

import (
	"net/http"

	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"     binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	pair, err := h.svc.Register(c.Request.Context(), services.RegisterParams{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, tokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), services.LoginParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.OK(c, tokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.OK(c, tokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
