package controllers

import (
	"strconv"

	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RenewalController struct {
	service services.RenewalService
}

func NewRenewalController(service services.RenewalService) *RenewalController {
	return &RenewalController{service: service}
}

func (c *RenewalController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/renewals")
	r.Post("/auto-generate", c.RunAutoGenerate)
	r.Get("/subscriptions/:id", c.GetSubscriptionHistory)
	r.Post("/subscriptions/:id/sync-recurring", c.SyncRecurringProfile)
}

func (c *RenewalController) RunAutoGenerate(ctx *fiber.Ctx) error {
	created, err := c.service.RunAutoGenerateNow()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"created": created,
		},
		"message": "Renewal auto-generate executed",
	})
}

func (c *RenewalController) GetSubscriptionHistory(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	limitStr := ctx.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	items, err := c.service.ListHistory(id, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Success get renewal history"})
}

func (c *RenewalController) SyncRecurringProfile(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.SyncRecurringProfile(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Recurring profile synced"})
}
