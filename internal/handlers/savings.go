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

type savingsService interface {
	Create(ctx context.Context, p services.CreateSavingsGoalParams) (models.SavingsGoal, error)
	GetByID(ctx context.Context, id, workspaceID uuid.UUID) (models.SavingsGoal, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]models.SavingsGoal, error)
	Update(ctx context.Context, p services.UpdateSavingsGoalParams) (models.SavingsGoal, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
	GetWithProgress(ctx context.Context, id, workspaceID uuid.UUID) (services.SavingsGoalProgress, error)
	AddContribution(ctx context.Context, workspaceID uuid.UUID, p services.AddContributionParams) (models.SavingsContribution, error)
	ListContributions(ctx context.Context, goalID, workspaceID uuid.UUID) ([]models.SavingsContribution, error)
	DeleteContribution(ctx context.Context, contribID, goalID, workspaceID uuid.UUID) error
}

type SavingsHandler struct {
	svc savingsService
}

func NewSavingsHandler(svc savingsService) *SavingsHandler {
	return &SavingsHandler{svc: svc}
}

type createSavingsGoalRequest struct {
	Name         string  `json:"name" binding:"required"`
	TargetAmount float64 `json:"target_amount" binding:"required,gt=0"`
	Deadline     string  `json:"deadline"`
	Notes        string  `json:"notes"`
}

// @Summary     Create savings goal
// @Tags        savings
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       body body createSavingsGoalRequest true "Goal data"
// @Success     201 {object} models.SavingsGoal
// @Failure     400 {object} map[string]string
// @Router      /workspaces/{workspace_id}/savings-goals [post]
func (h *SavingsHandler) Create(c *gin.Context) {
	var req createSavingsGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}
	var deadline *time.Time
	if req.Deadline != "" {
		d, err := time.Parse("2006-01-02", req.Deadline)
		if err != nil {
			response.BadRequest(c, "INVALID_DATE", "deadline must be YYYY-MM-DD")
			return
		}
		deadline = &d
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	goal, err := h.svc.Create(c.Request.Context(), services.CreateSavingsGoalParams{
		WorkspaceID:  wsID,
		Name:         req.Name,
		TargetAmount: req.TargetAmount,
		Deadline:     deadline,
		Notes:        req.Notes,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, goal)
}

// @Summary     List savings goals
// @Tags        savings
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {array} models.SavingsGoal
// @Router      /workspaces/{workspace_id}/savings-goals [get]
func (h *SavingsHandler) List(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)
	goals, err := h.svc.List(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, goals)
}

// @Summary     Get savings goal with progress
// @Tags        savings
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       goal_id path string true "Goal ID"
// @Success     200 {object} services.SavingsGoalProgress
// @Router      /workspaces/{workspace_id}/savings-goals/{goal_id} [get]
func (h *SavingsHandler) Get(c *gin.Context) {
	goalID, err := uuid.Parse(c.Param("goal_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid goal_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	prog, err := h.svc.GetWithProgress(c.Request.Context(), goalID, wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, prog)
}

type updateSavingsGoalRequest struct {
	Name         string  `json:"name" binding:"required"`
	TargetAmount float64 `json:"target_amount" binding:"required,gt=0"`
	Deadline     string  `json:"deadline"`
	Notes        string  `json:"notes"`
}

// @Summary     Update savings goal
// @Tags        savings
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       goal_id path string true "Goal ID"
// @Param       body body updateSavingsGoalRequest true "Goal data"
// @Success     200 {object} models.SavingsGoal
// @Router      /workspaces/{workspace_id}/savings-goals/{goal_id} [put]
func (h *SavingsHandler) Update(c *gin.Context) {
	goalID, err := uuid.Parse(c.Param("goal_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid goal_id")
		return
	}
	var req updateSavingsGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}
	var deadline *time.Time
	if req.Deadline != "" {
		d, err := time.Parse("2006-01-02", req.Deadline)
		if err != nil {
			response.BadRequest(c, "INVALID_DATE", "deadline must be YYYY-MM-DD")
			return
		}
		deadline = &d
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	goal, err := h.svc.Update(c.Request.Context(), services.UpdateSavingsGoalParams{
		ID:           goalID,
		WorkspaceID:  wsID,
		Name:         req.Name,
		TargetAmount: req.TargetAmount,
		Deadline:     deadline,
		Notes:        req.Notes,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, goal)
}

// @Summary     Delete savings goal
// @Tags        savings
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       goal_id path string true "Goal ID"
// @Success     204
// @Router      /workspaces/{workspace_id}/savings-goals/{goal_id} [delete]
func (h *SavingsHandler) Delete(c *gin.Context) {
	goalID, err := uuid.Parse(c.Param("goal_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid goal_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), goalID, wsID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(204)
}

type addContributionRequest struct {
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	ContributedAt string  `json:"contributed_at" binding:"required"`
	Notes         string  `json:"notes"`
}

// @Summary     Add contribution to savings goal
// @Tags        savings
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       goal_id path string true "Goal ID"
// @Param       body body addContributionRequest true "Contribution data"
// @Success     201 {object} models.SavingsContribution
// @Router      /workspaces/{workspace_id}/savings-goals/{goal_id}/contributions [post]
func (h *SavingsHandler) AddContribution(c *gin.Context) {
	goalID, err := uuid.Parse(c.Param("goal_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid goal_id")
		return
	}
	var req addContributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_BODY", err.Error())
		return
	}
	contributedAt, err := time.Parse("2006-01-02", req.ContributedAt)
	if err != nil {
		response.BadRequest(c, "INVALID_DATE", "contributed_at must be YYYY-MM-DD")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	contrib, err := h.svc.AddContribution(c.Request.Context(), wsID, services.AddContributionParams{
		GoalID:        goalID,
		Amount:        req.Amount,
		ContributedAt: contributedAt,
		Notes:         req.Notes,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, contrib)
}

// @Summary     List contributions for savings goal
// @Tags        savings
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       goal_id path string true "Goal ID"
// @Success     200 {array} models.SavingsContribution
// @Router      /workspaces/{workspace_id}/savings-goals/{goal_id}/contributions [get]
func (h *SavingsHandler) ListContributions(c *gin.Context) {
	goalID, err := uuid.Parse(c.Param("goal_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid goal_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	contribs, err := h.svc.ListContributions(c.Request.Context(), goalID, wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, contribs)
}

// @Summary     Delete contribution
// @Tags        savings
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       goal_id path string true "Goal ID"
// @Param       contribution_id path string true "Contribution ID"
// @Success     204
// @Router      /workspaces/{workspace_id}/savings-goals/{goal_id}/contributions/{contribution_id} [delete]
func (h *SavingsHandler) DeleteContribution(c *gin.Context) {
	goalID, err := uuid.Parse(c.Param("goal_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid goal_id")
		return
	}
	contribID, err := uuid.Parse(c.Param("contribution_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "invalid contribution_id")
		return
	}
	wsID := middleware.WorkspaceIDFromContext(c)
	if err := h.svc.DeleteContribution(c.Request.Context(), contribID, goalID, wsID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(204)
}
