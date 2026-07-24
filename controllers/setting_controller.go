package controllers

import (
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type SettingController struct {
	service services.SettingService
}

func NewSettingController(service services.SettingService) *SettingController {
	return &SettingController{service}
}

func (c *SettingController) RegisterRoutes(router fiber.Router) {
	api := router.Group("/admin_api/settings", middlewares.UserProtected())
	api.Get("/", c.GetAll)
	api.Get("/:key", c.GetByKey)
	api.Put("/:key", c.UpdateOrCreate)
}

func (c *SettingController) GetAll(ctx *fiber.Ctx) error {
	settings, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	// Convert to map for easier access on frontend
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	return ctx.JSON(fiber.Map{"success": true, "data": settingsMap, "message": "Success"})
}

func (c *SettingController) GetByKey(ctx *fiber.Ctx) error {
	key := ctx.Params("key")
	setting, err := c.service.GetByKey(key)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Setting not found"})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": setting, "message": "Success"})
}

type UpdateSettingRequest struct {
	Value string `json:"value" validate:"required"`
}

func (c *SettingController) UpdateOrCreate(ctx *fiber.Ctx) error {
	key := ctx.Params("key")

	var req UpdateSettingRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	setting, err := c.service.UpdateOrCreate(key, req.Value)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": setting, "message": "Setting updated successfully"})
}
