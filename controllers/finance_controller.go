package controllers

import (
	"strings"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type FinanceController struct {
	service services.FinanceService
}

func NewFinanceController(service services.FinanceService) *FinanceController {
	return &FinanceController{service}
}

func (c *FinanceController) RegisterRoutes(router fiber.Router) {
	admin_api := router.Group("/admin_api/finance", middlewares.UserProtected())
	admin_api.Get("/summary", c.GetSummary)
	admin_api.Get("/monthly", c.GetMonthly)
	admin_api.Get("/by-subscription", c.GetBySubscription)
}

func (c *FinanceController) GetSummary(ctx *fiber.Ctx) error {
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	startAt := strings.TrimSpace(ctx.Query("start_at", ""))
	endAt := strings.TrimSpace(ctx.Query("end_at", ""))

	data, err := c.service.GetSummary(adminID, startAt, endAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get finance summary"})
}

func (c *FinanceController) GetMonthly(ctx *fiber.Ctx) error {
	data, err := c.service.GetMonthly(12)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get monthly finance"})
}

func (c *FinanceController) GetBySubscription(ctx *fiber.Ctx) error {
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	startAt := strings.TrimSpace(ctx.Query("start_at", ""))
	endAt := strings.TrimSpace(ctx.Query("end_at", ""))

	data, err := c.service.GetBySubscription(adminID, startAt, endAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get finance by subscription"})
}
