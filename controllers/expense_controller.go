package controllers

import (
	"fmt"
	"strings"
	"time"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ExpenseController struct {
	service services.ExpenseService
}

func NewExpenseController(service services.ExpenseService) *ExpenseController {
	return &ExpenseController{service}
}

func (c *ExpenseController) RegisterRoutes(router fiber.Router) {
	admin_api := router.Group("/admin_api/expenses", middlewares.UserProtected())
	admin_api.Get("/", c.GetAll)
	admin_api.Get("/:id", c.GetByID)
	admin_api.Post("/", c.Create)
	admin_api.Put("/:id", c.Update)
	admin_api.Delete("/:id", c.Delete)
}

func (c *ExpenseController) GetAll(ctx *fiber.Ctx) error {
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	category := strings.TrimSpace(ctx.Query("category", ""))
	startAt := strings.TrimSpace(ctx.Query("start_at", ""))
	endAt := strings.TrimSpace(ctx.Query("end_at", ""))

	data, err := c.service.GetAll(adminID, category, startAt, endAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get all expenses"})
}

func (c *ExpenseController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Expense not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get expense"})
}

func parseExpenseForm(ctx *fiber.Ctx) (models.Expense, error) {
	var input models.Expense

	input.Title = strings.TrimSpace(ctx.FormValue("title"))
	input.Category = strings.TrimSpace(ctx.FormValue("category"))
	input.Description = strings.TrimSpace(ctx.FormValue("description"))

	amountStr := strings.TrimSpace(ctx.FormValue("amount"))
	if amountStr != "" {
		var amount int
		if _, err := fmt.Sscan(amountStr, &amount); err != nil {
			return input, fmt.Errorf("invalid amount")
		}
		input.Amount = amount
	}

	expenseDateStr := strings.TrimSpace(ctx.FormValue("expense_date"))
	if expenseDateStr != "" {
		t, err := time.Parse("2006-01-02", expenseDateStr)
		if err != nil {
			return input, fmt.Errorf("invalid expense_date format, expected YYYY-MM-DD")
		}
		input.ExpenseDate = t
	}

	adminIDStr := strings.TrimSpace(ctx.FormValue("admin_id"))
	if adminIDStr != "" {
		uid, err := uuid.Parse(adminIDStr)
		if err != nil {
			return input, fmt.Errorf("invalid admin_id")
		}
		input.AdminID = &uid
	}

	return input, nil
}

func (c *ExpenseController) Create(ctx *fiber.Ctx) error {
	input, err := parseExpenseForm(ctx)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	fileHeader, _ := ctx.FormFile("proof_image")

	data, err := c.service.Create(input, fileHeader)
	if err != nil {
		statusCode := 500
		msg := err.Error()
		if strings.Contains(msg, "invalid file type") {
			statusCode = 400
		} else if strings.Contains(msg, "file size exceeds") {
			statusCode = 413
		}
		return ctx.Status(statusCode).JSON(fiber.Map{"success": false, "message": msg})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Expense created successfully"})
}

func (c *ExpenseController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	input, err := parseExpenseForm(ctx)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	fileHeader, _ := ctx.FormFile("proof_image")

	data, err := c.service.Update(id, input, fileHeader)
	if err != nil {
		statusCode := 500
		msg := err.Error()
		if strings.Contains(msg, "invalid file type") {
			statusCode = 400
		} else if strings.Contains(msg, "file size exceeds") {
			statusCode = 413
		}
		return ctx.Status(statusCode).JSON(fiber.Map{"success": false, "message": msg})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Expense updated successfully"})
}

func (c *ExpenseController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Expense deleted successfully"})
}
