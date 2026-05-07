package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type budgetService interface {
	Upsert(ctx context.Context, p services.UpsertBudgetParams) (services.BudgetView, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]services.BudgetView, error)
	GetWithProgress(ctx context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error)
	Delete(ctx context.Context, workspaceID uuid.UUID, year, month int16) error
	UpsertCategory(ctx context.Context, workspaceID uuid.UUID, year, month int16, cat services.BudgetCategoryInput) error
	DeleteCategory(ctx context.Context, workspaceID uuid.UUID, year, month int16, categoryID uuid.UUID) error
}

type BudgetHandler struct {
	svc budgetService
}

func NewBudgetHandler(svc budgetService) *BudgetHandler {
	return &BudgetHandler{svc: svc}
}

type upsertBudgetRequest struct {
	TotalLimit float64                       `json:"total_limit" binding:"required,gt=0"`
	Categories []services.BudgetCategoryInput `json:"categories"`
}

// @Summary     Create or update monthly budget
// @Tags        budgets
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       year  path int true "Year"
// @Param       month path int true "Month (1-12)"
// @Param       body body upsertBudgetRequest true "Budget data"
// @Success     200 {object} services.BudgetView
// @Failure     400 {object} map[string]string
// @Router      /workspaces/{workspace_id}/budgets/{year}/{month} [put]
func (h *BudgetHandler) Upsert(c *gin.Context) {
	year, month, ok := parseYearMonth(c)
	if !ok {
		return
	}

	var req upsertBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	budget, err := h.svc.Upsert(c.Request.Context(), services.UpsertBudgetParams{
		WorkspaceID: wsID,
		Year:        year,
		Month:       month,
		TotalLimit:  req.TotalLimit,
		Categories:  req.Categories,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, budget)
}

// @Summary     List budgets
// @Tags        budgets
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {array} services.BudgetView
// @Router      /workspaces/{workspace_id}/budgets [get]
func (h *BudgetHandler) List(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)
	budgets, err := h.svc.List(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, budgets)
}

// @Summary     Get budget with progress
// @Tags        budgets
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       year  path int true "Year"
// @Param       month path int true "Month (1-12)"
// @Success     200 {object} services.BudgetView
// @Failure     404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/budgets/{year}/{month} [get]
func (h *BudgetHandler) Get(c *gin.Context) {
	year, month, ok := parseYearMonth(c)
	if !ok {
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	budget, err := h.svc.GetWithProgress(c.Request.Context(), wsID, year, month)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, budget)
}

// @Summary     Delete budget
// @Tags        budgets
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       year  path int true "Year"
// @Param       month path int true "Month (1-12)"
// @Success     204
// @Failure     404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/budgets/{year}/{month} [delete]
func (h *BudgetHandler) Delete(c *gin.Context) {
	year, month, ok := parseYearMonth(c)
	if !ok {
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), wsID, year, month); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type upsertBudgetCategoryRequest struct {
	LimitAmount float64 `json:"limit_amount" binding:"required,gt=0"`
}

// @Summary     Set category sub-limit
// @Tags        budgets
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       year  path int true "Year"
// @Param       month path int true "Month"
// @Param       category_id path string true "Category ID"
// @Param       body body upsertBudgetCategoryRequest true "Sub-limit data"
// @Success     204
// @Router      /workspaces/{workspace_id}/budgets/{year}/{month}/categories/{category_id} [put]
func (h *BudgetHandler) UpsertCategory(c *gin.Context) {
	year, month, ok := parseYearMonth(c)
	if !ok {
		return
	}
	catID, err := uuid.Parse(c.Param("category_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid category_id")
		return
	}

	var req upsertBudgetCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.UpsertCategory(c.Request.Context(), wsID, year, month, services.BudgetCategoryInput{
		CategoryID:  catID,
		LimitAmount: req.LimitAmount,
	}); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary     Remove category sub-limit
// @Tags        budgets
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       year  path int true "Year"
// @Param       month path int true "Month"
// @Param       category_id path string true "Category ID"
// @Success     204
// @Router      /workspaces/{workspace_id}/budgets/{year}/{month}/categories/{category_id} [delete]
func (h *BudgetHandler) DeleteCategory(c *gin.Context) {
	year, month, ok := parseYearMonth(c)
	if !ok {
		return
	}
	catID, err := uuid.Parse(c.Param("category_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid category_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.DeleteCategory(c.Request.Context(), wsID, year, month, catID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseYearMonth(c *gin.Context) (int16, int16, bool) {
	y, err := strconv.Atoi(c.Param("year"))
	if err != nil || y < 2000 || y > 2100 {
		response.BadRequest(c, "INVALID_INPUT", "invalid year")
		return 0, 0, false
	}
	m, err := strconv.Atoi(c.Param("month"))
	if err != nil || m < 1 || m > 12 {
		response.BadRequest(c, "INVALID_INPUT", "month must be 1-12")
		return 0, 0, false
	}
	return int16(y), int16(m), true
}
