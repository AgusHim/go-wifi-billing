package repositories

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryRepository interface {
	CreateItem(item *models.InventoryItem) error
	FindItems(search string, active *bool) ([]models.InventoryItem, error)
	FindItemByID(id uuid.UUID) (*models.InventoryItem, error)
	FindItemBySKU(sku string) (*models.InventoryItem, error)
	UpdateItem(item *models.InventoryItem) error
	DeleteItem(id uuid.UUID) error

	CreateLocation(location *models.InventoryLocation) error
	FindLocations(locationType string, active *bool) ([]models.InventoryLocation, error)
	FindLocationByID(id uuid.UUID) (*models.InventoryLocation, error)
	UpdateLocation(location *models.InventoryLocation) error
	DeleteLocation(id uuid.UUID) error

	FindStocks(itemID *uuid.UUID, locationID *uuid.UUID) ([]models.InventoryStock, error)
	FindMovements(itemID *uuid.UUID, locationID *uuid.UUID, movementType string, limit int) ([]models.InventoryStockMovement, error)
	FindSerialItems(itemID *uuid.UUID, status string) ([]models.InventorySerialItem, error)
	FindSerialItemByID(id uuid.UUID) (*models.InventorySerialItem, error)
	FindSerialItemByItemAndSerial(itemID uuid.UUID, serial string) (*models.InventorySerialItem, error)

	CreateSupplier(supplier *models.Supplier) error
	FindSuppliers(search string, active *bool) ([]models.Supplier, error)
	FindSupplierByID(id uuid.UUID) (*models.Supplier, error)
	UpdateSupplier(supplier *models.Supplier) error
	DeleteSupplier(id uuid.UUID) error

	CreatePurchaseOrder(po *models.PurchaseOrder) error
	FindPurchaseOrders(status string) ([]models.PurchaseOrder, error)
	FindPurchaseOrderByID(id uuid.UUID) (*models.PurchaseOrder, error)
	UpdatePurchaseOrder(po *models.PurchaseOrder) error

	CreateGoodsReceiptWithStock(receipt *models.GoodsReceipt, po *models.PurchaseOrder) error
	FindGoodsReceipts(status string) ([]models.GoodsReceipt, error)
	FindGoodsReceiptByID(id uuid.UUID) (*models.GoodsReceipt, error)

	CreateStockTransferWithReservation(transfer *models.StockTransfer) error
	FindStockTransfers(status string) ([]models.StockTransfer, error)
	FindStockTransferByID(id uuid.UUID) (*models.StockTransfer, error)
	ReceiveStockTransfer(transfer *models.StockTransfer, receivedBy *uuid.UUID) error

	CreateMaterialUsageWithStock(usage *models.MaterialUsage) error
	FindMaterialUsages(referenceType string, referenceID *uuid.UUID) ([]models.MaterialUsage, error)
	FindMaterialUsageByID(id uuid.UUID) (*models.MaterialUsage, error)

	FindCustomerAssets(customerID *uuid.UUID, status string) ([]models.CustomerAsset, error)
	FindCustomerAssetByID(id uuid.UUID) (*models.CustomerAsset, error)
	ReturnCustomerAsset(asset *models.CustomerAsset, returnLocationID uuid.UUID, notes string) error
	ReplaceCustomerAsset(asset *models.CustomerAsset, newSerialItem *models.InventorySerialItem, locationID uuid.UUID, notes string) error
	UpdateCustomerAssetStatus(asset *models.CustomerAsset, status string, notes string) error
	FindNetworkAssets(referenceType string, referenceID *uuid.UUID, status string) ([]models.NetworkAsset, error)
	FindNetworkAssetByID(id uuid.UUID) (*models.NetworkAsset, error)
	UpdateNetworkAssetStatus(asset *models.NetworkAsset, status string, notes string) error

	CreateStockOpname(opname *models.StockOpname) error
	FindStockOpnames(locationID *uuid.UUID, status string) ([]models.StockOpname, error)
	FindStockOpnameByID(id uuid.UUID) (*models.StockOpname, error)
	SubmitStockOpname(opname *models.StockOpname, submittedBy *uuid.UUID) error
	ApproveStockOpname(opname *models.StockOpname, approvedBy *uuid.UUID) error

	FindChartOfAccounts() ([]models.ChartOfAccount, error)
	FindAccountingJournals(sourceType string, sourceID *uuid.UUID) ([]models.AccountingJournal, error)
	FindInventoryValuation() ([]models.InventoryStock, error)
	UpsertPeriodLock(lock *models.AccountingPeriodLock) error
	FindPeriodLocks() ([]models.AccountingPeriodLock, error)
	CreateSupplierInvoice(invoice *models.SupplierInvoice) error
	FindSupplierInvoices(status string, supplierID *uuid.UUID) ([]models.SupplierInvoice, error)
	FindSupplierInvoiceByID(id uuid.UUID) (*models.SupplierInvoice, error)
	CreateSupplierPayment(payment *models.SupplierPayment) error
	FindSupplierPayments(invoiceID *uuid.UUID, supplierID *uuid.UUID) ([]models.SupplierPayment, error)
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) CreateItem(item *models.InventoryItem) error {
	return r.db.Create(item).Error
}

func (r *inventoryRepository) FindItems(search string, active *bool) ([]models.InventoryItem, error) {
	var items []models.InventoryItem
	query := r.db.Order("created_at DESC")
	if strings.TrimSpace(search) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		query = query.Where("LOWER(sku) LIKE ? OR LOWER(name) LIKE ? OR LOWER(category) LIKE ?", keyword, keyword, keyword)
	}
	if active != nil {
		query = query.Where("is_active = ?", *active)
	}
	err := query.Find(&items).Error
	return items, err
}

func (r *inventoryRepository) FindItemByID(id uuid.UUID) (*models.InventoryItem, error) {
	var item models.InventoryItem
	err := r.db.First(&item, "id = ?", id).Error
	return &item, err
}

func (r *inventoryRepository) FindItemBySKU(sku string) (*models.InventoryItem, error) {
	var item models.InventoryItem
	err := r.db.First(&item, "LOWER(sku) = LOWER(?)", strings.TrimSpace(sku)).Error
	return &item, err
}

func (r *inventoryRepository) UpdateItem(item *models.InventoryItem) error {
	return r.db.Omit(clause.Associations).Save(item).Error
}

func (r *inventoryRepository) DeleteItem(id uuid.UUID) error {
	return r.db.Delete(&models.InventoryItem{}, "id = ?", id).Error
}

func (r *inventoryRepository) CreateLocation(location *models.InventoryLocation) error {
	return r.db.Create(location).Error
}

func (r *inventoryRepository) FindLocations(locationType string, active *bool) ([]models.InventoryLocation, error) {
	var locations []models.InventoryLocation
	query := r.db.
		Preload("Coverage").
		Preload("TechnicianUser").
		Order("created_at DESC")
	if strings.TrimSpace(locationType) != "" {
		query = query.Where("type = ?", strings.TrimSpace(locationType))
	}
	if active != nil {
		query = query.Where("is_active = ?", *active)
	}
	err := query.Find(&locations).Error
	return locations, err
}

