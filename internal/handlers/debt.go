package handlers

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type debtService interface {
	Create(ctx context.Context, p services.CreateDebtParams) (models.Debt, error)
	GetByID(ctx context.Context, id, workspaceID uuid.UUID) (models.Debt, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error)
	Update(ctx context.Context, p services.UpdateDebtParams) (models.Debt, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
	GetSchedule(ctx context.Context, id, workspaceID uuid.UUID) ([]models.DebtScheduleInstallment, error)
	RecordPayment(ctx context.Context, workspaceID uuid.UUID, p services.RecordPaymentParams) (models.DebtPayment, error)
	ListPayments(ctx context.Context, debtID, workspaceID uuid.UUID) ([]models.DebtPayment, error)
	UpdatePayment(ctx context.Context, workspaceID uuid.UUID, p services.UpdatePaymentParams) (models.DebtPayment, error)
	DeletePayment(ctx context.Context, paymentID, debtID, workspaceID uuid.UUID) error
}

type DebtHandler struct {
	svc debtService
}

func NewDebtHandler(svc debtService) *DebtHandler {
	return &DebtHandler{svc: svc}
}

type createDebtRequest struct {
	Name             string  `json:"name" binding:"required"`
	Lender           string  `json:"lender"`
	Principal        float64 `json:"principal" binding:"required,gt=0"`
	Rate             float64 `json:"rate" binding:"min=0"`
	RateType         string  `json:"rate_type" binding:"required"`
	Installments     int32   `json:"installments" binding:"required,min=1"`
	FirstPaymentDate string  `json:"first_payment_date" binding:"required"`
	Notes            string  `json:"notes"`
	InsuranceRate    float64 `json:"insurance_rate"`
	InsuranceType    string  `json:"insurance_type"`
}

// @Summary     Create debt
// @Tags        debts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       body body createDebtRequest true "Debt data"
// @Success     201 {object} models.Debt
// @Failure     400 {object} map[string]string
// @Router      /workspaces/{workspace_id}/debts [post]
func (h *DebtHandler) Create(c *gin.Context) {
	var req createDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}

	firstPayment, err := time.Parse("2006-01-02", req.FirstPaymentDate)
	if err != nil {
		response.BadRequest(c, "INVALID_DATE", "first_payment_date must be YYYY-MM-DD")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	debt, err := h.svc.Create(c.Request.Context(), services.CreateDebtParams{
		WorkspaceID:      wsID,
		Name:             req.Name,
		Lender:           req.Lender,
		Principal:        req.Principal,
		Rate:             req.Rate,
		RateType:         req.RateType,
		Installments:     req.Installments,
		FirstPaymentDate: firstPayment,
		Notes:            req.Notes,
		InsuranceRate:    req.InsuranceRate,
		InsuranceType:    req.InsuranceType,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, debt)
}

// @Summary     List debts
// @Tags        debts
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {array} models.Debt
// @Router      /workspaces/{workspace_id}/debts [get]
func (h *DebtHandler) List(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)
	debts, err := h.svc.List(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, debts)
}

// @Summary     Get debt
// @Tags        debts
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Success     200 {object} models.Debt
// @Router      /workspaces/{workspace_id}/debts/{debt_id} [get]
func (h *DebtHandler) Get(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	debt, err := h.svc.GetByID(c.Request.Context(), debtID, wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, debt)
}

type updateDebtRequest struct {
	Name             string  `json:"name" binding:"required"`
	Lender           string  `json:"lender"`
	Principal        float64 `json:"principal" binding:"required,gt=0"`
	Rate             float64 `json:"rate" binding:"min=0"`
	RateType         string  `json:"rate_type" binding:"required"`
	Installments     int32   `json:"installments" binding:"required,min=1"`
	FirstPaymentDate string  `json:"first_payment_date" binding:"required"`
	Notes            string  `json:"notes"`
	InsuranceRate    float64 `json:"insurance_rate"`
	InsuranceType    string  `json:"insurance_type"`
}

// @Summary     Update debt
// @Tags        debts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Param       body body updateDebtRequest true "Debt data"
// @Success     200 {object} models.Debt
// @Router      /workspaces/{workspace_id}/debts/{debt_id} [put]
func (h *DebtHandler) Update(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	var req updateDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}
	firstPayment, err := time.Parse("2006-01-02", req.FirstPaymentDate)
	if err != nil {
		response.BadRequest(c, "INVALID_DATE", "first_payment_date must be YYYY-MM-DD")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	debt, err := h.svc.Update(c.Request.Context(), services.UpdateDebtParams{
		ID:               debtID,
		WorkspaceID:      wsID,
		Name:             req.Name,
		Lender:           req.Lender,
		Principal:        req.Principal,
		Rate:             req.Rate,
		RateType:         req.RateType,
		Installments:     req.Installments,
		FirstPaymentDate: firstPayment,
		Notes:            req.Notes,
		InsuranceRate:    req.InsuranceRate,
		InsuranceType:    req.InsuranceType,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, debt)
}

// @Summary     Delete debt
// @Tags        debts
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Success     204
// @Router      /workspaces/{workspace_id}/debts/{debt_id} [delete]
func (h *DebtHandler) Delete(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), debtID, wsID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(204)
}

// @Summary     Get amortization schedule
// @Tags        debts
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Success     200 {array} models.DebtScheduleInstallment
// @Router      /workspaces/{workspace_id}/debts/{debt_id}/schedule [get]
func (h *DebtHandler) GetSchedule(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	schedule, err := h.svc.GetSchedule(c.Request.Context(), debtID, wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, schedule)
}

type recordPaymentRequest struct {
	Period int32   `json:"period" binding:"required,min=1"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
	PaidAt string  `json:"paid_at" binding:"required"`
	Notes  string  `json:"notes"`
}

// @Summary     Record debt payment
// @Tags        debts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Param       body body recordPaymentRequest true "Payment data"
// @Success     201 {object} models.DebtPayment
// @Router      /workspaces/{workspace_id}/debts/{debt_id}/payments [post]
func (h *DebtHandler) RecordPayment(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	var req recordPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}
	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		response.BadRequest(c, "INVALID_DATE", "paid_at must be YYYY-MM-DD")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	payment, err := h.svc.RecordPayment(c.Request.Context(), wsID, services.RecordPaymentParams{
		DebtID: debtID,
		Period: req.Period,
		Amount: req.Amount,
		PaidAt: paidAt,
		Notes:  req.Notes,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, payment)
}

// @Summary     List debt payments
// @Tags        debts
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Success     200 {array} models.DebtPayment
// @Router      /workspaces/{workspace_id}/debts/{debt_id}/payments [get]
func (h *DebtHandler) ListPayments(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	payments, err := h.svc.ListPayments(c.Request.Context(), debtID, wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, payments)
}

type updatePaymentRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	PaidAt string  `json:"paid_at" binding:"required"`
	Notes  string  `json:"notes"`
}

// @Summary     Update debt payment
// @Tags        debts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Param       payment_id path string true "Payment ID"
// @Param       body body updatePaymentRequest true "Payment data"
// @Success     200 {object} models.DebtPayment
// @Router      /workspaces/{workspace_id}/debts/{debt_id}/payments/{payment_id} [put]
func (h *DebtHandler) UpdatePayment(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	paymentID, err := uuid.Parse(c.Param("payment_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid payment_id")
		return
	}
	var req updatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}
	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		response.BadRequest(c, "INVALID_DATE", "paid_at must be YYYY-MM-DD")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	payment, err := h.svc.UpdatePayment(c.Request.Context(), wsID, services.UpdatePaymentParams{
		PaymentID: paymentID,
		DebtID:    debtID,
		Amount:    req.Amount,
		PaidAt:    paidAt,
		Notes:     req.Notes,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, payment)
}

// @Summary     Delete debt payment
// @Tags        debts
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       debt_id path string true "Debt ID"
// @Param       payment_id path string true "Payment ID"
// @Success     204
// @Router      /workspaces/{workspace_id}/debts/{debt_id}/payments/{payment_id} [delete]
func (h *DebtHandler) DeletePayment(c *gin.Context) {
	debtID, err := uuid.Parse(c.Param("debt_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid debt_id")
		return
	}
	paymentID, err := uuid.Parse(c.Param("payment_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid payment_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.DeletePayment(c.Request.Context(), paymentID, debtID, wsID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(204)
}
