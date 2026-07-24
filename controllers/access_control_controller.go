package controllers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/observability"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessControlController struct {
	service              services.AccessControlService
	authorizationService services.AuthorizationService
}

func NewAccessControlController(service services.AccessControlService, authorizationService services.AuthorizationService) *AccessControlController {
	return &AccessControlController{service: service, authorizationService: authorizationService}
}

func (ctrl *AccessControlController) RegisterRoutes(router fiber.Router) {
	group := router.Group(
		"/admin_api/access-control",
		middlewares.UserProtected(),
		middlewares.RequireOwner(ctrl.authorizationService),
	)
	group.Get("/permissions", ctrl.ListPermissions)
	group.Get("/roles", ctrl.ListRoles)
	group.Get("/roles/:id", ctrl.GetRole)
	group.Post("/roles", ctrl.CreateRole)
	group.Put("/roles/:id", ctrl.UpdateRole)
	group.Put("/roles/:id/permissions", ctrl.UpdateRolePermissions)
	group.Delete("/roles/:id", ctrl.DeleteRole)
	group.Get("/users", ctrl.ListUsers)
	group.Get("/users/:id", ctrl.GetUserAccess)
	group.Put("/users/:id", ctrl.UpdateUserAccess)
	group.Delete("/users/:id/overrides", ctrl.ResetUserOverrides)
	group.Get("/audit-logs", ctrl.ListAuditLogs)
	group.Get("/audit-logs/export", ctrl.ExportAuditLogs)
	group.Get("/metrics", ctrl.GetMetrics)
}

func (ctrl *AccessControlController) ListPermissions(c *fiber.Ctx) error {
	groups, err := ctrl.service.ListPermissions(c.UserContext())
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": groups, "message": "Permission catalog retrieved"})
}

func (ctrl *AccessControlController) ListRoles(c *fiber.Ctx) error {
	page, limit := accessPagination(c)
	records, total, err := ctrl.service.ListRoles(c.UserContext(), page, limit, c.Query("search"))
	if err != nil {
		return accessControlError(c, err)
	}
	return accessPaginated(c, records, page, limit, total, "Roles retrieved")
}

func (ctrl *AccessControlController) GetRole(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	detail, err := ctrl.service.GetRole(c.UserContext(), roleID)
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": detail, "message": "Role retrieved"})
}

