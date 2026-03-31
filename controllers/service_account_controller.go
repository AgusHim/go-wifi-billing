package controllers

import (
	"strconv"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ServiceAccountController struct {
	service services.ServiceAccountService
}

func NewServiceAccountController(service services.ServiceAccountService) *ServiceAccountController {
	return &ServiceAccountController{service: service}
}

func (c *ServiceAccountController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/service-accounts")
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Get("/:id/status-history", c.GetStatusHistory)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
	r.Post("/:id/provision", c.Provision)
	r.Post("/:id/suspend", c.Suspend)
	r.Post("/:id/unsuspend", c.Unsuspend)
	r.Post("/:id/terminate", c.Terminate)
	r.Post("/:id/change-plan", c.ChangePlan)
}

func (c *ServiceAccountController) GetAll(ctx *fiber.Ctx) error {
	subscriptionID := ctx.Query("subscription_id", "")
	items, err := c.service.GetAll(subscriptionID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Success get data"})
}

func (c *ServiceAccountController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Success get data"})
}

func (c *ServiceAccountController) GetStatusHistory(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	limit, _ := strconv.Atoi(ctx.Query("limit", "50"))
	items, err := c.service.GetStatusHistory(id, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Success get status history"})
}

func (c *ServiceAccountController) Create(ctx *fiber.Ctx) error {
	var input models.ServiceAccount
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.Create(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Service account created"})
}

func (c *ServiceAccountController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input models.ServiceAccount
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.Update(id, &input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Service account updated"})
}

func (c *ServiceAccountController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Service account deleted"})
}

func (c *ServiceAccountController) Provision(ctx *fiber.Ctx) error {
	return c.enqueueAction(ctx, "create_account", "Provisioning job queued")
}

func (c *ServiceAccountController) Suspend(ctx *fiber.Ctx) error {
	return c.enqueueAction(ctx, "suspend_account", "Suspend job queued")
}

func (c *ServiceAccountController) Unsuspend(ctx *fiber.Ctx) error {
	return c.enqueueAction(ctx, "unsuspend_account", "Unsuspend job queued")
}

func (c *ServiceAccountController) Terminate(ctx *fiber.Ctx) error {
	return c.enqueueAction(ctx, "terminate_account", "Terminate job queued")
}

func (c *ServiceAccountController) ChangePlan(ctx *fiber.Ctx) error {
	return c.enqueueAction(ctx, "change_plan", "Change plan job queued")
}

func (c *ServiceAccountController) enqueueAction(ctx *fiber.Ctx, action string, message string) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	job, err := c.service.EnqueueAction(id, action)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": job, "message": message})
}