func (r *inventoryRepository) FindLocationByID(id uuid.UUID) (*models.InventoryLocation, error) {
	var location models.InventoryLocation
	err := r.db.
		Preload("Coverage").
		Preload("TechnicianUser").
		First(&location, "id = ?", id).Error
	return &location, err
}

func (r *inventoryRepository) UpdateLocation(location *models.InventoryLocation) error {
	return r.db.Omit(clause.Associations).Save(location).Error
}

func (r *inventoryRepository) DeleteLocation(id uuid.UUID) error {
	return r.db.Delete(&models.InventoryLocation{}, "id = ?", id).Error
}

func (r *inventoryRepository) FindStocks(itemID *uuid.UUID, locationID *uuid.UUID) ([]models.InventoryStock, error) {
	var stocks []models.InventoryStock
	query := r.db.
		Preload("Item").
		Preload("Location").
		Order("updated_at DESC")
	if itemID != nil {
		query = query.Where("item_id = ?", *itemID)
	}
	if locationID != nil {
		query = query.Where("location_id = ?", *locationID)
	}
	err := query.Find(&stocks).Error
	return stocks, err
}

func (r *inventoryRepository) FindMovements(itemID *uuid.UUID, locationID *uuid.UUID, movementType string, limit int) ([]models.InventoryStockMovement, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var movements []models.InventoryStockMovement
	query := r.db.
		Preload("Item").
		Preload("SerialItem").
		Preload("FromLocation").
		Preload("ToLocation").
		Preload("Creator").
		Order("created_at DESC").
		Limit(limit)
	if itemID != nil {
		query = query.Where("item_id = ?", *itemID)
	}
	if locationID != nil {
		query = query.Where("from_location_id = ? OR to_location_id = ?", *locationID, *locationID)
	}
	if strings.TrimSpace(movementType) != "" {
		query = query.Where("movement_type = ?", strings.TrimSpace(movementType))
	}
	err := query.Find(&movements).Error
	return movements, err
}

func (r *inventoryRepository) FindSerialItems(itemID *uuid.UUID, status string) ([]models.InventorySerialItem, error) {
	var serials []models.InventorySerialItem
	query := r.db.
		Preload("Item").
		Preload("CurrentLocation").
		Preload("Customer").
		Preload("Customer.User").
		Preload("ServiceAccount").
		Order("created_at DESC")
	if itemID != nil {
		query = query.Where("item_id = ?", *itemID)
	}
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&serials).Error
	return serials, err
}

func (r *inventoryRepository) FindSerialItemByID(id uuid.UUID) (*models.InventorySerialItem, error) {
	var item models.InventorySerialItem
	err := r.db.
		Preload("Item").
		Preload("CurrentLocation").
		First(&item, "id = ?", id).Error
	return &item, err
}

func (r *inventoryRepository) FindSerialItemByItemAndSerial(itemID uuid.UUID, serial string) (*models.InventorySerialItem, error) {
	var item models.InventorySerialItem
	err := r.db.First(&item, "item_id = ? AND LOWER(serial_number) = LOWER(?)", itemID, strings.TrimSpace(serial)).Error
	return &item, err
}

func (r *inventoryRepository) CreateSupplier(supplier *models.Supplier) error {
	return r.db.Create(supplier).Error
}

func (r *inventoryRepository) FindSuppliers(search string, active *bool) ([]models.Supplier, error) {
	var suppliers []models.Supplier
	query := r.db.Order("created_at DESC")
	if strings.TrimSpace(search) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(phone) LIKE ? OR LOWER(email) LIKE ?", keyword, keyword, keyword)
	}
	if active != nil {
		query = query.Where("is_active = ?", *active)
	}
	err := query.Find(&suppliers).Error
	return suppliers, err
}

func (r *inventoryRepository) FindSupplierByID(id uuid.UUID) (*models.Supplier, error) {
	var supplier models.Supplier
	err := r.db.First(&supplier, "id = ?", id).Error
	return &supplier, err
}

func (r *inventoryRepository) UpdateSupplier(supplier *models.Supplier) error {
	return r.db.Omit(clause.Associations).Save(supplier).Error
}

func (r *inventoryRepository) DeleteSupplier(id uuid.UUID) error {
	return r.db.Delete(&models.Supplier{}, "id = ?", id).Error
}

func (r *inventoryRepository) CreatePurchaseOrder(po *models.PurchaseOrder) error {
	return r.db.Create(po).Error
}

func (r *inventoryRepository) FindPurchaseOrders(status string) ([]models.PurchaseOrder, error) {
	var purchaseOrders []models.PurchaseOrder
	query := r.db.
		Preload("Supplier").
		Preload("Items").
		Preload("Items.Item").
		Order("created_at DESC")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&purchaseOrders).Error
	return purchaseOrders, err
}

func (r *inventoryRepository) FindPurchaseOrderByID(id uuid.UUID) (*models.PurchaseOrder, error) {
	var po models.PurchaseOrder
	err := r.db.
		Preload("Supplier").
		Preload("Items").
		Preload("Items.Item").
		Preload("Receipts").
		Preload("Receipts.Items").
		First(&po, "id = ?", id).Error
	return &po, err
}

func (r *inventoryRepository) UpdatePurchaseOrder(po *models.PurchaseOrder) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(po).Error
}

func (r *inventoryRepository) FindGoodsReceipts(status string) ([]models.GoodsReceipt, error) {
	var receipts []models.GoodsReceipt
	query := r.db.
		Preload("Supplier").
		Preload("PurchaseOrder").
		Preload("WarehouseLocation").
		Preload("Items").
		Preload("Items.Item").
		Order("created_at DESC")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&receipts).Error
	return receipts, err
}

func (r *inventoryRepository) FindGoodsReceiptByID(id uuid.UUID) (*models.GoodsReceipt, error) {
	var receipt models.GoodsReceipt
	err := r.db.
		Preload("Supplier").
		Preload("PurchaseOrder").
		Preload("WarehouseLocation").
		Preload("Items").
		Preload("Items.Item").
		First(&receipt, "id = ?", id).Error
	return &receipt, err
}

