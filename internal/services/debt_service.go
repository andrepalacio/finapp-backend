package services

import (
	"context"
	"math"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

var validRateTypes = map[string]bool{
	"effective_annual": true,
	"nominal_annual":   true,
	"monthly":          true,
}

var validInsuranceTypes = map[string]bool{
	"":              true,
	"fixed_monthly": true,
	"on_balance":    true,
}

type DebtRepository interface {
	Create(ctx context.Context, p repositories.CreateDebtParams) (models.Debt, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Debt, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error)
	Update(ctx context.Context, p repositories.UpdateDebtParams) (models.Debt, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
	CreatePayment(ctx context.Context, p repositories.CreateDebtPaymentParams) (models.DebtPayment, error)
	GetPayment(ctx context.Context, id uuid.UUID) (models.DebtPayment, error)
	ListPayments(ctx context.Context, debtID uuid.UUID) ([]models.DebtPayment, error)
	UpdatePayment(ctx context.Context, p repositories.UpdateDebtPaymentParams) (models.DebtPayment, error)
	DeletePayment(ctx context.Context, id, debtID uuid.UUID) error
}

type DebtService struct {
	repo DebtRepository
}

func NewDebtService(repo DebtRepository) *DebtService {
	return &DebtService{repo: repo}
}

type CreateDebtParams struct {
	WorkspaceID      uuid.UUID
	Name             string
	Lender           string
	Principal        float64
	Rate             float64
	RateType         string
	Installments     int32
	FirstPaymentDate time.Time
	Notes            string
	InsuranceRate    float64
	InsuranceType    string
}

func (s *DebtService) Create(ctx context.Context, p CreateDebtParams) (models.Debt, error) {
	if err := validateDebtParams(p.Name, p.Principal, p.Rate, p.RateType, p.Installments, p.FirstPaymentDate, p.InsuranceRate, p.InsuranceType); err != nil {
		return models.Debt{}, err
	}
	return s.repo.Create(ctx, repositories.CreateDebtParams{
		WorkspaceID:      p.WorkspaceID,
		Name:             p.Name,
		Lender:           p.Lender,
		Principal:        p.Principal,
		Rate:             p.Rate,
		RateType:         p.RateType,
		Installments:     p.Installments,
		FirstPaymentDate: p.FirstPaymentDate,
		Notes:            p.Notes,
		InsuranceRate:    p.InsuranceRate,
		InsuranceType:    p.InsuranceType,
	})
}

func (s *DebtService) GetByID(ctx context.Context, id, workspaceID uuid.UUID) (models.Debt, error) {
	debt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return models.Debt{}, err
	}
	if debt.WorkspaceID != workspaceID {
		return models.Debt{}, apperror.ErrNotFound
	}
	return debt, nil
}

func (s *DebtService) List(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error) {
	return s.repo.List(ctx, workspaceID)
}

type UpdateDebtParams struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	Name             string
	Lender           string
	Principal        float64
	Rate             float64
	RateType         string
	Installments     int32
	FirstPaymentDate time.Time
	Notes            string
	InsuranceRate    float64
	InsuranceType    string
}

func (s *DebtService) Update(ctx context.Context, p UpdateDebtParams) (models.Debt, error) {
	if err := validateDebtParams(p.Name, p.Principal, p.Rate, p.RateType, p.Installments, p.FirstPaymentDate, p.InsuranceRate, p.InsuranceType); err != nil {
		return models.Debt{}, err
	}
	return s.repo.Update(ctx, repositories.UpdateDebtParams{
		ID:               p.ID,
		WorkspaceID:      p.WorkspaceID,
		Name:             p.Name,
		Lender:           p.Lender,
		Principal:        p.Principal,
		Rate:             p.Rate,
		RateType:         p.RateType,
		Installments:     p.Installments,
		FirstPaymentDate: p.FirstPaymentDate,
		Notes:            p.Notes,
		InsuranceRate:    p.InsuranceRate,
		InsuranceType:    p.InsuranceType,
	})
}

func (s *DebtService) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	debt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if debt.WorkspaceID != workspaceID {
		return apperror.ErrNotFound
	}
	return s.repo.Delete(ctx, id, workspaceID)
}

func (s *DebtService) GetSchedule(ctx context.Context, id, workspaceID uuid.UUID) ([]models.DebtScheduleInstallment, error) {
	debt, err := s.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, err
	}
	payments, err := s.repo.ListPayments(ctx, id)
	if err != nil {
		return nil, err
	}
	return computeSchedule(debt, payments), nil
}

type RecordPaymentParams struct {
	DebtID uuid.UUID
	Period int32
	Amount float64
	PaidAt time.Time
	Notes  string
}

func (s *DebtService) RecordPayment(ctx context.Context, workspaceID uuid.UUID, p RecordPaymentParams) (models.DebtPayment, error) {
	debt, err := s.GetByID(ctx, p.DebtID, workspaceID)
	if err != nil {
		return models.DebtPayment{}, err
	}
	if p.Period < 1 || p.Period > debt.Installments {
		return models.DebtPayment{}, apperror.WithMessage(apperror.ErrInvalidInput, "period out of range")
	}
	if p.Amount <= 0 {
		return models.DebtPayment{}, apperror.WithMessage(apperror.ErrInvalidInput, "amount must be positive")
	}
	if p.PaidAt.IsZero() {
		return models.DebtPayment{}, apperror.WithMessage(apperror.ErrInvalidInput, "paid_at is required")
	}
	return s.repo.CreatePayment(ctx, repositories.CreateDebtPaymentParams{
		DebtID: p.DebtID,
		Period: p.Period,
		Amount: p.Amount,
		PaidAt: p.PaidAt,
		Notes:  p.Notes,
	})
}

