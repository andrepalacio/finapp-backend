package handlers

import (
	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc *services.UserService
}

func NewUserHandler(svc *services.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// GetProfile godoc
// @Summary      Obtener perfil del usuario autenticado
// @Tags         user
// @Produce      json
// @Success      200  {object}  services.UserProfile
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)

	profile, err := h.svc.GetProfile(c.Request.Context(), services.GetProfileParams{
		UserID: userID,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.OK(c, profile)
}

type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdateProfile godoc
// @Summary      Actualizar nombre o email del usuario
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        body body UpdateProfileRequest true "Campos a actualizar (al menos uno requerido)"
// @Success      200  {object}  services.UserProfile
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Security     BearerAuth
// @Router       /user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid_body", err.Error())
		return
	}

	userID := middleware.UserIDFromContext(c)

	profile, err := h.svc.UpdateProfile(c.Request.Context(), services.UpdateProfileParams{
		UserID: userID,
		Name:   req.Name,
		Email:  req.Email,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.OK(c, profile)
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// ChangePassword godoc
// @Summary      Cambiar contraseña del usuario autenticado
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        body body ChangePasswordRequest true "Contraseña actual y nueva"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /user/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid_body", err.Error())
		return
	}

	userID := middleware.UserIDFromContext(c)

	err := h.svc.ChangePassword(c.Request.Context(), services.ChangePasswordParams{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.NoContent(c)
}

// Delete godoc
// @Summary      Eliminar cuenta del usuario autenticado
// @Tags         user
// @Produce      json
// @Success      204
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /user [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)

	err := h.svc.Delete(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.NoContent(c)
}
