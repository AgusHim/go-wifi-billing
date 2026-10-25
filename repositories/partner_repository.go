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
	CreatePartner(*models.Partner) error
	SavePartner(*models.Partner) error
	CreateMember(*models.PartnerMember) error
	FindPartnerByUser(userID uuid.UUID) (*models.Partner, error)
	ListServices(partnerID uuid.UUID) ([]models.PartnerBandwidthService, error)
	CreateService(*models.PartnerBandwidthService) error
	CreateInvoice(*models.PartnerInvoice) error
	FindInvoice(id uuid.UUID) (*models.PartnerInvoice, error)
	ListInvoices(partnerID *uuid.UUID) ([]models.PartnerInvoice, error)
	SaveInvoice(*models.PartnerInvoice) error
	CreatePayment(*models.PartnerPayment) error
	FindPayment(id uuid.UUID) (*models.PartnerPayment, error)
	SavePayment(*models.PartnerPayment) error
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
	err := r.db.Preload("Partner").Preload("Lines").Preload("Payments").First(&row, "id = ?", id).Error
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