func (ctrl *AccessControlController) CreateRole(c *fiber.Ctx) error {
	var input dto.CreateRoleDTO
	if err := c.BodyParser(&input); err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	actorID, ok := accessActorID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	result, err := ctrl.service.CreateRole(c.UserContext(), actorID, input, accessMetadata(c))
	if err != nil {
		return accessControlError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": result, "message": "Role created"})
}

func (ctrl *AccessControlController) UpdateRole(c *fiber.Ctx) error {
	actorID, roleID, ok := accessActorAndTarget(c)
	if !ok {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	var input dto.UpdateRoleDTO
	if err := c.BodyParser(&input); err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	result, err := ctrl.service.UpdateRole(c.UserContext(), actorID, roleID, input, accessMetadata(c))
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": result, "message": "Role updated"})
}

func (ctrl *AccessControlController) UpdateRolePermissions(c *fiber.Ctx) error {
	actorID, roleID, ok := accessActorAndTarget(c)
	if !ok {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	var input dto.UpdateRolePermissionsDTO
	if err := c.BodyParser(&input); err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	result, err := ctrl.service.UpdateRolePermissions(c.UserContext(), actorID, roleID, input, accessMetadata(c))
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": result, "message": "Role permissions updated"})
}

func (ctrl *AccessControlController) DeleteRole(c *fiber.Ctx) error {
	actorID, roleID, ok := accessActorAndTarget(c)
	if !ok {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	if err := ctrl.service.DeleteRole(c.UserContext(), actorID, roleID, c.Query("reason"), accessMetadata(c)); err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Role deleted"})
}

func (ctrl *AccessControlController) ListUsers(c *fiber.Ctx) error {
	page, limit := accessPagination(c)
	var roleID *uuid.UUID
	if rawRoleID := strings.TrimSpace(c.Query("role_id")); rawRoleID != "" {
		parsed, err := uuid.Parse(rawRoleID)
		if err != nil {
			return accessControlError(c, services.ErrAccessControlInvalidInput)
		}
		roleID = &parsed
	}
	records, total, err := ctrl.service.ListUsers(c.UserContext(), page, limit, c.Query("search"), roleID)
	if err != nil {
		return accessControlError(c, err)
	}
	return accessPaginated(c, records, page, limit, total, "Users retrieved")
}

func (ctrl *AccessControlController) GetUserAccess(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	detail, err := ctrl.service.GetUserAccess(c.UserContext(), userID)
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": detail, "message": "User access retrieved"})
}

func (ctrl *AccessControlController) UpdateUserAccess(c *fiber.Ctx) error {
	actorID, userID, ok := accessActorAndTarget(c)
	if !ok {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	var input dto.UpdateUserAccessDTO
	if err := c.BodyParser(&input); err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	detail, err := ctrl.service.UpdateUserAccess(c.UserContext(), actorID, userID, input, accessMetadata(c))
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": detail, "message": "User access updated"})
}

func (ctrl *AccessControlController) ResetUserOverrides(c *fiber.Ctx) error {
	actorID, userID, ok := accessActorAndTarget(c)
	if !ok {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	var input dto.ResetUserOverridesDTO
	if err := c.BodyParser(&input); err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	detail, err := ctrl.service.ResetUserOverrides(c.UserContext(), actorID, userID, input, accessMetadata(c))
	if err != nil {
		return accessControlError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": detail, "message": "User overrides reset"})
}

func (ctrl *AccessControlController) ListAuditLogs(c *fiber.Ctx) error {
	page, limit := accessPagination(c)
	filter, err := accessAuditFilter(c)
	if err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	logs, total, err := ctrl.service.ListAuditLogs(c.UserContext(), page, limit, filter)
	if err != nil {
		return accessControlError(c, err)
	}
	return accessPaginated(c, logs, page, limit, total, "Audit logs retrieved")
}

func (ctrl *AccessControlController) ExportAuditLogs(c *fiber.Ctx) error {
	filter, err := accessAuditFilter(c)
	if err != nil {
		return accessControlError(c, services.ErrAccessControlInvalidInput)
	}
	content, err := ctrl.service.ExportAuditLogs(c.UserContext(), filter)
	if err != nil {
		return accessControlError(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="access-audit-logs.csv"`)
	return c.Send(content)
}

func (ctrl *AccessControlController) GetMetrics(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data":    observability.DefaultAccessControl.Snapshot(),
		"message": "Access-control metrics retrieved",
	})
}

func accessAuditFilter(c *fiber.Ctx) (repositories.AccessAuditFilter, error) {
	filter := repositories.AccessAuditFilter{
		TargetType: strings.TrimSpace(c.Query("target_type")),
		Action:     strings.TrimSpace(c.Query("action")),
	}
	var err error
	if filter.ActorUserID, err = accessOptionalUUIDQuery(c, "actor_user_id"); err != nil {
		return filter, err
	}
	if filter.TargetID, err = accessOptionalUUIDQuery(c, "target_id"); err != nil {
		return filter, err
	}
	if filter.DateFrom, err = accessOptionalTimeQuery(c, "date_from", false); err != nil {
		return filter, err
	}
	if filter.DateTo, err = accessOptionalTimeQuery(c, "date_to", true); err != nil {
		return filter, err
	}
	return filter, nil
}

func accessControlError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, services.ErrAccessControlNotFound):
		status, message = fiber.StatusNotFound, "resource not found"
	case errors.Is(err, services.ErrAccessControlConflict):
		status, message = fiber.StatusConflict, err.Error()
	case errors.Is(err, services.ErrAccessControlLastOwner):
		status, message = fiber.StatusConflict, err.Error()
	case errors.Is(err, services.ErrAccessControlInvalidInput),
		errors.Is(err, services.ErrAccessControlSystemRole),
		errors.Is(err, services.ErrAccessControlCriticalReason):
		status, message = fiber.StatusBadRequest, err.Error()
	}
	return c.Status(status).JSON(fiber.Map{"success": false, "message": message})
}

func accessActorID(c *fiber.Ctx) (uuid.UUID, bool) {
	claims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return uuid.Nil, false
	}
	raw, _ := claims["user_id"].(string)
	actorID, err := uuid.Parse(strings.TrimSpace(raw))
	return actorID, err == nil
}

func accessActorAndTarget(c *fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	actorID, ok := accessActorID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	targetID, err := uuid.Parse(c.Params("id"))
	return actorID, targetID, err == nil
}

func accessMetadata(c *fiber.Ctx) dto.AccessChangeMetadata {
	return dto.AccessChangeMetadata{IPAddress: c.IP(), UserAgent: c.Get("User-Agent"), RequestID: c.Get("X-Request-ID")}
}

func accessPagination(c *fiber.Ctx) (int, int) {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func accessPaginated(c *fiber.Ctx, data interface{}, page, limit int, total int64, message string) error {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return c.JSON(fiber.Map{
		"success": true, "data": data, "message": message,
		"meta": fiber.Map{"pagination": fiber.Map{"page": page, "limit": limit, "total": total, "total_pages": totalPages}},
	})
}

func accessOptionalUUIDQuery(c *fiber.Ctx, key string) (*uuid.UUID, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func accessOptionalTimeQuery(c *fiber.Ctx, key string, endOfDay bool) (*time.Time, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, err
		}
	}
	if endOfDay && len(raw) == len("2006-01-02") {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