func (r *inventoryRepository) CreateGoodsReceiptWithStock(receipt *models.GoodsReceipt, po *models.PurchaseOrder) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(receipt).Error; err != nil {
			return err
		}

		receivedByItem := map[uuid.UUID]float64{}
		for _, receiptItem := range receipt.Items {
			receivedByItem[receiptItem.ItemID] += receiptItem.Quantity

			var stock models.InventoryStock
			err := tx.Where("item_id = ? AND location_id = ?", receiptItem.ItemID, receipt.WarehouseLocationID).First(&stock).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			oldQuantity := stock.QuantityOnHand
			oldAverageCost := stock.AverageCost
			newQuantity := oldQuantity + receiptItem.Quantity
			newAverageCost := receiptItem.UnitCost
			if newQuantity > 0 {
				newAverageCost = ((oldQuantity * oldAverageCost) + (receiptItem.Quantity * receiptItem.UnitCost)) / newQuantity
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				stock = models.InventoryStock{
					ItemID:           receiptItem.ItemID,
					LocationID:       receipt.WarehouseLocationID,
					QuantityOnHand:   receiptItem.Quantity,
					QuantityReserved: 0,
					AverageCost:      newAverageCost,
				}
				if err := tx.Create(&stock).Error; err != nil {
					return err
				}
			} else {
				stock.QuantityOnHand = newQuantity
				stock.AverageCost = newAverageCost
				if err := tx.Save(&stock).Error; err != nil {
					return err
				}
			}

			movement := models.InventoryStockMovement{
				ItemID:        receiptItem.ItemID,
				ToLocationID:  &receipt.WarehouseLocationID,
				MovementType:  "purchase_receipt",
				Quantity:      receiptItem.Quantity,
				UnitCost:      receiptItem.UnitCost,
				TotalCost:     receiptItem.Total,
				ReferenceType: "goods_receipt",
				ReferenceID:   &receipt.ID,
				Notes:         receipt.ReceiptNumber,
				CreatedBy:     receipt.ReceivedBy,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}

			serialNumbers := splitInventoryCSV(receiptItem.SerialNumbers)
			macAddresses := splitInventoryCSV(receiptItem.MACAddresses)
			for index, serialNumber := range serialNumbers {
				macAddress := ""
				if len(macAddresses) > index {
					macAddress = macAddresses[index]
				}
				serialItem := models.InventorySerialItem{
					ItemID:            receiptItem.ItemID,
					SerialNumber:      serialNumber,
					MACAddress:        macAddress,
					CurrentLocationID: &receipt.WarehouseLocationID,
					Status:            "in_stock",
					PurchaseReceiptID: &receipt.ID,
				}
				if err := tx.Create(&serialItem).Error; err != nil {
					return err
				}
			}
		}

		fullyReceived := true
		for i := range po.Items {
			po.Items[i].ReceivedQuantity += receivedByItem[po.Items[i].ItemID]
			if po.Items[i].ReceivedQuantity < po.Items[i].Quantity {
				fullyReceived = false
			}
			if err := tx.Save(&po.Items[i]).Error; err != nil {
				return err
			}
		}
		if fullyReceived {
			po.Status = "received"
		} else {
			po.Status = "partially_received"
		}
		if err := tx.Omit(clause.Associations).Save(po).Error; err != nil {
			return err
		}
		if err := postGoodsReceiptJournal(tx, receipt); err != nil {
			return err
		}
		return createSupplierInvoiceFromGoodsReceipt(tx, receipt)
	})
}

func splitInventoryCSV(value string) []string {
	normalized := strings.NewReplacer("\n", ",", "\r", ",", ";", ",").Replace(value)
	parts := strings.Split(normalized, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (r *inventoryRepository) CreateStockTransferWithReservation(transfer *models.StockTransfer) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range transfer.Items {
			item := &transfer.Items[i]
			var stock models.InventoryStock
			if err := tx.Where("item_id = ? AND location_id = ?", item.ItemID, transfer.FromLocationID).First(&stock).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("stok lokasi asal tidak ditemukan")
				}
				return err
			}
			available := stock.QuantityOnHand - stock.QuantityReserved
			if available < item.Quantity {
				return errors.New("stok lokasi asal tidak cukup")
			}
			item.UnitCost = stock.AverageCost
			item.TotalCost = item.Quantity * stock.AverageCost
			stock.QuantityReserved += item.Quantity
			if err := tx.Save(&stock).Error; err != nil {
				return err
			}
		}
		return tx.Create(transfer).Error
	})
}

func (r *inventoryRepository) FindStockTransfers(status string) ([]models.StockTransfer, error) {
	var transfers []models.StockTransfer
	query := r.db.
		Preload("FromLocation").
		Preload("ToLocation").
		Preload("ToLocation.TechnicianUser").
		Preload("Items").
		Preload("Items.Item").
		Preload("Items.SerialItem").
		Order("created_at DESC")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&transfers).Error
	return transfers, err
}

func (r *inventoryRepository) FindStockTransferByID(id uuid.UUID) (*models.StockTransfer, error) {
	var transfer models.StockTransfer
	err := r.db.
		Preload("FromLocation").
		Preload("ToLocation").
		Preload("ToLocation.TechnicianUser").
		Preload("Items").
		Preload("Items.Item").
		Preload("Items.SerialItem").
		First(&transfer, "id = ?", id).Error
	return &transfer, err
}

func (r *inventoryRepository) ReceiveStockTransfer(transfer *models.StockTransfer, receivedBy *uuid.UUID) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range transfer.Items {
			item := &transfer.Items[i]

			var fromStock models.InventoryStock
			if err := tx.Where("item_id = ? AND location_id = ?", item.ItemID, transfer.FromLocationID).First(&fromStock).Error; err != nil {
				return err
			}
			if fromStock.QuantityOnHand < item.Quantity || fromStock.QuantityReserved < item.Quantity {
				return errors.New("stok lokasi asal tidak cukup untuk receive transfer")
			}
			fromStock.QuantityOnHand -= item.Quantity
			fromStock.QuantityReserved -= item.Quantity
			if err := tx.Save(&fromStock).Error; err != nil {
				return err
			}

			var toStock models.InventoryStock
			err := tx.Where("item_id = ? AND location_id = ?", item.ItemID, transfer.ToLocationID).First(&toStock).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				toStock = models.InventoryStock{
					ItemID:           item.ItemID,
					LocationID:       transfer.ToLocationID,
					QuantityOnHand:   item.Quantity,
					QuantityReserved: 0,
					AverageCost:      item.UnitCost,
				}
				if err := tx.Create(&toStock).Error; err != nil {
					return err
				}
			} else {
				oldQuantity := toStock.QuantityOnHand
				newQuantity := oldQuantity + item.Quantity
				newAverageCost := item.UnitCost
				if newQuantity > 0 {
					newAverageCost = ((oldQuantity * toStock.AverageCost) + (item.Quantity * item.UnitCost)) / newQuantity
				}
				toStock.QuantityOnHand = newQuantity
				toStock.AverageCost = newAverageCost
				if err := tx.Save(&toStock).Error; err != nil {
					return err
				}
			}

			movement := models.InventoryStockMovement{
				ItemID:         item.ItemID,
				SerialItemID:   item.SerialItemID,
				FromLocationID: &transfer.FromLocationID,
				ToLocationID:   &transfer.ToLocationID,
				MovementType:   "transfer",
				Quantity:       item.Quantity,
				UnitCost:       item.UnitCost,
				TotalCost:      item.TotalCost,
				ReferenceType:  "stock_transfer",
				ReferenceID:    &transfer.ID,
				Notes:          transfer.TransferNumber,
				CreatedBy:      receivedBy,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}

			if item.SerialItemID != nil {
				var serial models.InventorySerialItem
				if err := tx.First(&serial, "id = ?", *item.SerialItemID).Error; err != nil {
					return err
				}
				serial.CurrentLocationID = &transfer.ToLocationID
				serial.Status = "in_stock"
				var toLocation models.InventoryLocation
				if err := tx.First(&toLocation, "id = ?", transfer.ToLocationID).Error; err == nil && toLocation.Type == "technician" {
					serial.Status = "assigned_to_technician"
				}
				if err := tx.Save(&serial).Error; err != nil {
					return err
				}
			}

			item.ReceivedQuantity = item.Quantity
			if err := tx.Save(item).Error; err != nil {
				return err
			}
		}

		transfer.Status = "received"
		transfer.ReceivedBy = receivedBy
		transfer.ReceivedAt = &now
		return tx.Omit(clause.Associations).Save(transfer).Error
	})
}

