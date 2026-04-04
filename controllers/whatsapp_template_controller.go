package controllers

import (
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type WhatsAppTemplateController struct {
	service services.WhatsAppTemplateService
}

func NewWhatsAppTemplateController(service services.WhatsAppTemplateService) *WhatsAppTemplateController {
	return &WhatsAppTemplateController{service}
}

func (c *WhatsAppTemplateController) RegisterRoutes(router fiber.Router) {
	admin := router.Group("/admin_api/whatsapp/templates", middlewares.UserProtected())
	admin.Get("/", c.GetAll)
	admin.Post("/", c.Create)
	admin.Get("/:id", c.GetByID)
	admin.Put("/:id", c.Update)
	admin.Delete("/:id", c.Delete)
}

type templateRequest struct {
	Name    string `json:"name"`
	Key     string `json:"key"`
	Content string `json:"content"`
}

func (c *WhatsAppTemplateController) GetAll(ctx *fiber.Ctx) error {
	templates, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    templates,
	})
}

func (c *WhatsAppTemplateController) Create(ctx *fiber.Ctx) error {
	var req templateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}
	if req.Name == "" || req.Key == "" || req.Content == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "name, key, dan content wajib diisi",
		})
	}

	template := &models.WhatsAppTemplate{
		Name:    req.Name,
		Key:     req.Key,
		Content: req.Content,
	}
	if err := c.service.Create(template); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    template,
		"message": "Template berhasil dibuat",
	})
}

func (c *WhatsAppTemplateController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "id tidak valid",
		})
	}
	template, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	if template == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "template tidak ditemukan",
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    template,
	})
}

func (c *WhatsAppTemplateController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "id tidak valid",
		})
	}
	template, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	if template == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "template tidak ditemukan",
		})
	}

	var req templateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Key != "" {
		template.Key = req.Key
	}
	if req.Content != "" {
		template.Content = req.Content
	}

	if err := c.service.Update(template); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    template,
		"message": "Template berhasil diperbarui",
	})
}

func (c *WhatsAppTemplateController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "id tidak valid",
		})
	}
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "Template berhasil dihapus",
	})
}
