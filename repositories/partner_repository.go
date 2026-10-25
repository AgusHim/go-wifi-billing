package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PartnerRepository interface {
	ListPartners() ([]models.Partner, error)
	FindPartner(id uuid.UUID) (*models.Partner, error)
	FindPartnerByCode(code string) (*models.Partner, error)
	CreatePartner(*models.Partner) error
	SavePartner(*models.Partner) error
	DeletePartner(uuid.UUID) error
	CreateMember(*models.PartnerMember) error
	ListMembers(partnerID uuid.UUID) ([]models.PartnerMember, error)
	ListAllMembers() ([]models.PartnerMember, error)
	FindMemberByPartnerAndUser(partnerID, userID uuid.UUID) (*models.PartnerMember, error)
	CountMembers(partnerID uuid.UUID) (int64, error)
	FindPartnerByUser(userID uuid.UUID) (*models.Partner, error)
	ListServices(partnerID uuid.UUID) ([]models.PartnerBandwidthService, error)
	CreateService(*models.PartnerBandwidthService) error
	FindService(id uuid.UUID) (*models.PartnerBandwidthService, error)
	SaveService(*models.PartnerBandwidthService) error
	ListServicesDueForBilling() ([]models.PartnerBandwidthService, error)
	CreateInvoice(*models.PartnerInvoice) error
	FindInvoice(id uuid.UUID) (*models.PartnerInvoice, error)
	ListInvoices(partnerID *uuid.UUID) ([]models.PartnerInvoice, error)
	ListInvoicesFiltered(partnerID *uuid.UUID, status string, overdueOnly bool) ([]models.PartnerInvoice, error)
	ListInvoicesDueForReminder() ([]models.PartnerInvoice, error)
	ListOverdueInvoices() ([]models.PartnerInvoice, error)
	SaveInvoice(*models.PartnerInvoice) error
	CreatePayment(*models.PartnerPayment) error
	FindPayment(id uuid.UUID) (*models.PartnerPayment, error)
	FindPaymentByIdempotencyKey(key string) (*models.PartnerPayment, error)
	SavePayment(*models.PartnerPayment) error

	// Bandwidth usage
	CreateUsage(*models.PartnerBandwidthUsage) error
	SaveUsage(*models.PartnerBandwidthUsage) error
	FindUsage(id uuid.UUID) (*models.PartnerBandwidthUsage, error)
	FindUsageByPeriod(serviceID uuid.UUID, year, month int) (*models.PartnerBandwidthUsage, error)
	ListUsages(partnerID *uuid.UUID, serviceID *uuid.UUID, status string) ([]models.PartnerBandwidthUsage, error)

	// Inventory allocation (loan / sale)
	CreateAllocationWithStock(*models.PartnerInventoryAllocation) error
	FindAllocation(id uuid.UUID) (*models.PartnerInventoryAllocation, error)
	ListAllocations(partnerID *uuid.UUID, status, transactionType string) ([]models.PartnerInventoryAllocation, error)
	SaveAllocation(*models.PartnerInventoryAllocation) error
	ReturnAllocation(allocation *models.PartnerInventoryAllocation, toLocationID uuid.UUID, notes string) error
	LinkAllocationToInvoiceLine(allocationID, invoiceLineID uuid.UUID) error

	// Invoice audit + source references
	CreateStatusHistory(*models.PartnerInvoiceStatusHistory) error
	ListStatusHistory(invoiceID uuid.UUID) ([]models.PartnerInvoiceStatusHistory, error)
	FindInvoiceLineBySource(sourceRefType string, sourceRefID uuid.UUID, year, month *int) (*models.PartnerInvoiceLine, error)

	// Notifications (Brevo)
	CreateNotification(*models.PartnerInvoiceNotification) error
	SaveNotification(*models.PartnerInvoiceNotification) error
	FindNotificationByIdempotencyKey(key string) (*models.PartnerInvoiceNotification, error)
	FindNotificationByMessageID(messageID string) (*models.PartnerInvoiceNotification, error)
	ListNotifications(invoiceID uuid.UUID) ([]models.PartnerInvoiceNotification, error)

	// Legal documents
	CreateDocument(*models.LegalDocument) error
	SaveDocument(*models.LegalDocument) error
	FindDocument(id uuid.UUID) (*models.LegalDocument, error)
	ListDocuments(partnerVisible *bool, category string) ([]models.LegalDocument, error)
	DeleteDocument(id uuid.UUID) error
}

type partnerRepository struct{ db *gorm.DB }

func NewPartnerRepository(db *gorm.DB) PartnerRepository { return &partnerRepository{db: db} }
func (r *partnerRepository) ListPartners() ([]models.Partner, error) {
	var rows []models.Partner
	err := r.db.Order("name").Find(&rows).Error
	return rows, err
}
func (r *partnerRepository) FindPartner(id uuid.UUID) (*models.Partner, error) {
	var row models.Partner
	err := r.db.Preload("Members.User").First(&row, "id = ?", id).Error
	return &row, err
}
func (r *partnerRepository) CreatePartner(row *models.Partner) error { return r.db.Create(row).Error }
func (r *partnerRepository) SavePartner(row *models.Partner) error {
	return r.db.Omit(clause.Associations).Save(row).Error
}
func (r *partnerRepository) CreateMember(row *models.PartnerMember) error {
	return r.db.Create(row).Error
}
func (r *partnerRepository) FindPartnerByUser(userID uuid.UUID) (*models.Partner, error) {
	var member models.PartnerMember
	err := r.db.Preload("Partner").Where("user_id = ? AND status = ?", userID, "active").First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member.Partner, nil
}
func (r *partnerRepository) ListServices(partnerID uuid.UUID) ([]models.PartnerBandwidthService, error) {
	var rows []models.PartnerBandwidthService
	err := r.db.Where("partner_id = ?", partnerID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}
func (r *partnerRepository) CreateService(row *models.PartnerBandwidthService) error {
	return r.db.Create(row).Error
}
func (r *partnerRepository) CreateInvoice(row *models.PartnerInvoice) error {
	return r.db.Create(row).Error
}
func (r *partnerRepository) FindInvoice(id uuid.UUID) (*models.PartnerInvoice, error) {
	var row models.PartnerInvoice
	err := r.db.Preload("Partner").Preload("Lines").Preload("Payments").Preload("StatusHistory").First(&row, "id = ?", id).Error
	return &row, err
}
func (r *partnerRepository) ListInvoices(partnerID *uuid.UUID) ([]models.PartnerInvoice, error) {
	var rows []models.PartnerInvoice
	q := r.db.Preload("Lines")
	if partnerID != nil {
		q = q.Where("partner_id = ?", *partnerID)
	}
	err := q.Order("due_date DESC").Find(&rows).Error
	return rows, err
}
func (r *partnerRepository) SaveInvoice(row *models.PartnerInvoice) error {
	return r.db.Omit(clause.Associations).Save(row).Error
}
func (r *partnerRepository) CreatePayment(row *models.PartnerPayment) error {
	return r.db.Create(row).Error
}
func (r *partnerRepository) FindPayment(id uuid.UUID) (*models.PartnerPayment, error) {
	var row models.PartnerPayment
	err := r.db.First(&row, "id = ?", id).Error
	return &row, err
}
func (r *partnerRepository) SavePayment(row *models.PartnerPayment) error {
	return r.db.Omit(clause.Associations).Save(row).Error
}
