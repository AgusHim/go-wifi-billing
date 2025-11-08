package controllers

import (
	"strconv"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CustomerController struct {
	service services.CustomerService
}

func NewCustomerController(service services.CustomerService) *CustomerController {
	return &CustomerController{service}
}

func (c *CustomerController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/customers")
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
}

func (c *CustomerController) Create(ctx *fiber.Ctx) error {
	var customer dto.CreateCustomerDTO
	if err := ctx.BodyParser(&customer); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	new_customer, err := c.service.Create(&customer)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": new_customer, "message": "Customer created"})
}

func (c *CustomerController) GetAll(ctx *fiber.Ctx) error {
	pageStr := ctx.Query("page", "1")
	limitStr := ctx.Query("limit", "10")
	search := ctx.Query("search", "")
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	customers, total, err := c.service.GetAll(page, limit, search)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"meta": fiber.Map{
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": int((total + int64(limit) - 1) / int64(limit)),
			},
		},
		"data":    customers,
		"message": "Success get data",
	})
}

func (c *CustomerController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	customer, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Customer not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": customer, "message": "Success get data"})
}

func (c *CustomerController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	var customer dto.CreateCustomerDTO
	if err = ctx.BodyParser(&customer); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	updated, err := c.service.Update(id, &customer)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": updated, "message": "Customer updated"})
}

func (c *CustomerController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Customer deleted"})
}
