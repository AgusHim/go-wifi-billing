package controllers

import (
	"crypto/subtle"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/lib"
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
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
	admin.Get("/aging", c.AgingReport)
	admin.Get("/usages", c.ListAllUsages)
	admin.Get("/allocations", c.ListAllAllocations)
	admin.Get("/accounts", c.ListAllAccounts)
	admin.Post("/reminders/send", c.SendReminders)
	admin.Post("/invoices/mark-overdue", c.MarkOverdueInvoices)
	admin.Post("/invoices/generate-recurring", c.GenerateRecurringDrafts)
	admin.Get("/legal-documents", c.ListLegalDocuments)
	admin.Post("/legal-documents", c.CreateLegalDocument)
	admin.Put("/legal-documents/:id", c.UpdateLegalDocument)
	admin.Delete("/legal-documents/:id", c.DeleteLegalDocument)
	admin.Get("/invoices", c.ListInvoices)
	admin.Post("/invoices", c.CreateInvoice)
	admin.Post("/invoices/from-service", c.GenerateInvoiceFromService)
	admin.Get("/invoices/:id", c.GetInvoice)
	admin.Post("/invoices/:id/issue", c.IssueInvoice)
	admin.Post("/invoices/:id/void", c.VoidInvoice)
	admin.Get("/invoices/:id/history", c.InvoiceHistory)
	admin.Get("/invoices/:id/notifications", c.InvoiceNotifications)
	admin.Post("/invoices/:id/resend", c.ResendInvoiceNotification)
	admin.Post("/usages/:id/review", c.ReviewUsage)
	admin.Post("/allocations/:id/return", c.ReturnAllocation)
	admin.Post("/payments/:id/review", c.ReviewPayment)
	admin.Put("/services/:id", c.UpdateService)
	admin.Get("/:id", c.GetPartner)
	admin.Put("/:id", c.UpdatePartner)
	admin.Get("/:id/accounts", c.ListPartnerAccounts)
	admin.Post("/:id/accounts", c.CreatePartnerAccount)
	admin.Post("/:id/payments", c.RecordPartnerPayment)
	admin.Post("/:id/members", c.AddMember)
	admin.Get("/:id/services", c.ListServices)
	admin.Post("/:id/services", c.CreateService)
	admin.Get("/:id/usages", c.ListUsages)
	admin.Post("/:id/usages", c.CreateUsage)
	admin.Get("/:id/allocations", c.ListAllocations)
	admin.Post("/:id/allocations", c.CreateAllocation)

	portal := router.Group("/partner_api", middlewares.UserProtected())
	portal.Get("/me", c.GetMyPartner)
	portal.Get("/services", c.GetMyServices)
	portal.Get("/usages", c.GetMyUsages)
	portal.Get("/allocations", c.GetMyAllocations)
	portal.Get("/documents", c.GetMyDocuments)
	portal.Get("/documents/:id", c.GetMyDocument)
	portal.Get("/aging", c.GetMyAging)
	portal.Get("/invoices", c.ListMyInvoices)
	portal.Get("/invoices/:id", c.GetMyInvoice)
	portal.Get("/invoices/:id/history", c.GetMyInvoiceHistory)
	portal.Post("/invoices/:id/payments", c.SubmitMyPayment)

	// Brevo transactional webhooks. The handler verifies a shared secret header.
	router.Post("/api/webhooks/brevo", c.BrevoWebhook)
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
func (c *PartnerController) GetPartner(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.GetPartner(id)
	return respond(ctx, row, err, "Success get partner")
}
func (c *PartnerController) UpdatePartner(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	var in dto.PartnerRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.UpdatePartner(id, in)
	return respond(ctx, row, err, "Partner updated")
}
func (c *PartnerController) UpdateService(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	var in dto.PartnerBandwidthServiceRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.UpdateBandwidthService(id, in)
	return respond(ctx, row, err, "Partner service updated")
}
func (c *PartnerController) ListPartnerAccounts(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListPartnerAccounts(id)
	return respond(ctx, rows, err, "Success get partner accounts")
}
func (c *PartnerController) CreatePartnerAccount(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, _ := authenticatedUserID(ctx)
	var in dto.PartnerAccountRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.CreatePartnerAccount(id, in, actor)
	return respond(ctx, row, err, "Partner login account created")
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
	partnerID, err := queryPartnerID(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	filter := repositories.PartnerInvoiceFilter{
		PartnerID:   partnerID,
		Status:      strings.TrimSpace(ctx.Query("status")),
		OverdueOnly: strings.TrimSpace(ctx.Query("overdue")) == "true",
		UnpaidOnly:  strings.TrimSpace(ctx.Query("unpaid")) == "true",
	}
	rows, err := c.service.ListInvoicesFiltered(filter)
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
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	row, err := c.service.IssueInvoice(id, actor)
	return respond(ctx, row, err, "Partner invoice issued")
}
func (c *PartnerController) GenerateInvoiceFromService(ctx *fiber.Ctx) error {
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerInvoiceFromServiceRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	partnerID, err := uuid.Parse(strings.TrimSpace(in.PartnerID))
	if err != nil {
		return badRequest(ctx, errors.New("invalid partner_id"))
	}
	row, err := c.service.GenerateInvoiceForService(partnerID, in, actor)
	return respond(ctx, row, err, "Draft invoice generated")
}
func (c *PartnerController) VoidInvoice(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.VoidInvoice(id, in.Reason, actor)
	return respond(ctx, row, err, "Partner invoice voided")
}
func (c *PartnerController) InvoiceHistory(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListInvoiceHistory(id)
	return respond(ctx, rows, err, "Success get invoice history")
}
func (c *PartnerController) InvoiceNotifications(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListInvoiceNotifications(id)
	return respond(ctx, rows, err, "Success get invoice notifications")
}
func (c *PartnerController) ResendInvoiceNotification(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, _ := authenticatedUserID(ctx)
	row, err := c.service.ResendInvoiceNotification(id, actor)
	return respond(ctx, row, err, "Invoice notification queued")
}
func (c *PartnerController) ReviewUsage(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerBandwidthUsageApprovalRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.ReviewUsage(id, in, actor)
	return respond(ctx, row, err, "Usage reviewed")
}
func (c *PartnerController) CreateUsage(ctx *fiber.Ctx) error {
	partnerID, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerBandwidthUsageRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.RecordUsage(partnerID, in, actor)
	return respond(ctx, row, err, "Usage recorded")
}
func (c *PartnerController) ListUsages(ctx *fiber.Ctx) error {
	partnerID, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListUsages(&partnerID, nil, strings.TrimSpace(ctx.Query("status")))
	return respond(ctx, rows, err, "Success get usages")
}
func (c *PartnerController) CreateAllocation(ctx *fiber.Ctx) error {
	partnerID, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerAllocationRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.CreateAllocation(partnerID, in, actor)
	return respond(ctx, row, err, "Inventory allocated")
}
func (c *PartnerController) ListAllocations(ctx *fiber.Ctx) error {
	partnerID, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListAllocations(&partnerID, strings.TrimSpace(ctx.Query("status")), strings.TrimSpace(ctx.Query("transaction_type")))
	return respond(ctx, rows, err, "Success get allocations")
}
func (c *PartnerController) ReturnAllocation(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.PartnerAllocationReturnRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.ReturnAllocation(id, in, actor)
	return respond(ctx, row, err, "Allocation returned")
}
func (c *PartnerController) AgingReport(ctx *fiber.Ctx) error {
	rows, err := c.service.AgingReport(nil)
	return respond(ctx, rows, err, "Success get aging report")
}

// queryPartnerID reads an optional partner_id filter shared by the
// cross-partner list endpoints.
func queryPartnerID(ctx *fiber.Ctx) (*uuid.UUID, error) {
	raw := strings.TrimSpace(ctx.Query("partner_id"))
	if raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, errors.New("invalid partner_id")
	}
	return &parsed, nil
}

// ListAllUsages lists bandwidth readings across every partner.
func (c *PartnerController) ListAllUsages(ctx *fiber.Ctx) error {
	partnerID, err := queryPartnerID(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListUsages(partnerID, nil, strings.TrimSpace(ctx.Query("status")))
	return respond(ctx, rows, err, "Success get usages")
}

// ListAllAllocations lists inventory allocations across every partner.
func (c *PartnerController) ListAllAllocations(ctx *fiber.Ctx) error {
	partnerID, err := queryPartnerID(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListAllocations(partnerID, strings.TrimSpace(ctx.Query("status")), strings.TrimSpace(ctx.Query("transaction_type")))
	return respond(ctx, rows, err, "Success get allocations")
}

// ListAllAccounts lists every partner portal login account.
func (c *PartnerController) ListAllAccounts(ctx *fiber.Ctx) error {
	rows, err := c.service.ListAllPartnerAccounts()
	return respond(ctx, rows, err, "Success get partner accounts")
}
func (c *PartnerController) SendReminders(ctx *fiber.Ctx) error {
	count, err := c.service.SendDueReminders()
	return respond(ctx, fiber.Map{"sent": count}, err, "Reminders processed")
}
func (c *PartnerController) MarkOverdueInvoices(ctx *fiber.Ctx) error {
	count, err := c.service.MarkOverdueInvoices()
	return respond(ctx, fiber.Map{"updated": count}, err, "Overdue invoices marked")
}
func (c *PartnerController) GenerateRecurringDrafts(ctx *fiber.Ctx) error {
	count, err := c.service.GenerateRecurringDrafts()
	return respond(ctx, fiber.Map{"created": count}, err, "Recurring drafts generated")
}

// legalDocumentFilter builds the archive filter from the request query. The
// archive is filtered per partner because each partner owns its own documents,
// while documents without a partner apply to everyone.
func legalDocumentFilter(ctx *fiber.Ctx) (repositories.LegalDocumentFilter, error) {
	filter := repositories.LegalDocumentFilter{Category: strings.TrimSpace(ctx.Query("category"))}
	switch scope := strings.TrimSpace(ctx.Query("partner_id")); {
	case scope == "":
		// No partner filter: return the whole archive.
	case scope == "global" || scope == "unassigned":
		filter.UnassignedOnly = true
	default:
		partnerID, err := uuid.Parse(scope)
		if err != nil {
			return filter, errors.New("invalid partner_id")
		}
		filter.PartnerID = &partnerID
	}
	return filter, nil
}

func (c *PartnerController) ListLegalDocuments(ctx *fiber.Ctx) error {
	filter, err := legalDocumentFilter(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	rows, err := c.service.ListLegalDocuments(filter)
	return respond(ctx, rows, err, "Success get legal documents")
}
func (c *PartnerController) CreateLegalDocument(ctx *fiber.Ctx) error {
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	var in dto.LegalDocumentRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.CreateLegalDocument(in, actor)
	return respond(ctx, row, err, "Legal document created")
}
func (c *PartnerController) UpdateLegalDocument(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	var in dto.LegalDocumentRequest
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.UpdateLegalDocument(id, in)
	return respond(ctx, row, err, "Legal document updated")
}
func (c *PartnerController) DeleteLegalDocument(ctx *fiber.Ctx) error {
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	err = c.service.DeleteLegalDocument(id)
	return respond(ctx, fiber.Map{"deleted": err == nil}, err, "Legal document deleted")
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

// proofUpload extracts an optional uploaded payment proof from a multipart
// request and stores it on Cloudinary. Returns an empty string when the request
// is JSON or carries no file.
func proofUpload(ctx *fiber.Ctx) (string, error) {
	if !strings.HasPrefix(strings.ToLower(ctx.Get("Content-Type")), "multipart/form-data") {
		return "", nil
	}
	file, err := ctx.FormFile("proof")
	if err != nil || file == nil {
		return "", nil
	}
	url, _, uploadErr := lib.UploadPartnerPaymentProof(file)
	if uploadErr != nil {
		return "", uploadErr
	}
	return url, nil
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
	in, err := paymentRequest(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	// A transfer proof uploaded with the submission wins over a pasted URL.
	uploaded, err := proofUpload(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	if uploaded != "" {
		in.ProofURL = uploaded
	}
	row, err := c.service.SubmitPayment(id, in, userID)
	return respond(ctx, row, err, "Payment submitted for verification")
}

// paymentRequest reads the payment payload from either a JSON body or a
// multipart form so the portal can attach an uploaded transfer proof.
func paymentRequest(ctx *fiber.Ctx) (dto.PartnerPaymentRequest, error) {
	if strings.HasPrefix(strings.ToLower(ctx.Get("Content-Type")), "multipart/form-data") {
		amount, _ := strconv.Atoi(strings.TrimSpace(ctx.FormValue("amount")))
		return dto.PartnerPaymentRequest{
			Amount:         amount,
			Method:         strings.TrimSpace(ctx.FormValue("method")),
			Reference:      strings.TrimSpace(ctx.FormValue("reference")),
			ProofURL:       strings.TrimSpace(ctx.FormValue("proof_url")),
			IdempotencyKey: strings.TrimSpace(ctx.FormValue("idempotency_key")),
		}, nil
	}
	var in dto.PartnerPaymentRequest
	if err := ctx.BodyParser(&in); err != nil {
		return in, errors.New("invalid payment payload")
	}
	return in, nil
}

// RecordPartnerPayment lets staff settle an invoice on behalf of a partner, for
// example when cash or a bank transfer is received at the office.
func (c *PartnerController) RecordPartnerPayment(ctx *fiber.Ctx) error {
	partnerID, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	actor, ok := authenticatedUserID(ctx)
	if !ok {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	in, err := staffPaymentRequest(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	uploaded, err := proofUpload(ctx)
	if err != nil {
		return badRequest(ctx, err)
	}
	if uploaded != "" {
		in.ProofURL = uploaded
	}
	row, err := c.service.RecordPartnerPayment(partnerID, in, actor)
	return respond(ctx, row, err, "Partner payment recorded")
}

// staffPaymentRequest reads the staff payment payload from JSON or multipart.
func staffPaymentRequest(ctx *fiber.Ctx) (dto.PartnerStaffPaymentRequest, error) {
	if strings.HasPrefix(strings.ToLower(ctx.Get("Content-Type")), "multipart/form-data") {
		amount, _ := strconv.Atoi(strings.TrimSpace(ctx.FormValue("amount")))
		request := dto.PartnerStaffPaymentRequest{
			InvoiceID:      strings.TrimSpace(ctx.FormValue("invoice_id")),
			Amount:         amount,
			Method:         strings.TrimSpace(ctx.FormValue("method")),
			Reference:      strings.TrimSpace(ctx.FormValue("reference")),
			ProofURL:       strings.TrimSpace(ctx.FormValue("proof_url")),
			IdempotencyKey: strings.TrimSpace(ctx.FormValue("idempotency_key")),
		}
		if raw := strings.TrimSpace(ctx.FormValue("auto_verify")); raw != "" {
			value := raw == "1" || strings.EqualFold(raw, "true")
			request.AutoVerify = &value
		}
		return request, nil
	}
	var in dto.PartnerStaffPaymentRequest
	if err := ctx.BodyParser(&in); err != nil {
		return in, errors.New("invalid payment payload")
	}
	return in, nil
}
func (c *PartnerController) GetMyUsages(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	rows, err := c.service.ListUsages(&partner.ID, nil, "")
	return respond(ctx, rows, err, "Success get usages")
}
func (c *PartnerController) GetMyAllocations(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	rows, err := c.service.ListAllocations(&partner.ID, "", "")
	return respond(ctx, rows, err, "Success get allocations")
}
func (c *PartnerController) GetMyDocuments(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	// A partner only sees published documents shared with partners that either
	// belong to that partner or apply to every partner.
	visible := true
	rows, err := c.service.ListLegalDocuments(repositories.LegalDocumentFilter{
		PartnerVisible:    &visible,
		PartnerID:         &partner.ID,
		IncludeUnassigned: true,
	})
	return respond(ctx, rows, err, "Success get documents")
}
func (c *PartnerController) GetMyDocument(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	row, err := c.service.GetLegalDocument(id, &partner.ID)
	return respond(ctx, row, err, "Success get document")
}
func (c *PartnerController) GetMyAging(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	rows, err := c.service.AgingReport(&partner.ID)
	return respond(ctx, rows, err, "Success get aging report")
}
func (c *PartnerController) GetMyInvoiceHistory(ctx *fiber.Ctx) error {
	partner, err := c.portalPartner(ctx)
	if err != nil {
		return respond(ctx, nil, err, "")
	}
	id, err := parseID(ctx, "id")
	if err != nil {
		return badRequest(ctx, err)
	}
	if _, err := c.service.GetInvoice(id, &partner.ID); err != nil {
		return respond(ctx, nil, err, "")
	}
	rows, err := c.service.ListInvoiceHistory(id)
	return respond(ctx, rows, err, "Success get invoice history")
}

// BrevoWebhook verifies a shared secret before recording a transactional event.
func (c *PartnerController) BrevoWebhook(ctx *fiber.Ctx) error {
	secret := strings.TrimSpace(os.Getenv("BREVO_WEBHOOK_SECRET"))
	if secret == "" {
		return ctx.Status(503).JSON(fiber.Map{"success": false, "message": "webhook not configured"})
	}
	if subtle.ConstantTimeCompare([]byte(ctx.Get("X-Brevo-Secret")), []byte(secret)) != 1 {
		return ctx.Status(401).JSON(fiber.Map{"success": false, "message": "invalid webhook secret"})
	}
	var in dto.BrevoWebhookEvent
	if err := ctx.BodyParser(&in); err != nil {
		return badRequest(ctx, err)
	}
	if err := c.service.RecordBrevoEvent(in); err != nil {
		return respond(ctx, nil, err, "")
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "event recorded"})
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
