package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type transactionService interface {
	Create(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error)
	CreateTransfer(ctx context.Context, p services.CreateTransferParams) (services.TransferResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (services.TransactionView, error)
	List(ctx context.Context, p services.ListTransactionsParams) (services.TransactionListResult, error)
	DailySummary(ctx context.Context, p services.DailySummaryParams) (services.DailySummaryResult, error)
	ListByDate(ctx context.Context, p services.ListByDateParams) (services.CursorListResult, error)
	Update(ctx context.Context, p services.UpdateTransactionParams) (services.TransactionView, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
}

type TransactionHandler struct {
	svc transactionService
}

func NewTransactionHandler(svc transactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

type createTransactionRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	Type        string     `json:"type"   binding:"required,oneof=expense income"`
	Amount      float64    `json:"amount" binding:"required,gt=0"`
	Description string     `json:"description"`
	Date        string     `json:"date"   binding:"required"`
}

// @Summary     Create transaction
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       body body createTransactionRequest true "Transaction data"
// @Success     201 {object} services.TransactionView
// @Failure     400 {object} map[string]string
// @Router      /workspaces/{workspace_id}/transactions [post]
func (h *TransactionHandler) Create(c *gin.Context) {
	var req createTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "date must be YYYY-MM-DD")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	tx, err := h.svc.Create(c.Request.Context(), services.CreateTransactionParams{
		WorkspaceID: wsID,
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Type:        req.Type,
		Amount:      req.Amount,
		Description: req.Description,
		Date:        date,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, tx)
}

type createTransferRequest struct {
	ToWorkspaceID uuid.UUID `json:"to_workspace_id" binding:"required"`
	Amount        float64   `json:"amount"          binding:"required,gt=0"`
	Description   string    `json:"description"`
	Note          string    `json:"note"`
	Date          string    `json:"date"            binding:"required"`
}

// @Summary     Create transfer between workspaces
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "From Workspace ID"
// @Param       body body createTransferRequest true "Transfer data"
// @Success     201 {object} services.TransferResult
// @Failure     400 {object} map[string]string
// @Router      /workspaces/{workspace_id}/transactions/transfer [post]
func (h *TransactionHandler) CreateTransfer(c *gin.Context) {
	var req createTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "date must be YYYY-MM-DD")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	result, err := h.svc.CreateTransfer(c.Request.Context(), services.CreateTransferParams{
		FromWorkspaceID: wsID,
		ToWorkspaceID:   req.ToWorkspaceID,
		UserID:          userID,
		Amount:          req.Amount,
		Description:     req.Description,
		Note:            req.Note,
		Date:            date,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, result)
}

// @Summary     Get transaction
// @Tags        transactions
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       transaction_id path string true "Transaction ID"
// @Success     200 {object} services.TransactionView
// @Failure     404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/transactions/{transaction_id} [get]
func (h *TransactionHandler) Get(c *gin.Context) {
	txID, err := uuid.Parse(c.Param("transaction_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid transaction_id")
		return
	}
	tx, err := h.svc.GetByID(c.Request.Context(), txID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, tx)
}

// @Summary     List transactions (offset pagination)
// @Tags        transactions
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       date_from query string false "YYYY-MM-DD"
// @Param       date_to   query string false "YYYY-MM-DD"
// @Param       type      query string false "expense|income|transfer"
// @Param       category_id query string false "Category UUID"
// @Param       limit     query int false "Default 20"
// @Param       offset    query int false "Default 0"
// @Success     200 {object} services.TransactionListResult
// @Router      /workspaces/{workspace_id}/transactions [get]
func (h *TransactionHandler) List(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)

	p := services.ListTransactionsParams{
		WorkspaceID: wsID,
		Type:        c.Query("type"),
		Limit:       int32(queryInt(c, "limit", 20)),
		Offset:      int32(queryInt(c, "offset", 0)),
	}

	if s := c.Query("date_from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.BadRequest(c, "INVALID_INPUT", "date_from must be YYYY-MM-DD")
			return
		}
		p.DateFrom = &t
	}
	if s := c.Query("date_to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.BadRequest(c, "INVALID_INPUT", "date_to must be YYYY-MM-DD")
			return
		}
		p.DateTo = &t
	}
	if s := c.Query("category_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "INVALID_INPUT", "invalid category_id")
			return
		}
		p.CategoryID = &id
	}

	result, err := h.svc.List(c.Request.Context(), p)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, result)
}

