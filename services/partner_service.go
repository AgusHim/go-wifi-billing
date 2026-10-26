package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PartnerService interface {
	ListPartners() ([]models.Partner, error)
	GetPartner(uuid.UUID) (*models.Partner, error)
	CreatePartner(dto.PartnerRequest) (*PartnerCreateResult, error)
	UpdatePartner(uuid.UUID, dto.PartnerRequest) (*models.Partner, error)
	CreatePartnerAccount(uuid.UUID, dto.PartnerAccountRequest, uuid.UUID) (*PartnerAccountResult, error)
	ListPartnerAccounts(uuid.UUID) ([]PartnerAccountResult, error)
	ListAllPartnerAccounts() ([]PartnerAccountResult, error)
	AddMember(uuid.UUID, dto.PartnerMemberRequest) (*models.PartnerMember, error)
	CreateBandwidthService(uuid.UUID, dto.PartnerBandwidthServiceRequest) (*models.PartnerBandwidthService, error)
	UpdateBandwidthService(uuid.UUID, dto.PartnerBandwidthServiceRequest) (*models.PartnerBandwidthService, error)
	ListServices(uuid.UUID) ([]models.PartnerBandwidthService, error)
	CreateInvoice(dto.PartnerInvoiceRequest, uuid.UUID) (*models.PartnerInvoice, error)
	GenerateInvoiceForService(uuid.UUID, dto.PartnerInvoiceFromServiceRequest, uuid.UUID) (*models.PartnerInvoice, error)
	IssueInvoice(uuid.UUID, uuid.UUID) (*models.PartnerInvoice, error)
	VoidInvoice(uuid.UUID, string, uuid.UUID) (*models.PartnerInvoice, error)
	ListInvoices(*uuid.UUID) ([]models.PartnerInvoice, error)
	ListInvoicesFiltered(repositories.PartnerInvoiceFilter) ([]models.PartnerInvoice, error)
	GetInvoice(uuid.UUID, *uuid.UUID) (*models.PartnerInvoice, error)
	ListInvoiceHistory(uuid.UUID) ([]models.PartnerInvoiceStatusHistory, error)
	ListInvoiceNotifications(uuid.UUID) ([]models.PartnerInvoiceNotification, error)
	ResendInvoiceNotification(uuid.UUID, uuid.UUID) (*models.PartnerInvoiceNotification, error)
	SubmitPayment(uuid.UUID, dto.PartnerPaymentRequest, uuid.UUID) (*models.PartnerPayment, error)
	VerifyPayment(uuid.UUID, dto.PartnerPaymentVerificationRequest, uuid.UUID) (*models.PartnerPayment, error)
	RecordPartnerPayment(uuid.UUID, dto.PartnerStaffPaymentRequest, uuid.UUID) (*models.PartnerPayment, error)
	PartnerForUser(uuid.UUID) (*models.Partner, error)

	RecordUsage(uuid.UUID, dto.PartnerBandwidthUsageRequest, uuid.UUID) (*models.PartnerBandwidthUsage, error)
	ReviewUsage(uuid.UUID, dto.PartnerBandwidthUsageApprovalRequest, uuid.UUID) (*models.PartnerBandwidthUsage, error)
	ListUsages(*uuid.UUID, *uuid.UUID, string) ([]models.PartnerBandwidthUsage, error)

	CreateAllocation(uuid.UUID, dto.PartnerAllocationRequest, uuid.UUID) (*models.PartnerInventoryAllocation, error)
	ListAllocations(*uuid.UUID, string, string) ([]models.PartnerInventoryAllocation, error)
	ReturnAllocation(uuid.UUID, dto.PartnerAllocationReturnRequest, uuid.UUID) (*models.PartnerInventoryAllocation, error)

	AgingReport(*uuid.UUID) ([]PartnerAgingRow, error)
	SendDueReminders() (int, error)
	MarkOverdueInvoices() (int, error)
	GenerateRecurringDrafts() (int, error)
	StartScheduler()

	CreateLegalDocument(dto.LegalDocumentRequest, uuid.UUID) (*models.LegalDocument, error)
	UpdateLegalDocument(uuid.UUID, dto.LegalDocumentRequest) (*models.LegalDocument, error)
	ListLegalDocuments(repositories.LegalDocumentFilter) ([]models.LegalDocument, error)
	GetLegalDocument(uuid.UUID, *uuid.UUID) (*models.LegalDocument, error)
	DeleteLegalDocument(uuid.UUID) error

	RecordBrevoEvent(dto.BrevoWebhookEvent) error
}

