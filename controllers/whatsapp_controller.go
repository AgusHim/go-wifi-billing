package controllers

import (
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type WhatsAppController struct {
	service services.WhatsAppService
}

func NewWhatsAppController(service services.WhatsAppService) *WhatsAppController {
	return &WhatsAppController{service}
}

func (c *WhatsAppController) RegisterRoutes(router fiber.Router) {
	admin := router.Group("/admin_api/whatsapp", middlewares.UserProtected())
	admin.Post("/bulk-send", c.BulkSend)
}

type bulkSendRequest struct {
	Messages []services.BulkMessageItem `json:"messages"`
}

func (c *WhatsAppController) BulkSend(ctx *fiber.Ctx) error {
	var req bulkSendRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	if len(req.Messages) == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "messages cannot be empty",
		})
	}

	result := c.service.SendBulkMessages(req.Messages)
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"message": "Bulk messages sent",
	})
}
