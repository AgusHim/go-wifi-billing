package controllers

import (
	"strings"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func authorizationDecision(ctx *fiber.Ctx) (*services.AuthorizationDecision, bool) {
	decision, ok := ctx.Locals(middlewares.AuthorizationDecisionLocal).(*services.AuthorizationDecision)
	return decision, ok && decision != nil
}

func authenticatedUserID(ctx *fiber.Ctx) (uuid.UUID, bool) {
	decision, ok := authorizationDecision(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return decision.UserID, decision.UserID != uuid.Nil
}

func authenticatedCustomerUserID(ctx *fiber.Ctx) (uuid.UUID, bool) {
	decision, ok := authorizationDecision(ctx)
	if !ok || decision.RoleKey != "customer" {
		return uuid.Nil, false
	}
	return decision.UserID, decision.UserID != uuid.Nil
}

// hasGlobalOperationalDataScope is a data policy, not a feature permission.
// Operational roles are intentionally restricted to records assigned to them.
func hasGlobalOperationalDataScope(ctx *fiber.Ctx) bool {
	decision, ok := authorizationDecision(ctx)
	return ok && (decision.IsOwner || decision.RoleKey == "admin")
}

func scopedAdminID(ctx *fiber.Ctx, requested string) string {
	if hasGlobalOperationalDataScope(ctx) {
		return strings.TrimSpace(requested)
	}
	if userID, ok := authenticatedUserID(ctx); ok {
		return userID.String()
	}
	return strings.TrimSpace(requested)
}
