package controllers

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OdcController struct {
	service services.OdcService
}

func NewOdcController(service services.OdcService) *OdcController {
	return &OdcController{service}
}

func (c *OdcController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/odcs")
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
}

func (c *OdcController) Create(ctx *fiber.Ctx) error {
	var odc models.Odc
	if err := ctx.BodyParser(&odc); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	if err := c.service.Create(&odc); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": odc, "message": "ODC created"})
}

func (c *OdcController) GetAll(ctx *fiber.Ctx) error {
	odcs, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": odcs, "message": "Success get data"})
}

func (c *OdcController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	odc, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "ODC not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": odc, "message": "Success get data"})
}

func (c *OdcController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	var odc models.Odc
	if err = ctx.BodyParser(&odc); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	updated_odc, err := c.service.Update(id, &odc)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "ODC updated", "data": updated_odc})
}

func (c *OdcController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "ODC deleted"})
}
