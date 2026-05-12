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

type workspaceService interface {
	Create(ctx context.Context, p services.CreateWorkspaceParams) (services.WorkspaceView, error)
	GetByID(ctx context.Context, id uuid.UUID) (services.WorkspaceView, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]services.WorkspaceView, error)
	Update(ctx context.Context, p services.UpdateWorkspaceParams) (services.WorkspaceView, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]services.MemberView, error)
	UpdateMemberRole(ctx context.Context, p services.UpdateMemberRoleParams) error
	RemoveMember(ctx context.Context, workspaceID, targetID, requesterID uuid.UUID) error
}

type WorkspaceHandler struct {
	svc workspaceService
}

func NewWorkspaceHandler(svc workspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

type createWorkspaceRequest struct {
	Name     string `json:"name"     binding:"required"`
	Currency string `json:"currency"`
}

// @Summary     Create workspace
// @Tags        workspaces
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body createWorkspaceRequest true "Workspace data"
// @Success     201 {object} services.WorkspaceView
// @Failure     400 {object} map[string]string
// @Router      /workspaces [post]
func (h *WorkspaceHandler) Create(c *gin.Context) {
	var req createWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	userID := middleware.UserIDFromContext(c)
	ws, err := h.svc.Create(c.Request.Context(), services.CreateWorkspaceParams{
		Name:     req.Name,
		OwnerID:  userID,
		Currency: req.Currency,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, ws)
}

// @Summary     List user workspaces
// @Tags        workspaces
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} services.WorkspaceView
// @Router      /workspaces [get]
func (h *WorkspaceHandler) List(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	workspaces, err := h.svc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, workspaces)
}

// @Summary     Get workspace
// @Tags        workspaces
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {object} services.WorkspaceView
// @Failure     404 {object} map[string]string
// @Router      /workspaces/{workspace_id} [get]
func (h *WorkspaceHandler) Get(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)
	ws, err := h.svc.GetByID(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, ws)
}

type updateWorkspaceRequest struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

// @Summary     Update workspace
// @Tags        workspaces
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       body body updateWorkspaceRequest true "Workspace data"
// @Success     200 {object} services.WorkspaceView
// @Failure     403 {object} map[string]string
// @Router      /workspaces/{workspace_id} [put]
func (h *WorkspaceHandler) Update(c *gin.Context) {
	var req updateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	ws, err := h.svc.Update(c.Request.Context(), services.UpdateWorkspaceParams{
		ID:       wsID,
		UserID:   userID,
		Name:     req.Name,
		Currency: req.Currency,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, ws)
}

// @Summary     Delete workspace
// @Tags        workspaces
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Router      /workspaces/{workspace_id} [delete]
func (h *WorkspaceHandler) Delete(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	if err := h.svc.Delete(c.Request.Context(), wsID, userID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary     List workspace members
// @Tags        workspaces
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {array} services.MemberView
// @Router      /workspaces/{workspace_id}/members [get]
func (h *WorkspaceHandler) ListMembers(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)

	members, err := h.svc.ListMembers(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, members)
}

type updateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// @Summary     Update member role
// @Tags        workspaces
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       user_id path string true "User ID"
// @Param       body body updateMemberRoleRequest true "Role"
// @Success     204
// @Failure     400,403,404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/members/{user_id}/role [put]
func (h *WorkspaceHandler) UpdateMemberRole(c *gin.Context) {
	var req updateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid user_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	if err := h.svc.UpdateMemberRole(c.Request.Context(), services.UpdateMemberRoleParams{
		WorkspaceID: wsID,
		TargetID:    targetID,
		RequesterID: userID,
		Role:        req.Role,
	}); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary     Remove workspace member
// @Tags        workspaces
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       user_id path string true "User ID"
// @Success     204
// @Failure     403,404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/members/{user_id} [delete]
func (h *WorkspaceHandler) RemoveMember(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid user_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	if err := h.svc.RemoveMember(c.Request.Context(), wsID, targetID, userID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