func (r *inventoryRepository) CreateMaterialUsageWithStock(usage *models.MaterialUsage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(usage).Error; err != nil {
			return err
		}

		for index := range usage.Items {
			item := &usage.Items[index]
			var stock models.InventoryStock
			if err := tx.Where("item_id = ? AND location_id = ?", item.ItemID, usage.LocationID).First(&stock).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("stok lokasi pemakaian tidak ditemukan")
				}
				return err
			}
			available := stock.QuantityOnHand - stock.QuantityReserved
			if available < item.Quantity {
				return errors.New("stok lokasi pemakaian tidak cukup")
			}
			stock.QuantityOnHand -= item.Quantity
			item.UnitCost = stock.AverageCost
			item.TotalCost = item.Quantity * item.UnitCost
			if err := tx.Save(&stock).Error; err != nil {
				return err
			}

			var inventoryItem models.InventoryItem
			if err := tx.First(&inventoryItem, "id = ?", item.ItemID).Error; err != nil {
				return err
			}

			movementType := "material_issue"
			if inventoryItem.AccountingType == "customer_asset" {
				movementType = "customer_install"
			}
			if inventoryItem.AccountingType == "network_asset" {
				movementType = "network_install"
			}
			movement := models.InventoryStockMovement{
				ItemID:         item.ItemID,
				SerialItemID:   item.SerialItemID,
				FromLocationID: &usage.LocationID,
				MovementType:   movementType,
				Quantity:       item.Quantity,
				UnitCost:       item.UnitCost,
				TotalCost:      item.TotalCost,
				ReferenceType:  "material_usage",
				ReferenceID:    &usage.ID,
				Notes:          usage.UsageNumber,
				CreatedBy:      usage.TechnicianUserID,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}

			if item.SerialItemID != nil {
				var serial models.InventorySerialItem
				if err := tx.First(&serial, "id = ?", *item.SerialItemID).Error; err != nil {
					return err
				}
				serial.CurrentLocationID = nil
				if inventoryItem.AccountingType == "customer_asset" {
					serial.Status = "installed_customer"
					serial.CustomerID = usage.CustomerID
					serial.ServiceAccountID = usage.ServiceAccountID
				} else if inventoryItem.AccountingType == "network_asset" {
					serial.Status = "installed_network"
				} else {
					serial.Status = "used"
				}
				if err := tx.Save(&serial).Error; err != nil {
					return err
				}
			}

			if inventoryItem.AccountingType == "customer_asset" && usage.CustomerID != nil {
				asset := models.CustomerAsset{
					CustomerID:       *usage.CustomerID,
					SubscriptionID:   usage.SubscriptionID,
					ServiceAccountID: usage.ServiceAccountID,
					SerialItemID:     item.SerialItemID,
					ItemID:           item.ItemID,
					InstalledAt:      usage.UsedAt,
					Status:           "installed",
					OwnershipType:    "isp_owned",
					Notes:            usage.UsageNumber,
				}
				if err := tx.Create(&asset).Error; err != nil {
					return err
				}
			}

			if inventoryItem.AccountingType == "network_asset" {
				assetCode := fmt.Sprintf("%s-%02d", usage.UsageNumber, index+1)
				asset := models.NetworkAsset{
					ItemID:          item.ItemID,
					SerialItemID:    item.SerialItemID,
					AssetCode:       assetCode,
					AssetType:       inventoryItem.Category,
					CoverageID:      usage.CoverageID,
					OdcID:           usage.OdcID,
					OdpID:           usage.OdpID,
					RouterID:        usage.RouterID,
					InstalledAt:     usage.UsedAt,
					Status:          "installed",
					AcquisitionCost: item.TotalCost,
					Notes:           usage.UsageNumber,
				}
				if asset.AssetType == "" {
					asset.AssetType = inventoryItem.AccountingType
				}
				if err := tx.Create(&asset).Error; err != nil {
					return err
				}
			}
		}

		return postMaterialUsageJournal(tx, usage)
	})
}

func (r *inventoryRepository) FindMaterialUsages(referenceType string, referenceID *uuid.UUID) ([]models.MaterialUsage, error) {
	var usages []models.MaterialUsage
	query := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("ServiceAccount").
		Preload("Complain").
		Preload("Coverage").
		Preload("Odp").
		Preload("Location").
		Preload("Items").
		Preload("Items.Item").
		Preload("Items.SerialItem").
		Order("created_at DESC")
	if referenceID != nil {
		switch strings.TrimSpace(referenceType) {
		case "customer":
			query = query.Where("customer_id = ?", *referenceID)
		case "subscription":
			query = query.Where("subscription_id = ?", *referenceID)
		case "service_account":
			query = query.Where("service_account_id = ?", *referenceID)
		case "complain":
			query = query.Where("complain_id = ?", *referenceID)
		case "coverage":
			query = query.Where("coverage_id = ?", *referenceID)
		case "odp":
			query = query.Where("odp_id = ?", *referenceID)
		}
	}
	err := query.Find(&usages).Error
	return usages, err
}

func (r *inventoryRepository) FindMaterialUsageByID(id uuid.UUID) (*models.MaterialUsage, error) {
	var usage models.MaterialUsage
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("ServiceAccount").
		Preload("Complain").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		Preload("Router").
		Preload("Technician").
		Preload("Location").
		Preload("Items").
		Preload("Items.Item").
		Preload("Items.SerialItem").
		First(&usage, "id = ?", id).Error
	return &usage, err
}

func (r *inventoryRepository) FindCustomerAssets(customerID *uuid.UUID, status string) ([]models.CustomerAsset, error) {
	var assets []models.CustomerAsset
	query := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("ServiceAccount").
		Preload("SerialItem").
		Preload("Item").
		Order("created_at DESC")
	if customerID != nil {
		query = query.Where("customer_id = ?", *customerID)
	}
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&assets).Error
	return assets, err
}

func (r *inventoryRepository) FindCustomerAssetByID(id uuid.UUID) (*models.CustomerAsset, error) {
	var asset models.CustomerAsset
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("ServiceAccount").
		Preload("SerialItem").
		Preload("Item").
		First(&asset, "id = ?", id).Error
	return &asset, err
}

