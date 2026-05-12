package handlers

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type alertBudgetService interface {
	GetWithProgress(ctx context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error)
}

type AlertHandler struct {
	budgetSvc alertBudgetService
}

func NewAlertHandler(budgetSvc alertBudgetService) *AlertHandler {
	return &AlertHandler{budgetSvc: budgetSvc}
}

type Alert struct {
	Type        string  `json:"type"`
	CategoryID  *string `json:"category_id,omitempty"`
	CategoryName string `json:"category_name,omitempty"`
	Limit       float64 `json:"limit"`
	Spent       float64 `json:"spent"`
	Overage     float64 `json:"overage"`
	Message     string  `json:"message"`
}

type AlertsResponse struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Alerts []Alert `json:"alerts"`
}

// @Summary     Get budget alerts for current month
// @Tags        budgets
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {object} AlertsResponse
// @Router      /workspaces/{workspace_id}/alerts [get]
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)

	now := time.Now()
	year := int16(now.Year())
	month := int16(now.Month())

	budget, err := h.budgetSvc.GetWithProgress(c.Request.Context(), wsID, year, month)
	if err != nil {
		// No budget set — no alerts
		response.OK(c, AlertsResponse{
			Year:   now.Year(),
			Month:  int(now.Month()),
			Alerts: []Alert{},
		})
		return
	}

	alerts := make([]Alert, 0)

	// Total budget alert
	if budget.TotalSpent > budget.TotalLimit {
		overage := budget.TotalSpent - budget.TotalLimit
		alerts = append(alerts, Alert{
			Type:    "budget_exceeded",
			Limit:   budget.TotalLimit,
			Spent:   budget.TotalSpent,
			Overage: overage,
			Message: "El gasto total del mes supera el presupuesto",
		})
	} else if budget.TotalLimit > 0 && budget.TotalSpent/budget.TotalLimit >= 0.9 {
		alerts = append(alerts, Alert{
			Type:    "budget_warning",
			Limit:   budget.TotalLimit,
			Spent:   budget.TotalSpent,
			Overage: 0,
			Message: "Estas al 90% del presupuesto mensual",
		})
	}

	// Per-category alerts
	for _, cat := range budget.Categories {
		if cat.Spent > cat.LimitAmount {
			overage := cat.Spent - cat.LimitAmount
			catIDStr := cat.CategoryID.String()
			alerts = append(alerts, Alert{
				Type:         "category_exceeded",
				CategoryID:   &catIDStr,
				CategoryName: cat.CategoryName,
				Limit:        cat.LimitAmount,
				Spent:        cat.Spent,
				Overage:      overage,
				Message:      "La categoria supera su limite de presupuesto",
			})
		}
	}

	response.OK(c, AlertsResponse{
		Year:   now.Year(),
		Month:  int(now.Month()),
		Alerts: alerts,
	})
}
