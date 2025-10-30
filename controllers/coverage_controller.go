package controllers

import (
	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type CoverageController struct {
	service services.CoverageService
}

func NewCoverageController(s services.CoverageService) *CoverageController {
	return &CoverageController{service: s}
}

func (c *CoverageController) RegisterRoutes(app *fiber.App) {
	g := app.Group("/api/v1/coverages")
	g.Post("/", c.Create)
	g.Get("/", c.GetAll)
	g.Get("/:id", c.GetByID)
	g.Put("/:id", c.Update)
	g.Delete("/:id", c.Delete)
}

func (c *CoverageController) Create(ctx *fiber.Ctx) error {
	var input dto.CoverageCreateDTO
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": "invalid payload",
		})
	}
	created, err := c.service.Create(input)
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
		"message": "Coverage created",
	})
}

func (c *CoverageController) GetAll(ctx *fiber.Ctx) error {
	list, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    list,
		"message": "Success get data",
	})
}

func (c *CoverageController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	item, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    item,
		"message": "Success get data",
	})
}

func (c *CoverageController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input dto.CoverageUpdateDTO
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"data":    nil,
			"message": "invalid payload",
		})
	}
	updated, err := c.service.Update(id, input)
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
		"message": "Coverage updated",
	})
}

func (c *CoverageController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
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
		"message": "Coverage deleted",
	})
}
