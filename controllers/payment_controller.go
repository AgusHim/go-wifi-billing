package controllers

import (
	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type PaymentController struct {
	service services.PaymentService
}

func NewPaymentController(service services.PaymentService) *PaymentController {
	return &PaymentController{service}
}

func (c *PaymentController) RegisterRoutes(router fiber.Router) {
	router.Get("/api/payment/callback", c.MidtransCallback)
	user_api := router.Group("/user_api/payments")
	user_api.Get("/", c.GetAll)
<<<<<<< Updated upstream
	user_api.Get("/midtrans", c.CreateMidtrans)
=======
	user_api.Get("/user/:user_id", c.GetByUserID)
>>>>>>> Stashed changes

	admin_api := router.Group("/admin_api/payments")
	admin_api.Post("/", c.Create)
	admin_api.Get("/", c.GetAll)
	admin_api.Get("/:id", c.GetByID)
	admin_api.Put("/:id", c.Update)
	admin_api.Delete("/:id", c.Delete)
}

func (c *PaymentController) GetAll(ctx *fiber.Ctx) error {
	data, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get all payments"})
}

func (c *PaymentController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Payment not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get payment"})
}
func (c *PaymentController) GetByUserID(ctx *fiber.Ctx) error {
	userID := ctx.Params("user_id")

	data, err := c.service.GetByUserID(userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    data,
		"message": "Success get payments by user",
	})
}
func (c *PaymentController) Create(ctx *fiber.Ctx) error {
	var input models.Payment
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	data, err := c.service.Create(input)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Payment created successfully"})
}

func (c *PaymentController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input models.Payment
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	data, err := c.service.Update(id, input)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Payment updated successfully"})
}

func (c *PaymentController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Payment deleted successfully"})
}

func (c *PaymentController) CreateMidtrans(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.service.CreateMidtransTransaction(id)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Payment updated successfully"})
}

func (c *PaymentController) MidtransCallback(ctx *fiber.Ctx) error {
	type NotificationPayload struct {
		TransactionStatus string `json:"transaction_status"`
		OrderID           string `json:"order_id"`
		GrossAmount       string `json:"gross_amount"`
		PaymentType       string `json:"payment_type"`
		FraudStatus       string `json:"fraud_status"`
		SignatureKey      string `json:"signature_key"`
		StatusCode        string `json:"status_code"`
	}

	var payload NotificationPayload
	if err := ctx.BodyParser(&payload); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid callback payload",
		})
	}

	expectedSig := lib.GenerateSignature(payload.OrderID, payload.StatusCode, payload.GrossAmount)
	if expectedSig != payload.SignatureKey {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid signature key",
		})
	}

	// TODO: update status pembayaran di DB
	// contoh update order di DB pakai GORM
	err := c.service.HandleMindtransCallback(payload.OrderID, payload.TransactionStatus)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update payment status",
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "Payment status updated",
	})
}
