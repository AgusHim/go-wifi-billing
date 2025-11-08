package controllers

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ComplainController struct {
	service services.ComplainService
}

func NewComplainController(service services.ComplainService) *ComplainController {
	return &ComplainController{service: service}
}

func (c *ComplainController) RegisterRoutes(router fiber.Router) {
	group := router.Group("/admin_api/complains")
	group.Post("/", c.Create)
	group.Get("/", c.GetAll)
	group.Get("/:id", c.GetByID)
	group.Put("/:id", c.Update)
	group.Delete("/:id", c.Delete)
}

func (c *ComplainController) Create(ctx *fiber.Ctx) error {
	var payload models.Complain
	if err := ctx.BodyParser(&payload); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	created, err := c.service.Create(&payload)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": "Complain created",
	})
}

func (c *ComplainController) GetAll(ctx *fiber.Ctx) error {
	complains, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    complains,
		"message": "Success get data",
	})
}

func (c *ComplainController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": "Invalid ID",
		})
	}
	complain, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    complain,
		"message": "Success get data",
	})
}

func (c *ComplainController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": "Invalid ID",
		})
	}
	var payload models.Complain
	if err := ctx.BodyParser(&payload); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	updated, err := c.service.Update(id, &payload)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    updated,
		"message": "Complain updated",
	})
}

func (c *ComplainController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": "Invalid ID",
		})
	}
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    nil,
		"message": "Complain deleted",
	})
}

