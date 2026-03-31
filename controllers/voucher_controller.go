package controllers

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type VoucherController struct {
	service services.VoucherService
}

func NewVoucherController(service services.VoucherService) *VoucherController {
	return &VoucherController{service: service}
}

func (c *VoucherController) RegisterRoutes(router fiber.Router) {
	batches := router.Group("/admin_api/voucher-batches")
	batches.Get("/", c.GetBatches)
	batches.Get("/:id", c.GetBatchByID)
	batches.Post("/", c.CreateBatch)

	vouchers := router.Group("/admin_api/vouchers")
	vouchers.Get("/", c.GetVouchers)
	vouchers.Get("/:id", c.GetVoucherByID)

	router.Post("/api/vouchers/redeem", c.Redeem)
}

func (c *VoucherController) CreateBatch(ctx *fiber.Ctx) error {
	var input models.VoucherBatch
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateBatch(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Voucher batch created"})
}

func (c *VoucherController) GetBatches(ctx *fiber.Ctx) error {
	items, err := c.service.GetBatches()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Success get data"})
}

func (c *VoucherController) GetBatchByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetBatchByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Success get data"})
}

func (c *VoucherController) GetVouchers(ctx *fiber.Ctx) error {
	items, err := c.service.GetVouchers(ctx.Query("batch_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Success get data"})
}

func (c *VoucherController) GetVoucherByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetVoucherByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Success get data"})
}

func (c *VoucherController) Redeem(ctx *fiber.Ctx) error {
	var input struct {
		Code          string `json:"code"`
		RedeemerName  string `json:"redeemer_name"`
		RedeemerPhone string `json:"redeemer_phone"`
	}
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	voucher, job, err := c.service.Redeem(input.Code, input.RedeemerName, input.RedeemerPhone)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"voucher": voucher,
			"job":     job,
		},
		"message": "Voucher redeemed",
	})
}
