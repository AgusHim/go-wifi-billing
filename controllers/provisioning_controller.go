package controllers

import (
	"strconv"

	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type ProvisioningController struct {
	service services.ProvisioningService
}

func NewProvisioningController(service services.ProvisioningService) *ProvisioningController {
	return &ProvisioningController{service: service}
}

func (c *ProvisioningController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/provisioning")
	r.Get("/jobs", c.GetJobs)
	r.Get("/logs", c.GetLogs)
}

func (c *ProvisioningController) GetJobs(ctx *fiber.Ctx) error {
	limit, _ := strconv.Atoi(ctx.Query("limit", "50"))
	jobs, err := c.service.ListJobs(limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": jobs, "message": "Success get data"})
}

func (c *ProvisioningController) GetLogs(ctx *fiber.Ctx) error {
	limit, _ := strconv.Atoi(ctx.Query("limit", "100"))
	logs, err := c.service.ListLogs(limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": logs, "message": "Success get data"})
}
