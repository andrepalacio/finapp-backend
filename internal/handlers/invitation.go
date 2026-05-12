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

type invitationService interface {
	Send(ctx context.Context, p services.SendInvitationParams) (services.InvitationView, error)
	Accept(ctx context.Context, token uuid.UUID, userID uuid.UUID) (services.InvitationView, error)
	Cancel(ctx context.Context, invID uuid.UUID, workspaceID uuid.UUID, requesterID uuid.UUID) error
	ListPending(ctx context.Context, workspaceID uuid.UUID) ([]services.InvitationView, error)
}

type InvitationHandler struct {
	svc invitationService
}

func NewInvitationHandler(svc invitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

type sendInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role"`
}

// @Summary     Send workspace invitation
// @Tags        invitations
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       body body sendInvitationRequest true "Invitation data"
// @Success     201 {object} services.InvitationView
// @Failure     400,403,404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/invitations [post]
func (h *InvitationHandler) Send(c *gin.Context) {
	var req sendInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_INPUT", err.Error())
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	inv, err := h.svc.Send(c.Request.Context(), services.SendInvitationParams{
		WorkspaceID: wsID,
		Email:       req.Email,
		Role:        req.Role,
		InviterID:   userID,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, inv)
}

// @Summary     Accept workspace invitation
// @Tags        invitations
// @Produce     json
// @Security    BearerAuth
// @Param       token query string true "Invitation token (UUID)"
// @Success     200 {object} services.InvitationView
// @Failure     400,403,404 {object} map[string]string
// @Router      /invitations/accept [get]
func (h *InvitationHandler) Accept(c *gin.Context) {
	tokenStr := c.Query("token")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid token format")
		return
	}

	userID := middleware.UserIDFromContext(c)

	inv, err := h.svc.Accept(c.Request.Context(), token, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, inv)
}

// @Summary     Cancel workspace invitation
// @Tags        invitations
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       invitation_id path string true "Invitation ID"
// @Success     204
// @Failure     403,404 {object} map[string]string
// @Router      /workspaces/{workspace_id}/invitations/{invitation_id} [delete]
func (h *InvitationHandler) Cancel(c *gin.Context) {
	invID, err := uuid.Parse(c.Param("invitation_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid invitation_id")
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	if err := h.svc.Cancel(c.Request.Context(), invID, wsID, userID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary     List pending invitations
// @Tags        invitations
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200 {array} services.InvitationView
// @Router      /workspaces/{workspace_id}/invitations [get]
func (h *InvitationHandler) ListPending(c *gin.Context) {
	wsID := middleware.WorkspaceIDFromContext(c)

	invitations, err := h.svc.ListPending(c.Request.Context(), wsID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, invitations)
}