func (r *inventoryRepository) ReturnCustomerAsset(asset *models.CustomerAsset, returnLocationID uuid.UUID, notes string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if asset.SerialItemID != nil {
			var serial models.InventorySerialItem
			if err := tx.First(&serial, "id = ?", *asset.SerialItemID).Error; err != nil {
				return err
			}
			serial.CurrentLocationID = &returnLocationID
			serial.Status = "returned"
			serial.CustomerID = nil
			serial.ServiceAccountID = nil
			if err := tx.Save(&serial).Error; err != nil {
				return err
			}
		}

		if err := incrementInventoryStock(tx, asset.ItemID, returnLocationID, 1, 0); err != nil {
			return err
		}

		asset.Status = "returned"
		asset.ReturnedAt = &now
		asset.Notes = mergeInventoryNotes(asset.Notes, notes)
		return tx.Omit(clause.Associations).Save(asset).Error
	})
}

func (r *inventoryRepository) ReplaceCustomerAsset(asset *models.CustomerAsset, newSerialItem *models.InventorySerialItem, locationID uuid.UUID, notes string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		var newItem models.InventoryItem
		if err := tx.First(&newItem, "id = ?", newSerialItem.ItemID).Error; err != nil {
			return err
		}
		if err := decrementInventoryStock(tx, newSerialItem.ItemID, locationID, 1); err != nil {
			return err
		}
		newSerialItem.CurrentLocationID = nil
		newSerialItem.Status = "installed_customer"
		newSerialItem.CustomerID = &asset.CustomerID
		newSerialItem.ServiceAccountID = asset.ServiceAccountID
		if err := tx.Save(newSerialItem).Error; err != nil {
			return err
		}

		if asset.SerialItemID != nil {
			var oldSerial models.InventorySerialItem
			if err := tx.First(&oldSerial, "id = ?", *asset.SerialItemID).Error; err == nil {
				oldSerial.Status = "returned"
				oldSerial.CustomerID = nil
				oldSerial.ServiceAccountID = nil
				if err := tx.Save(&oldSerial).Error; err != nil {
					return err
				}
			}
		}

		asset.Status = "replaced"
		asset.ReturnedAt = &now
		asset.Notes = mergeInventoryNotes(asset.Notes, notes)
		if err := tx.Omit(clause.Associations).Save(asset).Error; err != nil {
			return err
		}

		replacement := models.CustomerAsset{
			CustomerID:       asset.CustomerID,
			SubscriptionID:   asset.SubscriptionID,
			ServiceAccountID: asset.ServiceAccountID,
			SerialItemID:     &newSerialItem.ID,
			ItemID:           newSerialItem.ItemID,
			InstalledAt:      now,
			Status:           "installed",
			OwnershipType:    asset.OwnershipType,
			DepositAmount:    asset.DepositAmount,
			Notes:            mergeInventoryNotes("replacement", notes),
		}
		if replacement.OwnershipType == "" {
			replacement.OwnershipType = "isp_owned"
		}
		_ = newItem
		return tx.Create(&replacement).Error
	})
}

func (r *inventoryRepository) UpdateCustomerAssetStatus(asset *models.CustomerAsset, status string, notes string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		asset.Status = status
		asset.Notes = mergeInventoryNotes(asset.Notes, notes)
		if asset.SerialItemID != nil {
			var serial models.InventorySerialItem
			if err := tx.First(&serial, "id = ?", *asset.SerialItemID).Error; err != nil {
				return err
			}
			serial.Status = serialStatusFromAssetStatus(status)
			serial.CurrentLocationID = nil
			if status == "lost" || status == "scrap" {
				serial.CustomerID = nil
				serial.ServiceAccountID = nil
			}
			if err := tx.Save(&serial).Error; err != nil {
				return err
			}
		}
		if err := tx.Omit(clause.Associations).Save(asset).Error; err != nil {
			return err
		}
		return postAssetStatusJournal(tx, "customer_asset_"+status, asset.ID, asset.ItemID, status, notes)
	})
}

func (r *inventoryRepository) FindNetworkAssets(referenceType string, referenceID *uuid.UUID, status string) ([]models.NetworkAsset, error) {
	var assets []models.NetworkAsset
	query := r.db.
		Preload("Item").
		Preload("SerialItem").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		Preload("Router").
		Order("created_at DESC")
	if referenceID != nil {
		switch strings.TrimSpace(referenceType) {
		case "coverage":
			query = query.Where("coverage_id = ?", *referenceID)
		case "odc":
			query = query.Where("odc_id = ?", *referenceID)
		case "odp":
			query = query.Where("odp_id = ?", *referenceID)
		case "router":
			query = query.Where("router_id = ?", *referenceID)
		}
	}
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&assets).Error
	return assets, err
}

func (r *inventoryRepository) FindNetworkAssetByID(id uuid.UUID) (*models.NetworkAsset, error) {
	var asset models.NetworkAsset
	err := r.db.
		Preload("Item").
		Preload("SerialItem").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		Preload("Router").
		First(&asset, "id = ?", id).Error
	return &asset, err
}

func (r *inventoryRepository) UpdateNetworkAssetStatus(asset *models.NetworkAsset, status string, notes string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		asset.Status = status
		asset.Notes = mergeInventoryNotes(asset.Notes, notes)
		if asset.SerialItemID != nil {
			var serial models.InventorySerialItem
			if err := tx.First(&serial, "id = ?", *asset.SerialItemID).Error; err != nil {
				return err
			}
			serial.Status = serialStatusFromAssetStatus(status)
			serial.CurrentLocationID = nil
			if err := tx.Save(&serial).Error; err != nil {
				return err
			}
		}
		if err := tx.Omit(clause.Associations).Save(asset).Error; err != nil {
			return err
		}
		return postAssetStatusJournal(tx, "network_asset_"+status, asset.ID, asset.ItemID, status, notes)
	})
}

func incrementInventoryStock(tx *gorm.DB, itemID uuid.UUID, locationID uuid.UUID, quantity float64, unitCost float64) error {
	var stock models.InventoryStock
	err := tx.Where("item_id = ? AND location_id = ?", itemID, locationID).First(&stock).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		stock = models.InventoryStock{ItemID: itemID, LocationID: locationID, QuantityOnHand: quantity, AverageCost: unitCost}
		return tx.Create(&stock).Error
	}
	stock.QuantityOnHand += quantity
	return tx.Save(&stock).Error
}

func decrementInventoryStock(tx *gorm.DB, itemID uuid.UUID, locationID uuid.UUID, quantity float64) error {
	var stock models.InventoryStock
	if err := tx.Where("item_id = ? AND location_id = ?", itemID, locationID).First(&stock).Error; err != nil {
		return err
	}
	if stock.QuantityOnHand-stock.QuantityReserved < quantity {
		return errors.New("stok lokasi tidak cukup")
	}
	stock.QuantityOnHand -= quantity
	return tx.Save(&stock).Error
}

