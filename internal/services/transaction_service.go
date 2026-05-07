package services

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

type TransactionRepository interface {
	Create(ctx context.Context, p repositories.CreateTransactionParams) (models.Transaction, error)
	CreateTransfer(ctx context.Context, p repositories.CreateTransferParams) (models.Transaction, models.Transaction, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Transaction, error)
	List(ctx context.Context, p repositories.ListTransactionsParams) ([]models.Transaction, error)
	Count(ctx context.Context, p repositories.ListTransactionsParams) (int64, error)
	DailySummary(ctx context.Context, p repositories.DailySummaryParams) ([]models.DailySummary, error)
	ListByDateCursor(ctx context.Context, p repositories.ListByDateCursorParams) ([]models.Transaction, error)
	Update(ctx context.Context, p repositories.UpdateTransactionParams) (models.Transaction, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
}

type TransactionService struct {
	repo TransactionRepository
}

func NewTransactionService(repo TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

type TransactionView struct {
	ID                uuid.UUID  `json:"id"`
	WorkspaceID       uuid.UUID  `json:"workspace_id"`
	UserID            uuid.UUID  `json:"user_id"`
	CategoryID        *uuid.UUID `json:"category_id,omitempty"`
	TransferID        *uuid.UUID `json:"transfer_id,omitempty"`
	Type              string     `json:"type"`
	TransferDirection string     `json:"transfer_direction,omitempty"`
	Amount            float64    `json:"amount"`
	Description       string     `json:"description,omitempty"`
	Date              string     `json:"date"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

func toTransactionView(t models.Transaction) TransactionView {
	return TransactionView{
		ID:                t.ID,
		WorkspaceID:       t.WorkspaceID,
		UserID:            t.UserID,
		CategoryID:        t.CategoryID,
		TransferID:        t.TransferID,
		Type:              t.Type,
		TransferDirection: t.TransferDirection,
		Amount:            t.Amount,
		Description:       t.Description,
		Date:              t.Date.Format("2006-01-02"),
		CreatedAt:         t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         t.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type DailySummaryView struct {
	Date             string  `json:"date"`
	TotalExpense     float64 `json:"total_expense"`
	TotalIncome      float64 `json:"total_income"`
	TotalTransferOut float64 `json:"total_transfer_out"`
	TotalTransferIn  float64 `json:"total_transfer_in"`
	TransactionCount int32   `json:"transaction_count"`
}

type CreateTransactionParams struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	CategoryID  *uuid.UUID
	Type        string
	Amount      float64
	Description string
	Date        time.Time
}

var validTransactionTypes = map[string]bool{"expense": true, "income": true}

func (s *TransactionService) Create(ctx context.Context, p CreateTransactionParams) (TransactionView, error) {
	if !validTransactionTypes[p.Type] {
		return TransactionView{}, apperror.ErrInvalidInput
	}
	if p.Amount <= 0 {
		return TransactionView{}, apperror.ErrInvalidInput
	}
	if p.Date.IsZero() {
		return TransactionView{}, apperror.ErrInvalidInput
	}

	tx, err := s.repo.Create(ctx, repositories.CreateTransactionParams{
		WorkspaceID: p.WorkspaceID,
		UserID:      p.UserID,
		CategoryID:  p.CategoryID,
		Type:        p.Type,
		Amount:      p.Amount,
		Description: p.Description,
		Date:        p.Date,
	})
	if err != nil {
		return TransactionView{}, err
	}
	return toTransactionView(tx), nil
}

type CreateTransferParams struct {
	FromWorkspaceID uuid.UUID
	ToWorkspaceID   uuid.UUID
	UserID          uuid.UUID
	Amount          float64
	Description     string
	Note            string
	Date            time.Time
}

type TransferResult struct {
	Out TransactionView `json:"out"`
	In  TransactionView `json:"in"`
}

func (s *TransactionService) CreateTransfer(ctx context.Context, p CreateTransferParams) (TransferResult, error) {
	if p.Amount <= 0 {
		return TransferResult{}, apperror.ErrInvalidInput
	}
	if p.Date.IsZero() {
		return TransferResult{}, apperror.ErrInvalidInput
	}

	out, in, err := s.repo.CreateTransfer(ctx, repositories.CreateTransferParams{
		FromWorkspaceID: p.FromWorkspaceID,
		ToWorkspaceID:   p.ToWorkspaceID,
		UserID:          p.UserID,
		Amount:          p.Amount,
		Description:     p.Description,
		Note:            p.Note,
		Date:            p.Date,
	})
	if err != nil {
		return TransferResult{}, err
	}
	return TransferResult{Out: toTransactionView(out), In: toTransactionView(in)}, nil
}

func (s *TransactionService) GetByID(ctx context.Context, id uuid.UUID) (TransactionView, error) {
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return TransactionView{}, err
	}
	return toTransactionView(tx), nil
}

type ListTransactionsParams struct {
	WorkspaceID uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	Type        string
	CategoryID  *uuid.UUID
	Limit       int32
	Offset      int32
}

type TransactionListResult struct {
	Items []TransactionView `json:"items"`
	Total int64             `json:"total"`
}

func (s *TransactionService) List(ctx context.Context, p ListTransactionsParams) (TransactionListResult, error) {
	if p.Limit <= 0 {
		p.Limit = 20
	}

	repoP := repositories.ListTransactionsParams{
		WorkspaceID: p.WorkspaceID,
		DateFrom:    p.DateFrom,
		DateTo:      p.DateTo,
		Type:        p.Type,
		CategoryID:  p.CategoryID,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}

	txs, err := s.repo.List(ctx, repoP)
	if err != nil {
		return TransactionListResult{}, err
	}
	total, err := s.repo.Count(ctx, repoP)
	if err != nil {
		return TransactionListResult{}, err
	}

	items := make([]TransactionView, len(txs))
	for i, tx := range txs {
		items[i] = toTransactionView(tx)
	}
	return TransactionListResult{Items: items, Total: total}, nil
}

type DailySummaryParams struct {
	WorkspaceID uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	Limit       int32
	Offset      int32
}

type DailySummaryResult struct {
	Items []DailySummaryView `json:"items"`
}

func (s *TransactionService) DailySummary(ctx context.Context, p DailySummaryParams) (DailySummaryResult, error) {
	if p.Limit <= 0 {
		p.Limit = 30
	}
	rows, err := s.repo.DailySummary(ctx, repositories.DailySummaryParams{
		WorkspaceID: p.WorkspaceID,
		DateFrom:    p.DateFrom,
		DateTo:      p.DateTo,
		Limit:       p.Limit,
		Offset:      p.Offset,
	})
	if err != nil {
		return DailySummaryResult{}, err
	}
	items := make([]DailySummaryView, len(rows))
	for i, row := range rows {
		items[i] = DailySummaryView{
			Date:             row.Date.Format("2006-01-02"),
			TotalExpense:     row.TotalExpense,
			TotalIncome:      row.TotalIncome,
			TotalTransferOut: row.TotalTransferOut,
			TotalTransferIn:  row.TotalTransferIn,
			TransactionCount: row.TransactionCount,
		}
	}
	return DailySummaryResult{Items: items}, nil
}

type ListByDateParams struct {
	WorkspaceID uuid.UUID
	Date        time.Time
	Cursor      *time.Time
	Limit       int32
}

type CursorListResult struct {
	Items      []TransactionView `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

func (s *TransactionService) ListByDate(ctx context.Context, p ListByDateParams) (CursorListResult, error) {
	if p.Limit <= 0 {
		p.Limit = 20
	}

	txs, err := s.repo.ListByDateCursor(ctx, repositories.ListByDateCursorParams{
		WorkspaceID: p.WorkspaceID,
		Date:        p.Date,
		Cursor:      p.Cursor,
		Limit:       p.Limit,
	})
	if err != nil {
		return CursorListResult{}, err
	}

	items := make([]TransactionView, len(txs))
	for i, tx := range txs {
		items[i] = toTransactionView(tx)
	}

	var nextCursor *string
	if int32(len(txs)) == p.Limit {
		last := txs[len(txs)-1].CreatedAt.UTC().Format(time.RFC3339Nano)
		nextCursor = &last
	}
	return CursorListResult{Items: items, NextCursor: nextCursor}, nil
}

type UpdateTransactionParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	CategoryID  *uuid.UUID
	Amount      float64
	Description string
	Date        time.Time
}

func (s *TransactionService) Update(ctx context.Context, p UpdateTransactionParams) (TransactionView, error) {
	existing, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return TransactionView{}, err
	}
	if existing.WorkspaceID != p.WorkspaceID {
		return TransactionView{}, apperror.ErrForbidden
	}
	if existing.Type == "transfer" {
		return TransactionView{}, apperror.ErrForbidden
	}
	if p.Amount <= 0 {
		return TransactionView{}, apperror.ErrInvalidInput
	}
	if p.Date.IsZero() {
		return TransactionView{}, apperror.ErrInvalidInput
	}

	tx, err := s.repo.Update(ctx, repositories.UpdateTransactionParams{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Amount:      p.Amount,
		Description: p.Description,
		Date:        p.Date,
	})
	if err != nil {
		return TransactionView{}, err
	}
	return toTransactionView(tx), nil
}

func (s *TransactionService) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.WorkspaceID != workspaceID {
		return apperror.ErrForbidden
	}
	return s.repo.Delete(ctx, id, workspaceID)
}
