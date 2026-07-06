package controllers

import (
	"strings"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OdpController struct {
	service services.OdpService
}

func NewOdpController(service services.OdpService) *OdpController {
	return &OdpController{service}
}

func (c *OdpController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/odps")
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
}

func (c *OdpController) Create(ctx *fiber.Ctx) error {
	var odp models.Odp
	if err := ctx.BodyParser(&odp); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.Create(&odp); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": odp, "message": "ODP created"})
}

func (c *OdpController) GetAll(ctx *fiber.Ctx) error {
	var coverageIDs []string
	for k, values := range ctx.Queries() {
		if (k == "coverage_ids" || k == "coverage_ids[]") && values != "" {
			for _, v := range strings.Split(values, ",") {
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					coverageIDs = append(coverageIDs, trimmed)
				}
			}
		}
	}
	if len(coverageIDs) == 0 {
		if coverageID := strings.TrimSpace(ctx.Query("coverage_id", "")); coverageID != "" {
			coverageIDs = append(coverageIDs, coverageID)
		}
	}

	odps, err := c.service.GetAll(coverageIDs)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid coverage_id") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": odps, "message": "Success get data"})
}

func (c *OdpController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	odp, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "ODP not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": odp, "message": "Success get data"})
}

func (c *OdpController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	var odp models.Odp
	if err = ctx.BodyParser(&odp); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	updated_odp, err := c.service.Update(id, &odp)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "ODP updated", "data": updated_odp})
}

func (c *OdpController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "ODP deleted"})
}
