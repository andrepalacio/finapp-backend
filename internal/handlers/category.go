package handlers

import (
	"context"
	"net/http"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type categoryService interface {
	Create(ctx context.Context, p services.CreateCategoryParams) (services.CategoryView, error)
	ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error)
	Update(ctx context.Context, p services.UpdateCategoryParams) (services.CategoryView, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
}

type CategoryHandler struct {
	svc categoryService
}

func NewCategoryHandler(svc categoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type createCategoryRequest struct {
	Name  string `json:"name"  binding:"required"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
	Type  string `json:"type"  binding:"required,oneof=expense income both"`
}

// @Summary     Create category
// @Tags        categories
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       body body createCategoryRequest true "Category data"
// @Success     201 {object} services.CategoryView
// @Failure     400 {object} map[string]string
// @Router      /workspaces/{workspace_id}/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	cat, err := h.svc.Create(c.Request.Context(), services.CreateCategoryParams{
		WorkspaceID: wsID,
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Type:        req.Type,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, cat)
}

// @Summary     List categories
// @Tags        categories
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {array} services.CategoryView
// @Router      /workspaces/{workspace_id}/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)
	cats, err := h.svc.ListForWorkspace(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, cats)
}

type updateCategoryRequest struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
	Type  string `json:"type" binding:"omitempty,oneof=expense income both"`
}

// @Summary     Update category
// @Tags        categories
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       category_id path string true "Category ID"
// @Param       body body updateCategoryRequest true "Category data"
// @Success     200 {object} services.CategoryView
// @Failure     403 {object} map[string]string
// @Router      /workspaces/{workspace_id}/categories/{category_id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	catID, err := uuid.Parse(c.Param("category_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid category_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	cat, err := h.svc.Update(c.Request.Context(), services.UpdateCategoryParams{
		ID:          catID,
		WorkspaceID: wsID,
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Type:        req.Type,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, cat)
}

// @Summary     Delete category
// @Tags        categories
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       category_id path string true "Category ID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Router      /workspaces/{workspace_id}/categories/{category_id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	catID, err := uuid.Parse(c.Param("category_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid category_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), catID, wsID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
