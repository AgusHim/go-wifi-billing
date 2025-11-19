package controllers

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type BillController struct {
	service services.BillService
}

func (c *BillController) RegisterRoutes(router fiber.Router) {
	api := router.Group("/api/bills")
	api.Get("/:public_id", c.GetByPublicID)

	user_api := router.Group("/user_api/bills")
	user_api.Get("/", c.GetAll)
	user_api.Get("/public/:public_id", c.GetByPublicID)
	admin_api := router.Group("/admin_api/bills")
	admin_api.Get("/generate", c.GenerateMonthlyBills)
	admin_api.Post("/create", c.Create)
	admin_api.Get("/", c.GetAll)
	admin_api.Get("/:id", c.GetByID)
	admin_api.Put("/:id", c.Update)
	admin_api.Delete("/:id", c.Delete)
}

func NewBillController(service services.BillService) *BillController {
	return &BillController{service}
}

func (c *BillController) GetAll(ctx *fiber.Ctx) error {
	data, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get all bills"})
}

func (c *BillController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Bill not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get bill"})
}
func (c *BillController) GetByPublicID(ctx *fiber.Ctx) error {
	publicID := ctx.Params("public_id")
	if publicID == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Public ID wajib diisi",
		})
	}

	data, err := c.service.GetByPublicID(publicID)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"success": false,
			"message": "Tagihan tidak ditemukan",
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    data,
		"message": "Success get bill by public ID",
	})
}
func (c *BillController) GetByUserID(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id")
	if userID == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}

	data, err := c.service.GetByUserID(userID.(string))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    data,
		"message": "Success get bills by user ID",
	})
}

func (c *BillController) Create(ctx *fiber.Ctx) error {
	var input models.Bill
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	data, err := c.service.Create(input)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Bill created successfully"})
}

func (c *BillController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input models.Bill
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	data, err := c.service.Update(id, input)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Bill updated successfully"})
}

func (c *BillController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Bill deleted successfully"})
}

func (c *BillController) GenerateMonthlyBills(ctx *fiber.Ctx) error {
	if err := c.service.GenerateMonthlyBills(); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Monthly bills generated successfully"})
}