// PartnerAgingRow summarizes outstanding receivables per partner for the
// receivables aging report.
type PartnerAgingRow struct {
	PartnerID    uuid.UUID `json:"partner_id"`
	PartnerName  string    `json:"partner_name"`
	Current      int       `json:"current"`
	Days1To30    int       `json:"days_1_30"`
	Days31To60   int       `json:"days_31_60"`
	Days61To90   int       `json:"days_61_90"`
	Over90       int       `json:"over_90"`
	TotalDue     int       `json:"total_due"`
	InvoiceCount int       `json:"invoice_count"`
}

// PartnerCreateResult bundles a newly created partner with the optional portal
// login account created in the same request.
type PartnerCreateResult struct {
	Partner *models.Partner       `json:"partner"`
	Account *PartnerAccountResult `json:"account,omitempty"`
}

// PartnerAccountResult describes a partner portal login account. The temporary
// password is only populated on creation when it was generated server-side.
type PartnerAccountResult struct {
	MemberID          uuid.UUID `json:"member_id"`
	UserID            uuid.UUID `json:"user_id"`
	PartnerID         uuid.UUID `json:"partner_id"`
	PartnerName       string    `json:"partner_name"`
	PartnerCode       string    `json:"partner_code"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	Phone             string    `json:"phone"`
	Role              string    `json:"role"`
	Status            string    `json:"status"`
	IsActive          bool      `json:"is_active"`
	TemporaryPassword string    `json:"temporary_password,omitempty"`
	InviteSent        bool      `json:"invite_sent"`
}

type partnerService struct {
	repo          repositories.PartnerRepository
	inventoryRepo repositories.InventoryRepository
	userRepo      repositories.UserRepository
	db            *gorm.DB
	brevo         BrevoService
}

func NewPartnerService(repo repositories.PartnerRepository, inventoryRepo repositories.InventoryRepository, userRepo repositories.UserRepository, db *gorm.DB, brevo BrevoService) PartnerService {
	return &partnerService{repo: repo, inventoryRepo: inventoryRepo, userRepo: userRepo, db: db, brevo: brevo}
}

func (s *partnerService) ListPartners() ([]models.Partner, error) { return s.repo.ListPartners() }

func (s *partnerService) GetPartner(id uuid.UUID) (*models.Partner, error) {
	row, err := s.repo.FindPartner(id)
	if err != nil {
		return nil, errors.New("partner not found")
	}
	return row, nil
}

func (s *partnerService) CreatePartner(in dto.PartnerRequest) (*PartnerCreateResult, error) {
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.BillingEmail) == "" {
		return nil, errors.New("code, name, and billing_email are required")
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	// A duplicate code is a user input problem, not a server error.
	if _, err := s.repo.FindPartnerByCode(code); err == nil {
		return nil, errors.New("partner code is already used")
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	// Validate the optional login account before persisting the partner so a
	// rejected account never leaves a half-created record.
	if in.Account != nil {
		if err := validatePartnerAccount(*in.Account); err != nil {
			return nil, err
		}
	}
	row := &models.Partner{Code: code, Name: strings.TrimSpace(in.Name), BillingEmail: strings.TrimSpace(in.BillingEmail), Phone: strings.TrimSpace(in.Phone), ContactName: strings.TrimSpace(in.ContactName), Address: strings.TrimSpace(in.Address), TaxNumber: strings.TrimSpace(in.TaxNumber), IsActive: active}
	if err := s.repo.CreatePartner(row); err != nil {
		return nil, err
	}
	result := &PartnerCreateResult{Partner: row}
	if in.Account != nil {
		account, err := s.createPartnerAccount(*row, *in.Account, uuid.Nil)
		if err != nil {
			// Roll back the partner so staff can retry with corrected account data.
			_ = s.repo.DeletePartner(row.ID)
			return nil, err
		}
		result.Account = account
	}
	return result, nil
}

func (s *partnerService) UpdatePartner(id uuid.UUID, in dto.PartnerRequest) (*models.Partner, error) {
	row, err := s.repo.FindPartner(id)
	if err != nil {
		return nil, errors.New("partner not found")
	}
	if code := strings.ToUpper(strings.TrimSpace(in.Code)); code != "" {
		row.Code = code
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		row.Name = name
	}
	if email := strings.TrimSpace(in.BillingEmail); email != "" {
		row.BillingEmail = email
	}
	row.Phone = strings.TrimSpace(in.Phone)
	row.ContactName = strings.TrimSpace(in.ContactName)
	row.Address = strings.TrimSpace(in.Address)
	row.TaxNumber = strings.TrimSpace(in.TaxNumber)
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := s.repo.SavePartner(row); err != nil {
		return nil, err
	}
	return row, nil
}
func (s *partnerService) AddMember(partnerID uuid.UUID, in dto.PartnerMemberRequest) (*models.PartnerMember, error) {
	userID, err := uuid.Parse(strings.TrimSpace(in.UserID))
	if err != nil {
		return nil, errors.New("invalid user_id")
	}
	if _, err = s.repo.FindPartner(partnerID); err != nil {
		return nil, errors.New("partner not found")
	}
	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "member"
	}
	if role != "owner" && role != "member" {
		return nil, errors.New("role must be owner or member")
	}
	row := &models.PartnerMember{PartnerID: partnerID, UserID: userID, Role: role, Status: "active"}
	if err := s.repo.CreateMember(row); err != nil {
		return nil, err
	}
	return row, nil
}
func (s *partnerService) CreateBandwidthService(partnerID uuid.UUID, in dto.PartnerBandwidthServiceRequest) (*models.PartnerBandwidthService, error) {
	if _, err := s.repo.FindPartner(partnerID); err != nil {
		return nil, errors.New("partner not found")
	}
	mode := strings.TrimSpace(in.BillingMode)
	if mode != "fixed" && mode != "usage" && mode != "manual" {
		return nil, errors.New("billing_mode must be fixed, usage, or manual")
	}
	if strings.TrimSpace(in.Name) == "" || in.Rate < 0 {
		return nil, errors.New("valid name and non-negative rate are required")
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	}
	row := &models.PartnerBandwidthService{PartnerID: partnerID, Name: strings.TrimSpace(in.Name), Capacity: strings.TrimSpace(in.Capacity), BillingMode: mode, Rate: in.Rate, DueDay: in.DueDay, StartsAt: in.StartsAt, EndsAt: in.EndsAt, Status: status}
	if err := s.repo.CreateService(row); err != nil {
		return nil, err
	}
	return row, nil
}
func (s *partnerService) ListServices(partnerID uuid.UUID) ([]models.PartnerBandwidthService, error) {
	return s.repo.ListServices(partnerID)
}

func (s *partnerService) UpdateBandwidthService(id uuid.UUID, in dto.PartnerBandwidthServiceRequest) (*models.PartnerBandwidthService, error) {
	row, err := s.repo.FindService(id)
	if err != nil {
		return nil, errors.New("service not found")
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		row.Name = name
	}
	if mode := strings.TrimSpace(in.BillingMode); mode != "" {
		if mode != "fixed" && mode != "usage" && mode != "manual" {
			return nil, errors.New("billing_mode must be fixed, usage, or manual")
		}
		row.BillingMode = mode
	}
	if in.Rate < 0 {
		return nil, errors.New("rate must not be negative")
	}
	if in.Rate > 0 {
		row.Rate = in.Rate
	}
	row.Capacity = strings.TrimSpace(in.Capacity)
	row.DueDay = in.DueDay
	if in.StartsAt != nil {
		row.StartsAt = in.StartsAt
	}
	row.EndsAt = in.EndsAt
	if status := strings.TrimSpace(in.Status); status != "" {
		row.Status = status
	}
	if err := s.repo.SaveService(row); err != nil {
		return nil, err
	}
	return row, nil
}
func (s *partnerService) CreateInvoice(in dto.PartnerInvoiceRequest, actorID uuid.UUID) (*models.PartnerInvoice, error) {
	partnerID, err := uuid.Parse(strings.TrimSpace(in.PartnerID))
	if err != nil {
		return nil, errors.New("invalid partner_id")
	}
	partner, err := s.repo.FindPartner(partnerID)
	if err != nil {
		return nil, errors.New("partner not found")
	}
	if in.DueDate.IsZero() || len(in.Lines) == 0 {
		return nil, errors.New("due_date and at least one line are required")
	}
	invoice := &models.PartnerInvoice{Number: fmt.Sprintf("DRAFT-%s", uuid.NewString()[:8]), PartnerID: partnerID, DueDate: in.DueDate, Status: "draft", IssuerName: "Cantika.net", PartnerNameSnapshot: partner.Name, PartnerEmailSnapshot: partner.BillingEmail}
	var actor *uuid.UUID
	if actorID != uuid.Nil {
		actor = &actorID
		invoice.CreatedBy = &actorID
	}
	allocationLinks := map[uuid.UUID]int{}
	for _, item := range in.Lines {
		if item.Quantity <= 0 || item.UnitPrice < 0 || item.TaxAmount < 0 || strings.TrimSpace(item.Description) == "" {
			return nil, errors.New("invoice lines require description, positive quantity, and valid amounts")
		}
		typ := strings.TrimSpace(item.LineType)
		if typ != "bandwidth" && typ != "inventory" && typ != "other" {
			return nil, errors.New("line_type must be bandwidth, inventory, or other")
		}
		line := models.PartnerInvoiceLine{Description: strings.TrimSpace(item.Description), LineType: typ, Quantity: item.Quantity, Unit: strings.TrimSpace(item.Unit), UnitPrice: item.UnitPrice, TaxAmount: item.TaxAmount}
		line.Total = int(item.Quantity*float64(item.UnitPrice)) + item.TaxAmount
		// Bandwidth lines carry a service period so the same period cannot be
		// billed twice.
		if typ == "bandwidth" {
			if item.ServiceID == "" || item.PeriodYear == nil || item.PeriodMonth == nil {
				return nil, errors.New("bandwidth lines require service_id, period_year, and period_month")
			}
			serviceID, err := uuid.Parse(strings.TrimSpace(item.ServiceID))
			if err != nil {
				return nil, errors.New("invalid service_id on invoice line")
			}
			service, err := s.repo.FindService(serviceID)
			if err != nil || service.PartnerID != partnerID {
				return nil, errors.New("bandwidth service not found for this partner")
			}
			if *item.PeriodMonth < 1 || *item.PeriodMonth > 12 {
				return nil, errors.New("period_month must be between 1 and 12")
			}
			if _, err := s.repo.FindInvoiceLineBySource("bandwidth", serviceID, item.PeriodYear, item.PeriodMonth); err == nil {
				return nil, errors.New("this service period has already been billed")
			}
			line.PeriodYear = item.PeriodYear
			line.PeriodMonth = item.PeriodMonth
			line.SourceRefType = "bandwidth"
			line.SourceRefID = &serviceID
		}
		if typ == "inventory" && strings.TrimSpace(item.AllocationID) != "" {
			allocationID, err := uuid.Parse(strings.TrimSpace(item.AllocationID))
			if err != nil {
				return nil, errors.New("invalid allocation_id on invoice line")
			}
			allocation, err := s.repo.FindAllocation(allocationID)
			if err != nil || allocation.PartnerID != partnerID {
				return nil, errors.New("inventory allocation not found for this partner")
			}
			if allocation.TransactionType != "sale" {
				return nil, errors.New("only sold inventory can be billed")
			}
			if allocation.InvoiceLineID != nil {
				return nil, errors.New("this allocation has already been billed")
			}
			line.SourceRefType = "allocation"
			line.SourceRefID = &allocationID
			allocationLinks[allocationID] = len(invoice.Lines)
		}
		invoice.Subtotal += int(item.Quantity * float64(item.UnitPrice))
		invoice.TaxAmount += item.TaxAmount
		invoice.Total += line.Total
		invoice.Lines = append(invoice.Lines, line)
	}
	invoice.BalanceDue = invoice.Total
	if err := s.repo.CreateInvoice(invoice); err != nil {
		return nil, err
	}
	for allocationID, index := range allocationLinks {
		if index < len(invoice.Lines) {
			_ = s.repo.LinkAllocationToInvoiceLine(allocationID, invoice.Lines[index].ID)
		}
	}
	s.recordStatus(invoice.ID, "", "draft", "invoice created", actor)
	return invoice, nil
}
func (s *partnerService) IssueInvoice(id uuid.UUID, actorID uuid.UUID) (*models.PartnerInvoice, error) {
	invoice, err := s.repo.FindInvoice(id)
	if err != nil {
		return nil, errors.New("invoice not found")
	}
	if invoice.Status != "draft" {
		return nil, errors.New("only draft invoice can be issued")
	}
	now := time.Now()
	invoice.IssuedAt = &now
	invoice.Status = "issued"
	invoice.Number = fmt.Sprintf("B2B-%s-%05d", now.Format("20060102"), now.Nanosecond()%100000)
	if err := s.repo.SaveInvoice(invoice); err != nil {
		return nil, err
	}
	s.recordStatus(invoice.ID, "draft", "issued", "invoice issued to partner", &actorID)
	go s.sendInvoiceEmail(*invoice, "invoice", "issued:"+invoice.ID.String())
	return invoice, nil
}

func (s *partnerService) VoidInvoice(id uuid.UUID, reason string, actorID uuid.UUID) (*models.PartnerInvoice, error) {
	invoice, err := s.repo.FindInvoice(id)
	if err != nil {
		return nil, errors.New("invoice not found")
	}
	if invoice.Status == "paid" {
		return nil, errors.New("a paid invoice cannot be voided")
	}
	if invoice.Status == "void" {
		return nil, errors.New("invoice is already void")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("void reason is required")
	}
	previous := invoice.Status
	invoice.Status = "void"
	invoice.BalanceDue = 0
	if err := s.repo.SaveInvoice(invoice); err != nil {
		return nil, err
	}
	s.recordStatus(invoice.ID, previous, "void", strings.TrimSpace(reason), &actorID)
	return invoice, nil
}

func (s *partnerService) recordStatus(invoiceID uuid.UUID, from, to, note string, actor *uuid.UUID) {
	_ = s.repo.CreateStatusHistory(&models.PartnerInvoiceStatusHistory{InvoiceID: invoiceID, FromStatus: from, ToStatus: to, Note: note, ChangedBy: actor})
}

// sendInvoiceEmail delivers an invoice or reminder through Brevo and always
// writes an audit row. The idempotency key guarantees one job cannot send the
// same email twice; explicit resends use a distinct key.
func (s *partnerService) sendInvoiceEmail(invoice models.PartnerInvoice, kind, idempotencyKey string) *models.PartnerInvoiceNotification {
	if existing, err := s.repo.FindNotificationByIdempotencyKey(idempotencyKey); err == nil && existing != nil {
		return existing
	}
	note := models.PartnerInvoiceNotification{InvoiceID: invoice.ID, Recipient: invoice.PartnerEmailSnapshot, Kind: kind, Status: "skipped", IdempotencyKey: idempotencyKey}
	if s.brevo != nil {
		var messageID, sendErr string
		if kind == "reminder" {
			messageID, sendErr = s.brevo.SendReminder(note.Recipient, invoice.Number, invoice.Total, invoice.BalanceDue, invoice.DueDate.Format("2006-01-02"))
		} else {
			messageID, sendErr = s.brevo.SendInvoice(note.Recipient, invoice.Number, invoice.Total, invoice.BalanceDue)
		}
		if sendErr == "" {
			note.Status = "sent"
			note.BrevoMessageID = messageID
		} else if sendErr != "Brevo is not configured" {
			note.Status = "failed"
			note.ErrorMessage = sendErr
		}
	}
	_ = s.repo.CreateNotification(&note)
	return &note
}
func (s *partnerService) ListInvoices(partnerID *uuid.UUID) ([]models.PartnerInvoice, error) {
	return s.repo.ListInvoices(partnerID)
}

func (s *partnerService) ListInvoicesFiltered(filter repositories.PartnerInvoiceFilter) ([]models.PartnerInvoice, error) {
	return s.repo.ListInvoicesFiltered(filter)
}

// RecordPartnerPayment lets staff settle an invoice on behalf of a partner, for
// example when cash or a bank transfer is received at the office. The record is
// created and, unless auto_verify is disabled, settled in the same transaction
// so the receivable can never be reduced without an auditable payment row.
func (s *partnerService) RecordPartnerPayment(partnerID uuid.UUID, in dto.PartnerStaffPaymentRequest, actorID uuid.UUID) (*models.PartnerPayment, error) {
	invoiceID, err := uuid.Parse(strings.TrimSpace(in.InvoiceID))
	if err != nil {
		return nil, errors.New("invalid invoice_id")
	}
	if in.Amount <= 0 {
		return nil, errors.New("payment amount must be greater than zero")
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	if key != "" {
		if existing, err := s.repo.FindPaymentByIdempotencyKey(key); err == nil && existing != nil {
			return existing, nil
		}
	}
	// The column is unique, so a caller that sends no key still needs a value.
	if key == "" {
		key = "auto:" + uuid.NewString()
	}
	autoVerify := true
	if in.AutoVerify != nil {
		autoVerify = *in.AutoVerify
	}

	var result models.PartnerPayment
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var invoice models.PartnerInvoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", invoiceID).Error; err != nil {
			return errors.New("invoice not found")
		}
		// A payment must never be attached to another partner's invoice.
		if invoice.PartnerID != partnerID {
			return errors.New("invoice not found")
		}
		if invoice.Status != "issued" && invoice.Status != "partially_paid" && invoice.Status != "overdue" {
			return errors.New("invoice is not payable")
		}
		if in.Amount > invoice.BalanceDue {
			return errors.New("payment amount must not exceed balance due")
		}

		now := time.Now()
		payment := models.PartnerPayment{
			InvoiceID: invoice.ID, SubmittedByID: &actorID, SubmittedAt: now,
			Amount: in.Amount, Method: strings.TrimSpace(in.Method), Reference: strings.TrimSpace(in.Reference),
			ProofURL: strings.TrimSpace(in.ProofURL), IdempotencyKey: key,
		}
		if autoVerify {
			payment.Status = "verified"
			payment.VerifiedByID = &actorID
			payment.VerifiedAt = &now

			previousStatus := invoice.Status
			invoice.VerifiedPaidAmount += in.Amount
			invoice.BalanceDue -= in.Amount
			if invoice.BalanceDue <= 0 {
				invoice.BalanceDue = 0
				invoice.Status = "paid"
			} else {
				invoice.Status = "partially_paid"
			}
			if err := tx.Save(&invoice).Error; err != nil {
				return err
			}
			if previousStatus != invoice.Status {
				history := models.PartnerInvoiceStatusHistory{
					InvoiceID: invoice.ID, FromStatus: previousStatus, ToStatus: invoice.Status,
					Note: "payment recorded by staff", ChangedBy: &actorID,
				}
				if err := tx.Create(&history).Error; err != nil {
					return err
				}
			}
		} else {
			payment.Status = "pending"
		}
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}
		result = payment
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (s *partnerService) GetInvoice(id uuid.UUID, partnerID *uuid.UUID) (*models.PartnerInvoice, error) {
	invoice, err := s.repo.FindInvoice(id)
	if err != nil {
		return nil, errors.New("invoice not found")
	}
	if partnerID != nil && invoice.PartnerID != *partnerID {
		return nil, errors.New("invoice not found")
	}
	return invoice, nil
}
func (s *partnerService) SubmitPayment(invoiceID uuid.UUID, in dto.PartnerPaymentRequest, submittedBy uuid.UUID) (*models.PartnerPayment, error) {
	invoice, err := s.repo.FindInvoice(invoiceID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}
	if invoice.Status != "issued" && invoice.Status != "partially_paid" && invoice.Status != "overdue" {
		return nil, errors.New("invoice is not payable")
	}
	if in.Amount <= 0 || in.Amount > invoice.BalanceDue {
		return nil, errors.New("payment amount must not exceed balance due")
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	// Retrying the same submission must not create a duplicate record.
	if key != "" {
		if existing, err := s.repo.FindPaymentByIdempotencyKey(key); err == nil && existing != nil {
			return existing, nil
		}
	}
	// The column is unique, so a caller that sends no key still needs a value.
	if key == "" {
		key = "auto:" + uuid.NewString()
	}
	row := &models.PartnerPayment{InvoiceID: invoiceID, SubmittedByID: &submittedBy, SubmittedAt: time.Now(), Amount: in.Amount, Method: strings.TrimSpace(in.Method), Reference: strings.TrimSpace(in.Reference), ProofURL: strings.TrimSpace(in.ProofURL), Status: "pending", IdempotencyKey: key}
	if err := s.repo.CreatePayment(row); err != nil {
		if key != "" {
			if existing, findErr := s.repo.FindPaymentByIdempotencyKey(key); findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return row, nil
}

func (s *partnerService) VerifyPayment(paymentID uuid.UUID, in dto.PartnerPaymentVerificationRequest, verifiedBy uuid.UUID) (*models.PartnerPayment, error) {
	var result models.PartnerPayment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var payment models.PartnerPayment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&payment, "id = ?", paymentID).Error; err != nil {
			return errors.New("payment not found")
		}
		if payment.Status != "pending" {
			return errors.New("payment has already been reviewed")
		}
		var invoice models.PartnerInvoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", payment.InvoiceID).Error; err != nil {
			return errors.New("invoice not found")
		}
		now := time.Now()
		payment.VerifiedByID = &verifiedBy
		payment.VerifiedAt = &now
		if !in.Approve {
			payment.Status = "rejected"
			payment.RejectionReason = strings.TrimSpace(in.RejectionReason)
			if payment.RejectionReason == "" {
				return errors.New("rejection_reason is required")
			}
		} else {
			// Only a verified payment reduces the receivable, and never below zero.
			if payment.Amount > invoice.BalanceDue {
				return errors.New("payment amount exceeds current balance")
			}
			payment.Status = "verified"
			invoice.VerifiedPaidAmount += payment.Amount
			invoice.BalanceDue -= payment.Amount
			previousStatus := invoice.Status
			if invoice.BalanceDue <= 0 {
				invoice.BalanceDue = 0
				invoice.Status = "paid"
			} else {
				invoice.Status = "partially_paid"
			}
			if err := tx.Save(&invoice).Error; err != nil {
				return err
			}
			if previousStatus != invoice.Status {
				history := models.PartnerInvoiceStatusHistory{InvoiceID: invoice.ID, FromStatus: previousStatus, ToStatus: invoice.Status, Note: "payment verified", ChangedBy: &verifiedBy}
				if err := tx.Create(&history).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Save(&payment).Error; err != nil {
			return err
		}
		result = payment
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (s *partnerService) PartnerForUser(userID uuid.UUID) (*models.Partner, error) {
	partner, err := s.repo.FindPartnerByUser(userID)
	if err != nil {
		return nil, errors.New("partner membership not found")
	}
	return partner, nil
}