func mergeInventoryNotes(existing string, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return next
	}
	return existing + "\n" + next
}

func serialStatusFromAssetStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "damaged":
		return "repair"
	case "lost":
		return "lost"
	case "scrap":
		return "scrap"
	case "returned":
		return "returned"
	default:
		return status
	}
}

func (r *inventoryRepository) CreateStockOpname(opname *models.StockOpname) error {
	return r.db.Create(opname).Error
}

func (r *inventoryRepository) FindStockOpnames(locationID *uuid.UUID, status string) ([]models.StockOpname, error) {
	var opnames []models.StockOpname
	query := r.db.
		Preload("Location").
		Preload("Items").
		Preload("Items.Item").
		Preload("Items.SerialItem").
		Order("created_at DESC")
	if locationID != nil {
		query = query.Where("location_id = ?", *locationID)
	}
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	err := query.Find(&opnames).Error
	return opnames, err
}

func (r *inventoryRepository) FindStockOpnameByID(id uuid.UUID) (*models.StockOpname, error) {
	var opname models.StockOpname
	err := r.db.
		Preload("Location").
		Preload("Items").
		Preload("Items.Item").
		Preload("Items.SerialItem").
		First(&opname, "id = ?", id).Error
	return &opname, err
}

func (r *inventoryRepository) SubmitStockOpname(opname *models.StockOpname, submittedBy *uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		opname.Status = "submitted"
		opname.SubmittedBy = submittedBy
		return tx.Omit(clause.Associations).Save(opname).Error
	})
}

func (r *inventoryRepository) ApproveStockOpname(opname *models.StockOpname, approvedBy *uuid.UUID) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range opname.Items {
			item := &opname.Items[i]
			var stock models.InventoryStock
			err := tx.Where("item_id = ? AND location_id = ?", item.ItemID, opname.LocationID).First(&stock).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			averageCost := 0.0
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				averageCost = stock.AverageCost
			}

			variance := item.CountedQuantity - item.SystemQuantity
			item.VarianceQuantity = variance
			item.VarianceCost = variance * averageCost
			if err := tx.Save(item).Error; err != nil {
				return err
			}
			if variance == 0 {
				continue
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				stock = models.InventoryStock{
					ItemID:         item.ItemID,
					LocationID:     opname.LocationID,
					QuantityOnHand: item.CountedQuantity,
					AverageCost:    averageCost,
				}
				if err := tx.Create(&stock).Error; err != nil {
					return err
				}
			} else {
				stock.QuantityOnHand = item.CountedQuantity
				if err := tx.Save(&stock).Error; err != nil {
					return err
				}
			}

			movementType := "adjustment_plus"
			if variance < 0 {
				movementType = "adjustment_minus"
			}
			quantity := mathAbsFloat(variance)
			movement := models.InventoryStockMovement{
				ItemID:        item.ItemID,
				SerialItemID:  item.SerialItemID,
				MovementType:  movementType,
				Quantity:      quantity,
				UnitCost:      averageCost,
				TotalCost:     quantity * averageCost,
				ReferenceType: "stock_opname",
				ReferenceID:   &opname.ID,
				Notes:         item.Notes,
				CreatedBy:     approvedBy,
			}
			if movementType == "adjustment_plus" {
				movement.ToLocationID = &opname.LocationID
			} else {
				movement.FromLocationID = &opname.LocationID
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}
		}

		if err := postStockOpnameJournal(tx, opname); err != nil {
			return err
		}
		opname.Status = "approved"
		opname.ApprovedBy = approvedBy
		opname.ApprovedAt = &now
		opname.FinishedAt = &now
		return tx.Omit(clause.Associations).Save(opname).Error
	})
}

func mathAbsFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func (r *inventoryRepository) FindChartOfAccounts() ([]models.ChartOfAccount, error) {
	if err := ensureDefaultChartOfAccounts(r.db); err != nil {
		return nil, err
	}
	var accounts []models.ChartOfAccount
	err := r.db.Order("code ASC").Find(&accounts).Error
	return accounts, err
}

func (r *inventoryRepository) FindAccountingJournals(sourceType string, sourceID *uuid.UUID) ([]models.AccountingJournal, error) {
	var journals []models.AccountingJournal
	query := r.db.
		Preload("Lines").
		Preload("Lines.Account").
		Order("journal_date DESC, created_at DESC")
	if strings.TrimSpace(sourceType) != "" {
		query = query.Where("source_type = ?", strings.TrimSpace(sourceType))
	}
	if sourceID != nil {
		query = query.Where("source_id = ?", *sourceID)
	}
	err := query.Find(&journals).Error
	return journals, err
}

func (r *inventoryRepository) FindInventoryValuation() ([]models.InventoryStock, error) {
	return r.FindStocks(nil, nil)
}

func (r *inventoryRepository) UpsertPeriodLock(lock *models.AccountingPeriodLock) error {
	lock.Period = strings.TrimSpace(lock.Period)
	if lock.Period == "" {
		return errors.New("period is required")
	}
	now := time.Now()
	if lock.IsLocked && lock.LockedAt == nil {
		lock.LockedAt = &now
	}
	var existing models.AccountingPeriodLock
	err := r.db.First(&existing, "period = ?", lock.Period).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(lock).Error
	}
	if err != nil {
		return err
	}
	existing.IsLocked = lock.IsLocked
	existing.LockedBy = lock.LockedBy
	existing.LockedAt = lock.LockedAt
	existing.Notes = lock.Notes
	return r.db.Save(&existing).Error
}

func (r *inventoryRepository) FindPeriodLocks() ([]models.AccountingPeriodLock, error) {
	var locks []models.AccountingPeriodLock
	err := r.db.Order("period DESC").Find(&locks).Error
	return locks, err
}

func (r *inventoryRepository) CreateSupplierInvoice(invoice *models.SupplierInvoice) error {
	return r.db.Create(invoice).Error
}

func (r *inventoryRepository) FindSupplierInvoices(status string, supplierID *uuid.UUID) ([]models.SupplierInvoice, error) {
	if err := backfillSupplierInvoicesFromGoodsReceipts(r.db); err != nil {
		return nil, err
	}
	var invoices []models.SupplierInvoice
	query := r.db.
		Preload("Supplier").
		Preload("GoodsReceipt").
		Preload("Payments").
		Order("invoice_date DESC, created_at DESC")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	if supplierID != nil {
		query = query.Where("supplier_id = ?", *supplierID)
	}
	err := query.Find(&invoices).Error
	return invoices, err
}

func (r *inventoryRepository) FindSupplierInvoiceByID(id uuid.UUID) (*models.SupplierInvoice, error) {
	var invoice models.SupplierInvoice
	err := r.db.
		Preload("Supplier").
		Preload("GoodsReceipt").
		Preload("Payments").
		Preload("Payments.Supplier").
		First(&invoice, "id = ?", id).Error
	return &invoice, err
}

