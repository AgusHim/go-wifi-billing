package controllers

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SubscriptionController struct {
	service services.SubscriptionService
}

func NewSubscriptionController(service services.SubscriptionService) *SubscriptionController {
	return &SubscriptionController{service}
}

func (c *SubscriptionController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/subscriptions")
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
}

func (c *SubscriptionController) Create(ctx *fiber.Ctx) error {
	var subscription models.Subscription
	if err := ctx.BodyParser(&subscription); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.Create(&subscription); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": subscription, "message": "Subscription created"})
}

func (c *SubscriptionController) GetAll(ctx *fiber.Ctx) error {
	subscriptions, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": subscriptions, "message": "Success get data"})
}

func (c *SubscriptionController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	subscription, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Subscription not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": subscription, "message": "Success get data"})
}

func (c *SubscriptionController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	var subscription models.Subscription
	if err := ctx.BodyParser(&subscription); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.Update(id, &subscription); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Subscription updated"})
}

func (c *SubscriptionController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Subscription deleted"})
}