func (s *DebtService) ListPayments(ctx context.Context, debtID, workspaceID uuid.UUID) ([]models.DebtPayment, error) {
	if _, err := s.GetByID(ctx, debtID, workspaceID); err != nil {
		return nil, err
	}
	return s.repo.ListPayments(ctx, debtID)
}

type UpdatePaymentParams struct {
	PaymentID uuid.UUID
	DebtID    uuid.UUID
	Amount    float64
	PaidAt    time.Time
	Notes     string
}

func (s *DebtService) UpdatePayment(ctx context.Context, workspaceID uuid.UUID, p UpdatePaymentParams) (models.DebtPayment, error) {
	if _, err := s.GetByID(ctx, p.DebtID, workspaceID); err != nil {
		return models.DebtPayment{}, err
	}
	payment, err := s.repo.GetPayment(ctx, p.PaymentID)
	if err != nil {
		return models.DebtPayment{}, err
	}
	if payment.DebtID != p.DebtID {
		return models.DebtPayment{}, apperror.ErrNotFound
	}
	if p.Amount <= 0 {
		return models.DebtPayment{}, apperror.WithMessage(apperror.ErrInvalidInput, "amount must be positive")
	}
	if p.PaidAt.IsZero() {
		return models.DebtPayment{}, apperror.WithMessage(apperror.ErrInvalidInput, "paid_at is required")
	}
	return s.repo.UpdatePayment(ctx, repositories.UpdateDebtPaymentParams{
		ID:     p.PaymentID,
		Amount: p.Amount,
		PaidAt: p.PaidAt,
		Notes:  p.Notes,
	})
}

func (s *DebtService) DeletePayment(ctx context.Context, paymentID, debtID, workspaceID uuid.UUID) error {
	if _, err := s.GetByID(ctx, debtID, workspaceID); err != nil {
		return err
	}
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.DebtID != debtID {
		return apperror.ErrNotFound
	}
	return s.repo.DeletePayment(ctx, paymentID, debtID)
}

func validateDebtParams(name string, principal, rate float64, rateType string, installments int32, firstPaymentDate time.Time, insuranceRate float64, insuranceType string) error {
	if name == "" {
		return apperror.WithMessage(apperror.ErrInvalidInput, "name is required")
	}
	if principal <= 0 {
		return apperror.WithMessage(apperror.ErrInvalidInput, "principal must be positive")
	}
	if rate < 0 {
		return apperror.WithMessage(apperror.ErrInvalidInput, "rate must be non-negative")
	}
	if !validRateTypes[rateType] {
		return apperror.WithMessage(apperror.ErrInvalidInput, "rate_type must be effective_annual, nominal_annual, or monthly")
	}
	if installments < 1 {
		return apperror.WithMessage(apperror.ErrInvalidInput, "installments must be at least 1")
	}
	if firstPaymentDate.IsZero() {
		return apperror.WithMessage(apperror.ErrInvalidInput, "first_payment_date is required")
	}
	if !validInsuranceTypes[insuranceType] {
		return apperror.WithMessage(apperror.ErrInvalidInput, "insurance_type must be fixed_monthly or on_balance")
	}
	if insuranceRate < 0 {
		return apperror.WithMessage(apperror.ErrInvalidInput, "insurance_rate must be non-negative")
	}
	return nil
}

func monthlyRate(rate float64, rateType string) float64 {
	r := rate / 100
	switch rateType {
	case "effective_annual":
		return math.Pow(1+r, 1.0/12) - 1
	case "nominal_annual":
		return r / 12
	default: // monthly
		return r
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func insuranceAmount(insuranceType string, insuranceRate, balance float64) float64 {
	switch insuranceType {
	case "fixed_monthly":
		return insuranceRate
	case "on_balance":
		return round2(balance * insuranceRate / 100)
	default:
		return 0
	}
}

func computeSchedule(debt models.Debt, payments []models.DebtPayment) []models.DebtScheduleInstallment {
	r := monthlyRate(debt.Rate, debt.RateType)
	n := int(debt.Installments)
	P := debt.Principal

	var M float64
	if r == 0 {
		M = P / float64(n)
	} else {
		factor := math.Pow(1+r, float64(n))
		M = P * r * factor / (factor - 1)
	}

	paidMap := make(map[int32]models.DebtPayment, len(payments))
	for _, p := range payments {
		paidMap[p.Period] = p
	}

	schedule := make([]models.DebtScheduleInstallment, n)
	balance := P
	for i := 0; i < n; i++ {
		period := int32(i + 1)
		interest := balance * r
		principal := M - interest
		if i == n-1 {
			// absorb rounding residual in last installment
			principal = balance
		}
		balance = round2(balance - principal)
		if balance < 0 {
			balance = 0
		}

		insurance := insuranceAmount(debt.InsuranceType, debt.InsuranceRate, balance)

		inst := models.DebtScheduleInstallment{
			Period:    period,
			DueDate:   debt.FirstPaymentDate.AddDate(0, i, 0),
			Payment:   round2(principal + interest),
			Principal: round2(principal),
			Interest:  round2(interest),
			Insurance: insurance,
			Total:     round2(principal + interest + insurance),
			Balance:   balance,
			Status:    "pending",
		}

		if p, ok := paidMap[period]; ok {
			inst.Status = "paid"
			t := p.PaidAt
			inst.PaidAt = &t
			amt := p.Amount
			inst.PaidAmount = &amt
		}

		schedule[i] = inst
	}

	return schedule
}