func (r *inventoryRepository) CreateSupplierPayment(payment *models.SupplierPayment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var invoice models.SupplierInvoice
		if err := tx.First(&invoice, "id = ?", payment.SupplierInvoiceID).Error; err != nil {
			return err
		}
		if invoice.Status == "paid" || invoice.Status == "cancelled" {
			return errors.New("supplier invoice is not payable")
		}
		if payment.Amount <= 0 {
			return errors.New("payment amount must be greater than zero")
		}
		if payment.Amount > invoice.OutstandingAmount {
			return errors.New("payment amount exceeds outstanding invoice")
		}
		payment.SupplierID = invoice.SupplierID
		if err := tx.Create(payment).Error; err != nil {
			return err
		}
		invoice.PaidAmount += payment.Amount
		invoice.OutstandingAmount = invoice.GrandTotal - invoice.PaidAmount
		if invoice.OutstandingAmount <= 0 {
			invoice.OutstandingAmount = 0
			invoice.Status = "paid"
		} else {
			invoice.Status = "partially_paid"
		}
		if err := tx.Omit(clause.Associations).Save(&invoice).Error; err != nil {
			return err
		}
		return postSupplierPaymentJournal(tx, payment)
	})
}

func (r *inventoryRepository) FindSupplierPayments(invoiceID *uuid.UUID, supplierID *uuid.UUID) ([]models.SupplierPayment, error) {
	var payments []models.SupplierPayment
	query := r.db.
		Preload("Supplier").
		Preload("SupplierInvoice").
		Order("payment_date DESC, created_at DESC")
	if invoiceID != nil {
		query = query.Where("supplier_invoice_id = ?", *invoiceID)
	}
	if supplierID != nil {
		query = query.Where("supplier_id = ?", *supplierID)
	}
	err := query.Find(&payments).Error
	return payments, err
}

type accountingLineInput struct {
	AccountCode string
	Debit       float64
	Credit      float64
	EntityType  string
	EntityID    *uuid.UUID
	Memo        string
}

func postGoodsReceiptJournal(tx *gorm.DB, receipt *models.GoodsReceipt) error {
	lines := make([]accountingLineInput, 0, len(receipt.Items)+1)
	total := 0.0
	for _, item := range receipt.Items {
		var inventoryItem models.InventoryItem
		if err := tx.First(&inventoryItem, "id = ?", item.ItemID).Error; err != nil {
			return err
		}
		total += item.Total
		itemID := item.ItemID
		lines = append(lines, accountingLineInput{
			AccountCode: inventoryAccountCode(inventoryItem.AccountingType),
			Debit:       item.Total,
			EntityType:  "inventory_item",
			EntityID:    &itemID,
			Memo:        inventoryItem.Name,
		})
	}
	lines = append(lines, accountingLineInput{AccountCode: "2000", Credit: total, EntityType: "supplier", EntityID: &receipt.SupplierID, Memo: "Supplier payable"})
	return postAccountingJournal(tx, "goods_receipt", &receipt.ID, receipt.ReceivedAt, "Goods receipt "+receipt.ReceiptNumber, receipt.ReceivedBy, lines)
}

func createSupplierInvoiceFromGoodsReceipt(tx *gorm.DB, receipt *models.GoodsReceipt) error {
	var existing models.SupplierInvoice
	err := tx.First(&existing, "goods_receipt_id = ?", receipt.ID).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	total := 0.0
	for _, item := range receipt.Items {
		total += item.Total
	}
	if total <= 0 {
		return nil
	}
	invoiceDate := receipt.ReceivedAt
	if invoiceDate.IsZero() {
		invoiceDate = time.Now()
	}
	dueDate := invoiceDate.AddDate(0, 0, 30)
	invoiceNumber := strings.TrimSpace(receipt.SupplierInvoiceNumber)
	if invoiceNumber == "" {
		invoiceNumber = "INV-" + strings.TrimSpace(receipt.ReceiptNumber)
	}
	invoice := models.SupplierInvoice{
		SupplierID:        receipt.SupplierID,
		GoodsReceiptID:    &receipt.ID,
		InvoiceNumber:     invoiceNumber,
		InvoiceDate:       invoiceDate,
		DueDate:           &dueDate,
		Status:            "posted",
		Subtotal:          total,
		GrandTotal:        total,
		OutstandingAmount: total,
		Notes:             "Auto generated from goods receipt " + receipt.ReceiptNumber,
	}
	return tx.Create(&invoice).Error
}

func backfillSupplierInvoicesFromGoodsReceipts(tx *gorm.DB) error {
	var receipts []models.GoodsReceipt
	err := tx.
		Preload("Items").
		Where("status <> ?", "cancelled").
		Where("NOT EXISTS (SELECT 1 FROM supplier_invoices WHERE supplier_invoices.goods_receipt_id = goods_receipts.id AND supplier_invoices.deleted_at IS NULL)").
		Find(&receipts).Error
	if err != nil {
		return err
	}
	for i := range receipts {
		if err := createSupplierInvoiceFromGoodsReceipt(tx, &receipts[i]); err != nil {
			return err
		}
	}
	return nil
}

func postSupplierPaymentJournal(tx *gorm.DB, payment *models.SupplierPayment) error {
	cashCode := strings.TrimSpace(payment.CashAccountCode)
	if cashCode == "" {
		cashCode = "1000"
	}
	return postAccountingJournal(tx, "supplier_payment", &payment.ID, payment.PaymentDate, "Supplier payment "+payment.PaymentNumber, payment.CreatedBy, []accountingLineInput{
		{AccountCode: "2000", Debit: payment.Amount, EntityType: "supplier", EntityID: &payment.SupplierID, Memo: payment.PaymentMethod},
		{AccountCode: cashCode, Credit: payment.Amount, EntityType: "supplier", EntityID: &payment.SupplierID, Memo: payment.ReferenceNumber},
	})
}

func postMaterialUsageJournal(tx *gorm.DB, usage *models.MaterialUsage) error {
	lines := make([]accountingLineInput, 0, len(usage.Items)*2)
	for _, item := range usage.Items {
		if item.TotalCost == 0 {
			continue
		}
		var inventoryItem models.InventoryItem
		if err := tx.First(&inventoryItem, "id = ?", item.ItemID).Error; err != nil {
			return err
		}
		debitCode := materialUsageDebitAccountCode(inventoryItem.AccountingType, usage.UsageType)
		itemID := item.ItemID
		lines = append(lines,
			accountingLineInput{AccountCode: debitCode, Debit: item.TotalCost, EntityType: "inventory_item", EntityID: &itemID, Memo: usage.UsageType},
			accountingLineInput{AccountCode: inventoryAccountCode(inventoryItem.AccountingType), Credit: item.TotalCost, EntityType: "inventory_item", EntityID: &itemID, Memo: inventoryItem.Name},
		)
	}
	return postAccountingJournal(tx, "material_usage", &usage.ID, usage.UsedAt, "Material usage "+usage.UsageNumber, usage.TechnicianUserID, lines)
}

