package middlewares

import (
	"errors"
	"log"
	"strings"

	"github.com/Agushim/go_wifi_billing/observability"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const AuthorizationDecisionLocal = "authorization_decision"

func RequirePermission(authorizationService services.AuthorizationService, permission string) fiber.Handler {
	return requireAuthorization(authorizationService, "permission", []string{permission}, func(decision *services.AuthorizationDecision, permissions []string) bool {
		return decision.HasPermission(permissions[0])
	})
}

func RequireAnyPermission(authorizationService services.AuthorizationService, permissions ...string) fiber.Handler {
	return requireAuthorization(authorizationService, "any", permissions, func(decision *services.AuthorizationDecision, permissions []string) bool {
		for _, permission := range permissions {
			if decision.HasPermission(permission) {
				return true
			}
		}
		return false
	})
}

func RequireAllPermissions(authorizationService services.AuthorizationService, permissions ...string) fiber.Handler {
	return requireAuthorization(authorizationService, "all", permissions, func(decision *services.AuthorizationDecision, permissions []string) bool {
		for _, permission := range permissions {
			if !decision.HasPermission(permission) {
				return false
			}
		}
		return true
	})
}

// RequireOwner checks the non-delegable role flag, never a permission key.
func RequireOwner(authorizationService services.AuthorizationService) fiber.Handler {
	return requireAuthorization(authorizationService, "owner", nil, func(decision *services.AuthorizationDecision, _ []string) bool {
		return decision.IsOwner
	})
}

func requireAuthorization(
	authorizationService services.AuthorizationService,
	requirement string,
	permissions []string,
	allowed func(*services.AuthorizationDecision, []string) bool,
) fiber.Handler {
	normalized := normalizePermissions(permissions)

	return func(c *fiber.Ctx) error {
		userID, ok := authorizationUserID(c)
		if !ok {
			recordAuthorizationStatus(c, authorizationPermissionLabel(requirement, normalized), fiber.StatusUnauthorized)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
		}
		if authorizationService == nil || (requirement != "owner" && len(normalized) == 0) {
			logAuthorizationDecision(c, userID, "", requirement, normalized, false, "invalid_configuration")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "authorization unavailable"})
		}

		decision, err := authorizationService.Resolve(c.UserContext(), userID)
		if err != nil {
			if isAuthorizationDeniedError(err) {
				logAuthorizationDecision(c, userID, "", requirement, normalized, false, "inactive_or_missing_principal")
				recordAuthorizationStatus(c, authorizationPermissionLabel(requirement, normalized), fiber.StatusForbidden)
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "forbidden"})
			}
			logAuthorizationDecision(c, userID, "", requirement, normalized, false, "resolver_error")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "authorization unavailable"})
		}

		if !allowed(decision, normalized) {
			permissionLabel := authorizationPermissionLabel(requirement, normalized)
			if baselineRollbackAllows(decision, requirement, normalized) {
				c.Locals(AuthorizationDecisionLocal, decision)
				logAuthorizationDecision(c, userID, decision.RoleKey, requirement, normalized, true, "baseline_rollback_allowed")
				return c.Next()
			}
			if !shouldEnforceAuthorization(requirement, normalized) {
				c.Locals(AuthorizationDecisionLocal, decision)
				logAuthorizationDecision(c, userID, decision.RoleKey, requirement, normalized, true, "shadow_permission_denied")
				recordAuthorizationStatus(c, permissionLabel, fiber.StatusForbidden)
				return c.Next()
			}
			logAuthorizationDecision(c, userID, decision.RoleKey, requirement, normalized, false, "permission_denied")
			recordAuthorizationStatus(c, permissionLabel, fiber.StatusForbidden)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "forbidden"})
		}

		c.Locals(AuthorizationDecisionLocal, decision)
		logAuthorizationDecision(c, userID, decision.RoleKey, requirement, normalized, true, "allowed")
		return c.Next()
	}
}

func authorizationPermissionLabel(requirement string, permissions []string) string {
	if requirement == "owner" {
		return "__owner__"
	}
	if len(permissions) == 0 {
		return "__authorization__"
	}
	return strings.Join(permissions, ",")
}

func recordAuthorizationStatus(c *fiber.Ctx, permission string, status int) {
	route := c.Path()
	if policy, ok := c.Locals(RoutePermissionPolicyLocal).(RoutePermissionPolicy); ok {
		route = policy.Path
	}
	observability.DefaultAccessControl.RecordAuthorization(c.Method(), route, permission, status)
}

func authorizationUserID(c *fiber.Ctx) (uuid.UUID, bool) {
	claims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return uuid.Nil, false
	}
	rawUserID, _ := claims["user_id"].(string)
	userID, err := uuid.Parse(strings.TrimSpace(rawUserID))
	return userID, err == nil
}

func normalizePermissions(permissions []string) []string {
	normalized := make([]string, 0, len(permissions))
	seen := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		permission = strings.TrimSpace(permission)
		if permission != "" && !seen[permission] {
			normalized = append(normalized, permission)
			seen[permission] = true
		}
	}
	return normalized
}

func isAuthorizationDeniedError(err error) bool {
	return errors.Is(err, services.ErrAuthorizationUserNotFound) ||
		errors.Is(err, services.ErrAuthorizationUserInactive) ||
		errors.Is(err, services.ErrAuthorizationRoleMissing) ||
		errors.Is(err, services.ErrAuthorizationRoleInactive)
}

func logAuthorizationDecision(c *fiber.Ctx, userID uuid.UUID, roleKey, requirement string, permissions []string, allowed bool, reason string) {
	log.Printf(
		"authorization_decision allowed=%t user_id=%s role=%s requirement=%s permissions=%s method=%s path=%s reason=%s",
		allowed,
		userID.String(),
		roleKey,
		requirement,
		strings.Join(permissions, ","),
		c.Method(),
		c.Route().Path,
		reason,
	)
}
