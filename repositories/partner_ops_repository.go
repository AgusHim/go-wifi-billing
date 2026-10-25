package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Partner members (portal login accounts) ────────────────────────────────

func (r *partnerRepository) DeletePartner(id uuid.UUID) error {
	return r.db.Delete(&models.Partner{}, "id = ?", id).Error
}

func (r *partnerRepository) FindPartnerByCode(code string) (*models.Partner, error) {
	var row models.Partner
	err := r.db.Where("code = ?", code).First(&row).Error
	return &row, err
}

func (r *partnerRepository) ListMembers(partnerID uuid.UUID) ([]models.PartnerMember, error) {
	var rows []models.PartnerMember
	err := r.db.Preload("User").Where("partner_id = ?", partnerID).Order("created_at").Find(&rows).Error
	return rows, err
}

// ListAllMembers backs the cross-partner portal account list.
func (r *partnerRepository) ListAllMembers() ([]models.PartnerMember, error) {
	var rows []models.PartnerMember
	err := r.db.Preload("User").Preload("Partner").Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) FindMemberByPartnerAndUser(partnerID, userID uuid.UUID) (*models.PartnerMember, error) {
	var row models.PartnerMember
	err := r.db.Where("partner_id = ? AND user_id = ?", partnerID, userID).First(&row).Error
	return &row, err
}

func (r *partnerRepository) CountMembers(partnerID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.PartnerMember{}).Where("partner_id = ?", partnerID).Count(&count).Error
	return count, err
}

// ── Bandwidth usage ────────────────────────────────────────────────────────

func (r *partnerRepository) CreateUsage(row *models.PartnerBandwidthUsage) error {
	return r.db.Create(row).Error
}

func (r *partnerRepository) SaveUsage(row *models.PartnerBandwidthUsage) error {
	return r.db.Omit(clause.Associations).Save(row).Error
}

func (r *partnerRepository) FindUsage(id uuid.UUID) (*models.PartnerBandwidthUsage, error) {
	var row models.PartnerBandwidthUsage
	err := r.db.Preload("Service").First(&row, "id = ?", id).Error
	return &row, err
}

func (r *partnerRepository) FindUsageByPeriod(serviceID uuid.UUID, year, month int) (*models.PartnerBandwidthUsage, error) {
	var row models.PartnerBandwidthUsage
	err := r.db.Where("service_id = ? AND period_year = ? AND period_month = ?", serviceID, year, month).First(&row).Error
	return &row, err
}

func (r *partnerRepository) ListUsages(partnerID *uuid.UUID, serviceID *uuid.UUID, status string) ([]models.PartnerBandwidthUsage, error) {
	var rows []models.PartnerBandwidthUsage
	query := r.db.Preload("Service")
	if partnerID != nil {
		query = query.Where("partner_id = ?", *partnerID)
	}
	if serviceID != nil {
		query = query.Where("service_id = ?", *serviceID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("period_year DESC, period_month DESC, created_at DESC").Find(&rows).Error
	return rows, err
}

// ── Inventory allocation ───────────────────────────────────────────────────

// CreateAllocationWithStock records a loan or sale and moves stock exactly
// once. Loans keep Cantika.net ownership (the allocation row is the asset
// record); sales also reduce on-hand stock and must later be linked to an
// invoice line by the invoice service.
func (r *partnerRepository) CreateAllocationWithStock(row *models.PartnerInventoryAllocation) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var item models.InventoryItem
		if err := tx.First(&item, "id = ?", row.ItemID).Error; err != nil {
			return errors.New("inventory item not found")
		}
		if row.FromLocationID == nil {
			return errors.New("from_location_id is required")
		}
		quantity := row.Quantity
		if row.SerialItemID != nil {
			quantity = 1
			row.Quantity = 1
		}
		if quantity <= 0 {
			return errors.New("quantity must be greater than zero")
		}

		var stock models.InventoryStock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("item_id = ? AND location_id = ?", row.ItemID, *row.FromLocationID).First(&stock).Error; err != nil {
			return errors.New("stock at source location not found")
		}
		available := stock.QuantityOnHand - stock.QuantityReserved
		if available < quantity {
			return fmt.Errorf("insufficient stock for %s at source location", item.Name)
		}
		stock.QuantityOnHand -= quantity
		if err := tx.Save(&stock).Error; err != nil {
			return err
		}

		if row.SerialItemID != nil {
			var serial models.InventorySerialItem
			if err := tx.First(&serial, "id = ?", *row.SerialItemID).Error; err != nil {
				return errors.New("serial item not found")
			}
			if serial.ItemID != row.ItemID {
				return errors.New("serial item does not match item")
			}
			if serial.CurrentLocationID == nil || *serial.CurrentLocationID != *row.FromLocationID {
				return errors.New("serial item is not at the source location")
			}
			serial.CurrentLocationID = nil
			if row.TransactionType == "sale" {
				serial.Status = "sold"
			} else {
				serial.Status = "installed_partner"
			}
			if err := tx.Save(&serial).Error; err != nil {
				return err
			}
		}

		movementType := "partner_loan"
		if row.TransactionType == "sale" {
			movementType = "partner_sale"
		}
		movement := models.InventoryStockMovement{
			ItemID:         row.ItemID,
			SerialItemID:   row.SerialItemID,
			FromLocationID: row.FromLocationID,
			MovementType:   movementType,
			Quantity:       quantity,
			UnitCost:       stock.AverageCost,
			TotalCost:      quantity * stock.AverageCost,
			ReferenceType:  "partner_allocation",
			ReferenceID:    &row.ID,
			Notes:          row.AllocationNo,
			CreatedBy:      row.CreatedByID,
		}
		if err := tx.Create(&movement).Error; err != nil {
			return err
		}
		return tx.Create(row).Error
	})
}