func postAssetStatusJournal(tx *gorm.DB, sourceType string, sourceID uuid.UUID, itemID uuid.UUID, status string, notes string) error {
	if status != "lost" && status != "scrap" && status != "damaged" {
		return nil
	}
	var item models.InventoryItem
	if err := tx.First(&item, "id = ?", itemID).Error; err != nil {
		return err
	}
	amount := item.DefaultCost
	if amount <= 0 {
		amount = item.SalePrice
	}
	if amount <= 0 {
		return nil
	}
	return postAccountingJournal(tx, sourceType, &sourceID, time.Now(), "Asset marked "+status, nil, []accountingLineInput{
		{AccountCode: "5030", Debit: amount, EntityType: "inventory_item", EntityID: &itemID, Memo: notes},
		{AccountCode: materialUsageDebitAccountCode(item.AccountingType, ""), Credit: amount, EntityType: "inventory_item", EntityID: &itemID, Memo: item.Name},
	})
}

func postStockOpnameJournal(tx *gorm.DB, opname *models.StockOpname) error {
	lines := make([]accountingLineInput, 0, len(opname.Items)*2)
	for _, item := range opname.Items {
		if item.VarianceQuantity == 0 || item.VarianceCost == 0 {
			continue
		}
		var inventoryItem models.InventoryItem
		if err := tx.First(&inventoryItem, "id = ?", item.ItemID).Error; err != nil {
			return err
		}
		amount := mathAbsFloat(item.VarianceCost)
		itemID := item.ItemID
		if item.VarianceQuantity > 0 {
			lines = append(lines,
				accountingLineInput{AccountCode: inventoryAccountCode(inventoryItem.AccountingType), Debit: amount, EntityType: "inventory_item", EntityID: &itemID, Memo: item.Notes},
				accountingLineInput{AccountCode: "4030", Credit: amount, EntityType: "inventory_item", EntityID: &itemID, Memo: "Adjustment gain"},
			)
		} else {
			lines = append(lines,
				accountingLineInput{AccountCode: "5030", Debit: amount, EntityType: "inventory_item", EntityID: &itemID, Memo: item.Notes},
				accountingLineInput{AccountCode: inventoryAccountCode(inventoryItem.AccountingType), Credit: amount, EntityType: "inventory_item", EntityID: &itemID, Memo: "Adjustment loss"},
			)
		}
	}
	return postAccountingJournal(tx, "stock_opname", &opname.ID, time.Now(), "Stock opname "+opname.OpnameNumber, opname.ApprovedBy, lines)
}

func postAccountingJournal(tx *gorm.DB, sourceType string, sourceID *uuid.UUID, journalDate time.Time, description string, postedBy *uuid.UUID, inputs []accountingLineInput) error {
	if len(inputs) == 0 {
		return nil
	}
	if journalDate.IsZero() {
		journalDate = time.Now()
	}
	if err := ensureAccountingPeriodOpen(tx, journalDate); err != nil {
		return err
	}
	if err := ensureDefaultChartOfAccounts(tx); err != nil {
		return err
	}
	var existing models.AccountingJournal
	if sourceID != nil {
		err := tx.First(&existing, "source_type = ? AND source_id = ?", sourceType, *sourceID).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	now := time.Now()
	journal := models.AccountingJournal{
		JournalNumber: fmt.Sprintf("JRN-%s-%d", journalDate.Format("20060102"), now.UnixNano()),
		JournalDate:   journalDate,
		SourceType:    sourceType,
		SourceID:      sourceID,
		Description:   description,
		Status:        "posted",
		PostedBy:      postedBy,
		PostedAt:      &now,
	}
	if err := tx.Create(&journal).Error; err != nil {
		return err
	}
	for _, input := range inputs {
		if input.Debit == 0 && input.Credit == 0 {
			continue
		}
		account, err := findAccountByCode(tx, input.AccountCode)
		if err != nil {
			return err
		}
		line := models.AccountingJournalLine{
			JournalID:  journal.ID,
			AccountID:  account.ID,
			Debit:      input.Debit,
			Credit:     input.Credit,
			EntityType: input.EntityType,
			EntityID:   input.EntityID,
			Memo:       input.Memo,
		}
		if err := tx.Create(&line).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureAccountingPeriodOpen(tx *gorm.DB, date time.Time) error {
	period := date.Format("2006-01")
	var lock models.AccountingPeriodLock
	err := tx.First(&lock, "period = ? AND is_locked = ?", period, true).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("accounting period %s is locked", period)
}

func findAccountByCode(tx *gorm.DB, code string) (*models.ChartOfAccount, error) {
	var account models.ChartOfAccount
	err := tx.First(&account, "code = ?", code).Error
	return &account, err
}

func ensureDefaultChartOfAccounts(tx *gorm.DB) error {
	defaults := []models.ChartOfAccount{
		{Code: "1000", Name: "Cash and Bank", Type: "asset", IsActive: true},
		{Code: "1100", Name: "Inventory - Consumables", Type: "asset", IsActive: true},
		{Code: "1110", Name: "Inventory - Saleable Goods", Type: "asset", IsActive: true},
		{Code: "1200", Name: "Customer Equipment Assets", Type: "asset", IsActive: true},
		{Code: "1210", Name: "Network Infrastructure Assets", Type: "asset", IsActive: true},
		{Code: "1220", Name: "Tools and Equipment", Type: "asset", IsActive: true},
		{Code: "2000", Name: "Accounts Payable - Suppliers", Type: "liability", IsActive: true},
		{Code: "4030", Name: "Inventory Adjustment Gain", Type: "income", IsActive: true},
		{Code: "5000", Name: "Installation Material Expense", Type: "expense", IsActive: true},
		{Code: "5010", Name: "Network Maintenance Expense", Type: "expense", IsActive: true},
		{Code: "5020", Name: "Cost of Goods Sold", Type: "expense", IsActive: true},
		{Code: "5030", Name: "Inventory Adjustment Loss", Type: "expense", IsActive: true},
		{Code: "5040", Name: "General Inventory Expense", Type: "expense", IsActive: true},
	}
	for _, account := range defaults {
		var existing models.ChartOfAccount
		err := tx.First(&existing, "code = ?", account.Code).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func inventoryAccountCode(accountingType string) string {
	switch strings.TrimSpace(accountingType) {
	case "stock_saleable":
		return "1110"
	case "customer_asset":
		return "1200"
	case "network_asset":
		return "1210"
	case "tool_asset":
		return "1220"
	default:
		return "1100"
	}
}

func materialUsageDebitAccountCode(accountingType string, usageType string) string {
	switch strings.TrimSpace(accountingType) {
	case "customer_asset":
		return "1200"
	case "network_asset":
		return "1210"
	case "stock_saleable":
		return "5020"
	}
	switch strings.TrimSpace(usageType) {
	case "network_build", "network_repair":
		return "5010"
	case "internal_use":
		return "5040"
	default:
		return "5000"
	}
}
