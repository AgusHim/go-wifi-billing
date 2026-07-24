package middlewares

import (
	"os"
	"strings"

	"github.com/Agushim/go_wifi_billing/services"
)

const (
	authorizationModeShadow  = "shadow"
	authorizationModeWarning = "warning"
	authorizationModeEnforce = "enforce"
)

func shouldEnforceAuthorization(requirement string, permissions []string) bool {
	if requirement == "owner" {
		return true
	}
	switch authorizationEnforcementMode() {
	case authorizationModeShadow:
		return false
	case authorizationModeWarning:
		modules := configuredEnforcedModules()
		for _, permission := range permissions {
			module := permissionModule(permission)
			if modules[module] {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func baselineRollbackAllows(decision *services.AuthorizationDecision, requirement string, permissions []string) bool {
	if decision == nil || decision.IsOwner || requirement == "owner" || !authorizationBaselineRollbackEnabled() {
		return false
	}
	switch requirement {
	case "permission":
		return len(permissions) == 1 && decision.BaselinePermissions[permissions[0]]
	case "any":
		for _, permission := range permissions {
			if decision.BaselinePermissions[permission] {
				return true
			}
		}
	case "all":
		if len(permissions) == 0 {
			return false
		}
		for _, permission := range permissions {
			if !decision.BaselinePermissions[permission] {
				return false
			}
		}
		return true
	}
	return false
}

func authorizationEnforcementMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AUTHZ_ENFORCEMENT_MODE"))) {
	case authorizationModeShadow:
		return authorizationModeShadow
	case authorizationModeWarning:
		return authorizationModeWarning
	default:
		return authorizationModeEnforce
	}
}

func configuredEnforcedModules() map[string]bool {
	result := make(map[string]bool)
	for _, module := range strings.Split(os.Getenv("AUTHZ_ENFORCED_MODULES"), ",") {
		module = strings.ToLower(strings.TrimSpace(module))
		if module != "" {
			result[module] = true
		}
	}
	return result
}

func authorizationBaselineRollbackEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("AUTHZ_ROLLBACK_BASELINE")))
	return value == "1" || value == "true" || value == "yes"
}

func permissionModule(permission string) string {
	permission = strings.TrimSpace(permission)
	if index := strings.Index(permission, "."); index > 0 {
		return permission[:index]
	}
	return permission
}