// ── Invoice audit + source references ──────────────────────────────────────

func (r *partnerRepository) CreateStatusHistory(row *models.PartnerInvoiceStatusHistory) error {
	return r.db.Create(row).Error
}

func (r *partnerRepository) ListStatusHistory(invoiceID uuid.UUID) ([]models.PartnerInvoiceStatusHistory, error) {
	var rows []models.PartnerInvoiceStatusHistory
	err := r.db.Where("invoice_id = ?", invoiceID).Order("created_at").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) ListInvoicesFiltered(partnerID *uuid.UUID, status string, overdueOnly bool) ([]models.PartnerInvoice, error) {
	var rows []models.PartnerInvoice
	query := r.db.Preload("Partner").Preload("Lines").Preload("Payments")
	if partnerID != nil {
		query = query.Where("partner_id = ?", *partnerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if overdueOnly {
		query = query.Where("balance_due > 0 AND due_date < ?", time.Now())
	}
	err := query.Order("due_date DESC").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) FindInvoiceLineBySource(sourceRefType string, sourceRefID uuid.UUID, year, month *int) (*models.PartnerInvoiceLine, error) {
	var row models.PartnerInvoiceLine
	query := r.db.Where("source_ref_type = ? AND source_ref_id = ?", sourceRefType, sourceRefID)
	if year != nil {
		query = query.Where("period_year = ?", *year)
	}
	if month != nil {
		query = query.Where("period_month = ?", *month)
	}
	err := query.First(&row).Error
	return &row, err
}

// ── Notifications ──────────────────────────────────────────────────────────

func (r *partnerRepository) CreateNotification(row *models.PartnerInvoiceNotification) error {
	return r.db.Create(row).Error
}

func (r *partnerRepository) SaveNotification(row *models.PartnerInvoiceNotification) error {
	return r.db.Save(row).Error
}

func (r *partnerRepository) FindNotificationByIdempotencyKey(key string) (*models.PartnerInvoiceNotification, error) {
	var row models.PartnerInvoiceNotification
	err := r.db.Where("idempotency_key = ?", key).First(&row).Error
	return &row, err
}

func (r *partnerRepository) FindNotificationByMessageID(messageID string) (*models.PartnerInvoiceNotification, error) {
	var row models.PartnerInvoiceNotification
	err := r.db.Where("brevo_message_id = ?", messageID).First(&row).Error
	return &row, err
}

func (r *partnerRepository) ListNotifications(invoiceID uuid.UUID) ([]models.PartnerInvoiceNotification, error) {
	var rows []models.PartnerInvoiceNotification
	err := r.db.Where("invoice_id = ?", invoiceID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// ── Legal documents ────────────────────────────────────────────────────────

func (r *partnerRepository) CreateDocument(row *models.LegalDocument) error {
	return r.db.Create(row).Error
}

func (r *partnerRepository) SaveDocument(row *models.LegalDocument) error {
	return r.db.Save(row).Error
}

func (r *partnerRepository) FindDocument(id uuid.UUID) (*models.LegalDocument, error) {
	var row models.LegalDocument
	err := r.db.First(&row, "id = ?", id).Error
	return &row, err
}

func (r *partnerRepository) ListDocuments(partnerVisible *bool, category string) ([]models.LegalDocument, error) {
	var rows []models.LegalDocument
	query := r.db.Model(&models.LegalDocument{})
	if partnerVisible != nil {
		query = query.Where("is_partner_visible = ? AND status = ?", *partnerVisible, "published")
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	err := query.Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) DeleteDocument(id uuid.UUID) error {
	return r.db.Delete(&models.LegalDocument{}, "id = ?", id).Error
}

// ── Service + invoice helpers ──────────────────────────────────────────────

func (r *partnerRepository) FindService(id uuid.UUID) (*models.PartnerBandwidthService, error) {
	var row models.PartnerBandwidthService
	err := r.db.First(&row, "id = ?", id).Error
	return &row, err
}

func (r *partnerRepository) SaveService(row *models.PartnerBandwidthService) error {
	return r.db.Omit(clause.Associations).Save(row).Error
}

func (r *partnerRepository) ListServicesDueForBilling() ([]models.PartnerBandwidthService, error) {
	var rows []models.PartnerBandwidthService
	err := r.db.Where("status = ? AND billing_mode = ?", "active", "fixed").Order("partner_id").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) ListInvoicesDueForReminder() ([]models.PartnerInvoice, error) {
	var rows []models.PartnerInvoice
	err := r.db.Preload("Partner").
		Where("status IN ? AND balance_due > 0 AND due_date <= ?", []string{"issued", "partially_paid", "overdue"}, time.Now().AddDate(0, 0, 3)).
		Order("due_date").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) ListOverdueInvoices() ([]models.PartnerInvoice, error) {
	var rows []models.PartnerInvoice
	err := r.db.Where("status IN ? AND balance_due > 0 AND due_date < ?", []string{"issued", "partially_paid"}, time.Now()).Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) FindPaymentByIdempotencyKey(key string) (*models.PartnerPayment, error) {
	var row models.PartnerPayment
	err := r.db.Where("idempotency_key = ?", key).First(&row).Error
	return &row, err
}

func (r *partnerRepository) FindAllocation(id uuid.UUID) (*models.PartnerInventoryAllocation, error) {
	var row models.PartnerInventoryAllocation
	err := r.db.Preload("Item").Preload("SerialItem").First(&row, "id = ?", id).Error
	return &row, err
}

func (r *partnerRepository) ListAllocations(partnerID *uuid.UUID, status, transactionType string) ([]models.PartnerInventoryAllocation, error) {
	var rows []models.PartnerInventoryAllocation
	query := r.db.Preload("Item").Preload("SerialItem")
	if partnerID != nil {
		query = query.Where("partner_id = ?", *partnerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if transactionType != "" {
		query = query.Where("transaction_type = ?", transactionType)
	}
	err := query.Order("allocated_at DESC, created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *partnerRepository) SaveAllocation(row *models.PartnerInventoryAllocation) error {
	return r.db.Omit(clause.Associations).Save(row).Error
}

// ReturnAllocation reverses a loan: stock is put back and the asset is marked
// returned. Sales cannot be returned through this path.
func (r *partnerRepository) ReturnAllocation(allocation *models.PartnerInventoryAllocation, toLocationID uuid.UUID, notes string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var locked models.PartnerInventoryAllocation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", allocation.ID).Error; err != nil {
			return errors.New("allocation not found")
		}
		if locked.Status != "active" {
			return errors.New("only active allocations can be returned")
		}
		if locked.TransactionType != "loan" {
			return errors.New("only loan allocations can be returned")
		}

		var stock models.InventoryStock
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("item_id = ? AND location_id = ?", locked.ItemID, toLocationID).First(&stock).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			stock = models.InventoryStock{ID: uuid.New(), ItemID: locked.ItemID, LocationID: toLocationID, QuantityOnHand: locked.Quantity}
			if err := tx.Create(&stock).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			stock.QuantityOnHand += locked.Quantity
			if err := tx.Save(&stock).Error; err != nil {
				return err
			}
		}

		if locked.SerialItemID != nil {
			var serial models.InventorySerialItem
			if err := tx.First(&serial, "id = ?", *locked.SerialItemID).Error; err != nil {
				return err
			}
			serial.CurrentLocationID = &toLocationID
			serial.Status = "available"
			if err := tx.Save(&serial).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		movement := models.InventoryStockMovement{
			ItemID:        locked.ItemID,
			SerialItemID:  locked.SerialItemID,
			ToLocationID:  &toLocationID,
			MovementType:  "partner_loan_return",
			Quantity:      locked.Quantity,
			UnitCost:      stock.AverageCost,
			TotalCost:     locked.Quantity * stock.AverageCost,
			ReferenceType: "partner_allocation",
			ReferenceID:   &locked.ID,
			Notes:         notes,
		}
		if err := tx.Create(&movement).Error; err != nil {
			return err
		}

		locked.Status = "returned"
		locked.ReturnedAt = &now
		if notes != "" {
			locked.Notes = notes
		}
		if err := tx.Omit(clause.Associations).Save(&locked).Error; err != nil {
			return err
		}
		*allocation = locked
		return nil
	})
}

func (r *partnerRepository) LinkAllocationToInvoiceLine(allocationID, invoiceLineID uuid.UUID) error {
	return r.db.Model(&models.PartnerInventoryAllocation{}).
		Where("id = ?", allocationID).
		Update("invoice_line_id", invoiceLineID).Error
}