// @Summary     Daily summary (offset pagination)
// @Tags        transactions
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       date_from query string false "YYYY-MM-DD"
// @Param       date_to   query string false "YYYY-MM-DD"
// @Param       limit     query int false "Default 30"
// @Param       offset    query int false "Default 0"
// @Success     200 {object} services.DailySummaryResult
// @Router      /workspaces/{workspace_id}/transactions/summary [get]
func (h *TransactionHandler) DailySummary(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)

	p := services.DailySummaryParams{
		WorkspaceID: wsID,
		Limit:       int32(queryInt(c, "limit", 30)),
		Offset:      int32(queryInt(c, "offset", 0)),
	}
	if s := c.Query("date_from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.BadRequest(c, "INVALID_INPUT", "date_from must be YYYY-MM-DD")
			return
		}
		p.DateFrom = &t
	}
	if s := c.Query("date_to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.BadRequest(c, "INVALID_INPUT", "date_to must be YYYY-MM-DD")
			return
		}
		p.DateTo = &t
	}

	result, err := h.svc.DailySummary(c.Request.Context(), p)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, result)
}

// @Summary     List transactions by date (cursor pagination)
// @Tags        transactions
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       date path string true "YYYY-MM-DD"
// @Param       cursor query string false "RFC3339Nano timestamp cursor"
// @Param       limit  query int false "Default 20"
// @Success     200 {object} services.CursorListResult
// @Router      /workspaces/{workspace_id}/transactions/by-date/{date} [get]
func (h *TransactionHandler) ListByDate(c *gin.Context) {
	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "date must be YYYY-MM-DD")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	p := services.ListByDateParams{
		WorkspaceID: wsID,
		Date:        date,
		Limit:       int32(queryInt(c, "limit", 20)),
	}

	if s := c.Query("cursor"); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			response.BadRequest(c, "INVALID_INPUT", "cursor must be RFC3339Nano timestamp")
			return
		}
		p.Cursor = &t
	}

	result, err := h.svc.ListByDate(c.Request.Context(), p)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, result)
}

type updateTransactionRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	Amount      float64    `json:"amount"  binding:"required,gt=0"`
	Description string     `json:"description"`
	Date        string     `json:"date"    binding:"required"`
}

// @Summary     Update transaction
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       transaction_id path string true "Transaction ID"
// @Param       body body updateTransactionRequest true "Transaction data"
// @Success     200 {object} services.TransactionView
// @Failure     403 {object} map[string]string
// @Router      /workspaces/{workspace_id}/transactions/{transaction_id} [put]
func (h *TransactionHandler) Update(c *gin.Context) {
	txID, err := uuid.Parse(c.Param("transaction_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid transaction_id")
		return
	}

	var req updateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "date must be YYYY-MM-DD")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	tx, err := h.svc.Update(c.Request.Context(), services.UpdateTransactionParams{
		ID:          txID,
		WorkspaceID: wsID,
		CategoryID:  req.CategoryID,
		Amount:      req.Amount,
		Description: req.Description,
		Date:        date,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, tx)
}

// @Summary     Delete transaction
// @Tags        transactions
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       transaction_id path string true "Transaction ID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Router      /workspaces/{workspace_id}/transactions/{transaction_id} [delete]
func (h *TransactionHandler) Delete(c *gin.Context) {
	txID, err := uuid.Parse(c.Param("transaction_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid transaction_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), txID, wsID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func queryInt(c *gin.Context, key string, def int) int {
	if s := c.Query(key); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}
	return def
}
