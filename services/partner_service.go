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
	CreatePartner(dto.PartnerRequest) (*models.Partner, error)
	AddMember(uuid.UUID, dto.PartnerMemberRequest) (*models.PartnerMember, error)
	CreateBandwidthService(uuid.UUID, dto.PartnerBandwidthServiceRequest) (*models.PartnerBandwidthService, error)
	ListServices(uuid.UUID) ([]models.PartnerBandwidthService, error)
	CreateInvoice(dto.PartnerInvoiceRequest, uuid.UUID) (*models.PartnerInvoice, error)
	IssueInvoice(uuid.UUID) (*models.PartnerInvoice, error)
	ListInvoices(*uuid.UUID) ([]models.PartnerInvoice, error)
	GetInvoice(uuid.UUID, *uuid.UUID) (*models.PartnerInvoice, error)
	SubmitPayment(uuid.UUID, dto.PartnerPaymentRequest, uuid.UUID) (*models.PartnerPayment, error)
	VerifyPayment(uuid.UUID, dto.PartnerPaymentVerificationRequest, uuid.UUID) (*models.PartnerPayment, error)
	PartnerForUser(uuid.UUID) (*models.Partner, error)
}

type partnerService struct {
	repo  repositories.PartnerRepository
	db    *gorm.DB
	brevo BrevoService
}

func NewPartnerService(repo repositories.PartnerRepository, db *gorm.DB, brevo BrevoService) PartnerService {
	return &partnerService{repo: repo, db: db, brevo: brevo}
}

func (s *partnerService) ListPartners() ([]models.Partner, error) { return s.repo.ListPartners() }
func (s *partnerService) CreatePartner(in dto.PartnerRequest) (*models.Partner, error) {
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.BillingEmail) == "" {
		return nil, errors.New("code, name, and billing_email are required")
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	row := &models.Partner{Code: strings.ToUpper(strings.TrimSpace(in.Code)), Name: strings.TrimSpace(in.Name), BillingEmail: strings.TrimSpace(in.BillingEmail), Phone: strings.TrimSpace(in.Phone), ContactName: strings.TrimSpace(in.ContactName), Address: strings.TrimSpace(in.Address), TaxNumber: strings.TrimSpace(in.TaxNumber), IsActive: active}
	if err := s.repo.CreatePartner(row); err != nil {
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
	invoice := &models.PartnerInvoice{Number: fmt.Sprintf("DRAFT-%s", uuid.NewString()[:8]), PartnerID: partnerID, DueDate: in.DueDate, Status: "draft", IssuerName: "Cantika.net", PartnerNameSnapshot: partner.Name, PartnerEmailSnapshot: partner.BillingEmail, CreatedBy: &actorID}
	for _, item := range in.Lines {
		if item.Quantity <= 0 || item.UnitPrice < 0 || item.TaxAmount < 0 || strings.TrimSpace(item.Description) == "" {
			return nil, errors.New("invoice lines require description, positive quantity, and valid amounts")
		}
		typ := strings.TrimSpace(item.LineType)
		if typ != "bandwidth" && typ != "inventory" && typ != "other" {
			return nil, errors.New("line_type must be bandwidth, inventory, or other")
		}
		total := int(item.Quantity*float64(item.UnitPrice)) + item.TaxAmount
		invoice.Subtotal += int(item.Quantity * float64(item.UnitPrice))
		invoice.TaxAmount += item.TaxAmount
		invoice.Total += total
		invoice.Lines = append(invoice.Lines, models.PartnerInvoiceLine{Description: strings.TrimSpace(item.Description), LineType: typ, Quantity: item.Quantity, Unit: strings.TrimSpace(item.Unit), UnitPrice: item.UnitPrice, TaxAmount: item.TaxAmount, Total: total})
	}
	invoice.BalanceDue = invoice.Total
	if err := s.repo.CreateInvoice(invoice); err != nil {
		return nil, err
	}
	return invoice, nil
}
func (s *partnerService) IssueInvoice(id uuid.UUID) (*models.PartnerInvoice, error) {
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
	go s.sendInvoiceEmail(*invoice)
	return invoice, nil
}

func (s *partnerService) sendInvoiceEmail(invoice models.PartnerInvoice) {
	note := models.PartnerInvoiceNotification{InvoiceID: invoice.ID, Recipient: invoice.PartnerEmailSnapshot, Kind: "invoice", Status: "skipped"}
	if s.brevo != nil {
		messageID, sendErr := s.brevo.SendInvoice(note.Recipient, invoice.Number, invoice.Total, invoice.BalanceDue)
		if sendErr == "" {
			note.Status = "sent"
			note.BrevoMessageID = messageID
		} else if sendErr != "Brevo is not configured" {
			note.Status = "failed"
			note.ErrorMessage = sendErr
		}
	}
	_ = s.db.Create(&note).Error
}
func (s *partnerService) ListInvoices(partnerID *uuid.UUID) ([]models.PartnerInvoice, error) {
	return s.repo.ListInvoices(partnerID)
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
	row := &models.PartnerPayment{InvoiceID: invoiceID, SubmittedByID: &submittedBy, SubmittedAt: time.Now(), Amount: in.Amount, Method: strings.TrimSpace(in.Method), Reference: strings.TrimSpace(in.Reference), ProofURL: strings.TrimSpace(in.ProofURL), Status: "pending"}
	if err := s.repo.CreatePayment(row); err != nil {
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
			if payment.Amount > invoice.BalanceDue {
				return errors.New("payment amount exceeds current balance")
			}
			payment.Status = "verified"
			invoice.VerifiedPaidAmount += payment.Amount
			invoice.BalanceDue -= payment.Amount
			if invoice.BalanceDue == 0 {
				invoice.Status = "paid"
			} else {
				invoice.Status = "partially_paid"
			}
			if err := tx.Save(&invoice).Error; err != nil {
				return err
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
