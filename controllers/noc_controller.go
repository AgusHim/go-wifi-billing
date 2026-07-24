package controllers

import (
	"strings"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type NOCController struct {
	service services.NOCService
	alerts  services.AlertService
	actions services.NOCActionService
}

func NewNOCController(service services.NOCService, alerts services.AlertService, actions services.NOCActionService) *NOCController {
	return &NOCController{service: service, alerts: alerts, actions: actions}
}

func (c *NOCController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/noc", middlewares.UserProtected())
	r.Get("/overview", c.GetOverview)
	r.Get("/routers", c.GetRouters)
	r.Get("/routers/:id/snapshots", c.GetRouterSnapshots)
	r.Get("/routers/:id/interfaces", c.GetRouterInterfaces)
	r.Get("/customers", c.GetCustomers)
	r.Get("/reconciliation", c.GetReconciliationFindings)
	r.Get("/alerts", c.GetAlerts)
	r.Get("/metrics", c.GetMetrics)
	r.Post("/collect", c.CollectAll)
	r.Post("/alerts/evaluate", c.EvaluateAlerts)
	r.Post("/alerts/:id/ack", c.AcknowledgeAlert)
	r.Post("/alerts/:id/resolve", c.ResolveAlert)
	r.Post("/service-accounts/:id/actions", c.RunServiceAccountAction)
	r.Post("/reconciliation/:id/resolve", c.ResolveReconciliationFinding)
}

func (c *NOCController) GetMetrics(ctx *fiber.Ctx) error {
	result, err := c.service.GetMetrics()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC metrics retrieved"})
}

func (c *NOCController) GetOverview(ctx *fiber.Ctx) error {
	result, err := c.service.GetOverview()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC overview retrieved"})
}

func (c *NOCController) CollectAll(ctx *fiber.Ctx) error {
	result, err := c.service.CollectAll()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC collection executed"})
}

func (c *NOCController) GetRouters(ctx *fiber.Ctx) error {
	result, err := c.service.GetRouters()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC routers retrieved"})
}

func (c *NOCController) GetRouterSnapshots(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	limit := ctx.QueryInt("limit", 100)
	result, err := c.service.GetRouterSnapshots(id, limit)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router snapshots retrieved"})
}

func (c *NOCController) GetRouterInterfaces(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	limit := ctx.QueryInt("limit", 200)
	result, err := c.service.GetRouterInterfaces(id, limit)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router interface snapshots retrieved"})
}

func (c *NOCController) GetCustomers(ctx *fiber.Ctx) error {
	status := ctx.Query("status", "")
	routerID := ctx.Query("router_id", "")
	var coverageIDs []string
	ctx.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		v := strings.TrimSpace(string(value))
		if (k == "coverage_ids" || k == "coverage_ids[]") && v != "" {
			coverageIDs = append(coverageIDs, v)
		}
	})
	if len(coverageIDs) == 0 {
		if coverageID := strings.TrimSpace(ctx.Query("coverage_id", "")); coverageID != "" {
			coverageIDs = []string{coverageID}
		}
	}
	packageID := ctx.Query("package_id", "")
	limit := ctx.QueryInt("limit", 200)
	result, err := c.service.GetCustomers(status, routerID, coverageIDs, packageID, limit)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC customers retrieved"})
}

func (c *NOCController) GetReconciliationFindings(ctx *fiber.Ctx) error {
	status := ctx.Query("status", "open")
	result, err := c.service.GetReconciliationFindings(status)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Reconciliation findings retrieved"})
}

func (c *NOCController) ResolveReconciliationFinding(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.ResolveReconciliationFinding(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Reconciliation finding resolved"})
}

func (c *NOCController) GetAlerts(ctx *fiber.Ctx) error {
	status := ctx.Query("status", "open")
	limit := ctx.QueryInt("limit", 200)
	result, err := c.alerts.GetAlerts(status, limit)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC alerts retrieved"})
}

func (c *NOCController) EvaluateAlerts(ctx *fiber.Ctx) error {
	result, err := c.alerts.Evaluate()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC alerts evaluated"})
}

func (c *NOCController) AcknowledgeAlert(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.alerts.Acknowledge(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Alert acknowledged"})
}

func (c *NOCController) ResolveAlert(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.alerts.Resolve(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Alert resolved"})
}

func (c *NOCController) RunServiceAccountAction(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var request services.NOCActionRequest
	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	result, err := c.actions.RunServiceAccountAction(id, request)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "NOC action executed"})
}
