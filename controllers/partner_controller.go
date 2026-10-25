package controllers

import (
	"errors"
	"strings"

	"github.com/Agushim/go_wifi_billing/dto"
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PartnerController struct{ service services.PartnerService }

func NewPartnerController(service services.PartnerService) *PartnerController {
	return &PartnerController{service: service}
}
func (c *PartnerController) RegisterRoutes(router fiber.Router) {
	admin := router.Group("/admin_api/partners", middlewares.UserProtected())
	admin.Get("/", c.ListPartners)
	admin.Post("/", c.CreatePartner)
	admin.Post("/:id/members", c.AddMember)
	admin.Get("/:id/services", c.ListServices)
	admin.Post("/:id/services", c.CreateService)
	admin.Get("/invoices", c.ListInvoices)
	admin.Post("/invoices", c.CreateInvoice)
	admin.Get("/invoices/:id", c.GetInvoice)
	admin.Post("/invoices/:id/issue", c.IssueInvoice)
	admin.Post("/payments/:id/review", c.ReviewPayment)
	portal := router.Group("/partner_api", middlewares.UserProtected())
	portal.Get("/me", c.GetMyPartner)
	portal.Get("/services", c.GetMyServices)
	portal.Get("/invoices", c.ListMyInvoices)
	portal.Get("/invoices/:id", c.GetMyInvoice)
	portal.Post("/invoices/:id/payments", c.SubmitMyPayment)
}
func parseID(ctx *fiber.Ctx, key string) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Params(key))
	if err != nil {
		return uuid.Nil, errors.New("invalid id")
	}
	return id, nil
}
func (c *PartnerController) ListPartners(ctx *fiber.Ctx) error {
	rows, err := c.service.ListPartners()
	return respond(ctx, rows, err, "Success get partners")
}
func (c *PartnerController) CreatePartner(ctx *fiber.Ctx) error {
	var in dto.PartnerRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.CreatePartner(in)
	return respond(ctx, row, err, "Partner created")
}
func (c *PartnerController) AddMember(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	var in dto.PartnerMemberRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.AddMember(id, in)
	return respond(ctx, row, err, "Partner member added")
}
func (c *PartnerController) ListServices(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListServices(id)
	return respond(ctx, rows, err, "Success get partner services")
}
func (c *PartnerController) CreateService(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	var in dto.PartnerBandwidthServiceRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.CreateBandwidthService(id, in)
	return respond(ctx, row, err, "Partner service created")
}
func (c *PartnerController) ListInvoices(ctx *fiber.Ctx) error {
	rows, err := c.service.ListInvoices(nil)
	return respond(ctx, rows, err, "Success get partner invoices")
}
func (c *PartnerController) CreateInvoice(ctx *fiber.Ctx) error {
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerInvoiceRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.CreateInvoice(in, actor)
	return respond(ctx, row, err, "Partner invoice created")
}
func (c *PartnerController) GetInvoice(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.GetInvoice(id, nil)
	return respond(ctx, row, err, "Success get partner invoice")
}
func (c *PartnerController) IssueInvoice(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.IssueInvoice(id)
	return respond(ctx, row, err, "Partner invoice issued")
}
func (c *PartnerController) ReviewPayment(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerPaymentVerificationRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.VerifyPayment(id, in, actor)
	return respond(ctx, row, err, "Partner payment reviewed")
}
func (c *PartnerController) GetMyPartner(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	return respond(ctx, partner, err, "Success get partner profile")
}
func (c *PartnerController) GetMyServices(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	rows, err := c.service.ListServices(partner.ID)
	return respond(ctx, rows, err, "Success get services")
}
func (c *PartnerController) ListMyInvoices(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	rows, err := c.service.ListInvoices(&partner.ID)
	return respond(ctx, rows, err, "Success get invoices")
}
func (c *PartnerController) GetMyInvoice(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.GetInvoice(id, &partner.ID)
	return respond(ctx, row, err, "Success get invoice")
}
func (c *PartnerController) SubmitMyPayment(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	if _, err = c.service.GetInvoice(id, &partner.ID); err != nil {
		return respond(ctx, nil, err, "")
	}
	userID, _ := authenticatedUserID(ctx)
	var in dto.PartnerPaymentRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.SubmitPayment(id, in, userID)
	return respond(ctx, row, err, "Payment submitted for verification")
}
func (c *PartnerController) portalPartner(ctx *fiber.Ctx) (*models.Partner, error) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		return nil, errors.New("unauthorized")
	}
	return c.service.PartnerForUser(userID)
}
func badRequest(ctx *fiber.Ctx, err error) error {
	return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
}
func respond(ctx *fiber.Ctx, data interface{}, err error, message string) error {
	if err != nil {
		status := 500
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = 404
		} else if strings.Contains(strings.ToLower(err.Error()), "invalid") || strings.Contains(strings.ToLower(err.Error()), "required") || strings.Contains(strings.ToLower(err.Error()), "must") || strings.Contains(strings.ToLower(err.Error()), "only draft") || strings.Contains(strings.ToLower(err.Error()), "not payable") || strings.Contains(strings.ToLower(err.Error()), "already") {
			status = 400
		}
		return ctx.Status(status).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": message})
}
