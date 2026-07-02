package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryService interface {
	CreateItem(item *models.InventoryItem) (*models.InventoryItem, error)
	GetItems(search string, active *bool) ([]models.InventoryItem, error)
	GetItemByID(id uuid.UUID) (*models.InventoryItem, error)
	UpdateItem(id uuid.UUID, input *models.InventoryItem) (*models.InventoryItem, error)
	DeleteItem(id uuid.UUID) error

	CreateLocation(location *models.InventoryLocation) (*models.InventoryLocation, error)
	GetLocations(locationType string, active *bool) ([]models.InventoryLocation, error)
	GetLocationByID(id uuid.UUID) (*models.InventoryLocation, error)
	UpdateLocation(id uuid.UUID, input *models.InventoryLocation) (*models.InventoryLocation, error)
	DeleteLocation(id uuid.UUID) error

	GetStocks(itemID *uuid.UUID, locationID *uuid.UUID) ([]models.InventoryStock, error)
	GetMovements(itemID *uuid.UUID, locationID *uuid.UUID, movementType string, limit int) ([]models.InventoryStockMovement, error)
	GetSerialItems(itemID *uuid.UUID, status string) ([]models.InventorySerialItem, error)

	CreateSupplier(supplier *models.Supplier) (*models.Supplier, error)
	GetSuppliers(search string, active *bool) ([]models.Supplier, error)
	GetSupplierByID(id uuid.UUID) (*models.Supplier, error)
	UpdateSupplier(id uuid.UUID, input *models.Supplier) (*models.Supplier, error)
	DeleteSupplier(id uuid.UUID) error

	CreatePurchaseOrder(po *models.PurchaseOrder) (*models.PurchaseOrder, error)
	GetPurchaseOrders(status string) ([]models.PurchaseOrder, error)
	GetPurchaseOrderByID(id uuid.UUID) (*models.PurchaseOrder, error)
	UpdatePurchaseOrder(id uuid.UUID, input *models.PurchaseOrder) (*models.PurchaseOrder, error)

	ReceiveGoods(receipt *models.GoodsReceipt) (*models.GoodsReceipt, error)
	GetGoodsReceipts(status string) ([]models.GoodsReceipt, error)
	GetGoodsReceiptByID(id uuid.UUID) (*models.GoodsReceipt, error)

	CreateStockTransfer(transfer *models.StockTransfer) (*models.StockTransfer, error)
	GetStockTransfers(status string) ([]models.StockTransfer, error)
	GetStockTransferByID(id uuid.UUID) (*models.StockTransfer, error)
	ReceiveStockTransfer(id uuid.UUID, receivedBy *uuid.UUID) (*models.StockTransfer, error)

	CreateMaterialUsage(usage *models.MaterialUsage) (*models.MaterialUsage, error)
	GetMaterialUsages(referenceType string, referenceID *uuid.UUID) ([]models.MaterialUsage, error)
	GetMaterialUsageByID(id uuid.UUID) (*models.MaterialUsage, error)

	GetCustomerAssets(customerID *uuid.UUID, status string) ([]models.CustomerAsset, error)
	ReturnCustomerAsset(id uuid.UUID, returnLocationID uuid.UUID, notes string) (*models.CustomerAsset, error)
	ReplaceCustomerAsset(id uuid.UUID, newSerialItemID uuid.UUID, locationID uuid.UUID, notes string) (*models.CustomerAsset, error)
	UpdateCustomerAssetStatus(id uuid.UUID, status string, notes string) (*models.CustomerAsset, error)
	GetNetworkAssets(referenceType string, referenceID *uuid.UUID, status string) ([]models.NetworkAsset, error)
	UpdateNetworkAssetStatus(id uuid.UUID, status string, notes string) (*models.NetworkAsset, error)

	CreateStockOpname(opname *models.StockOpname) (*models.StockOpname, error)
	GetStockOpnames(locationID *uuid.UUID, status string) ([]models.StockOpname, error)
	GetStockOpnameByID(id uuid.UUID) (*models.StockOpname, error)
	SubmitStockOpname(id uuid.UUID, submittedBy *uuid.UUID) (*models.StockOpname, error)
	ApproveStockOpname(id uuid.UUID, approvedBy *uuid.UUID) (*models.StockOpname, error)

	GetChartOfAccounts() ([]models.ChartOfAccount, error)
	GetAccountingJournals(sourceType string, sourceID *uuid.UUID) ([]models.AccountingJournal, error)
	GetInventoryValuation() ([]models.InventoryStock, error)
	UpsertPeriodLock(lock *models.AccountingPeriodLock) (*models.AccountingPeriodLock, error)
	GetPeriodLocks() ([]models.AccountingPeriodLock, error)
}

type inventoryService struct {
	repo repositories.InventoryRepository
}

func NewInventoryService(repo repositories.InventoryRepository) InventoryService {
	return &inventoryService{repo: repo}
}

func (s *inventoryService) CreateItem(item *models.InventoryItem) (*models.InventoryItem, error) {
	if err := validateInventoryItem(item); err != nil {
		return nil, err
	}
	if err := s.ensureSKUUnique(item.SKU, nil); err != nil {
		return nil, err
	}
	if err := s.repo.CreateItem(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *inventoryService) GetItems(search string, active *bool) ([]models.InventoryItem, error) {
	return s.repo.FindItems(search, active)
}

func (s *inventoryService) GetItemByID(id uuid.UUID) (*models.InventoryItem, error) {
	return s.repo.FindItemByID(id)
}

func (s *inventoryService) UpdateItem(id uuid.UUID, input *models.InventoryItem) (*models.InventoryItem, error) {
	if err := validateInventoryItem(input); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindItemByID(id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSKUUnique(input.SKU, &id); err != nil {
		return nil, err
	}
	existing.SKU = strings.TrimSpace(input.SKU)
	existing.Name = strings.TrimSpace(input.Name)
	existing.Category = strings.TrimSpace(input.Category)
	existing.AccountingType = strings.TrimSpace(input.AccountingType)
	existing.Unit = strings.TrimSpace(input.Unit)
	existing.TrackSerial = input.TrackSerial
	existing.TrackMACAddress = input.TrackMACAddress
	existing.IsActive = input.IsActive
	existing.MinimumStock = input.MinimumStock
	existing.DefaultCost = input.DefaultCost
	existing.SalePrice = input.SalePrice
	existing.Description = input.Description
	if err := s.repo.UpdateItem(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *inventoryService) DeleteItem(id uuid.UUID) error {
	return s.repo.DeleteItem(id)
}

func (s *inventoryService) CreateLocation(location *models.InventoryLocation) (*models.InventoryLocation, error) {
	if err := validateInventoryLocation(location); err != nil {
		return nil, err
	}
	if err := s.repo.CreateLocation(location); err != nil {
		return nil, err
	}
	return s.repo.FindLocationByID(location.ID)
}

func (s *inventoryService) GetLocations(locationType string, active *bool) ([]models.InventoryLocation, error) {
	return s.repo.FindLocations(locationType, active)
}

func (s *inventoryService) GetLocationByID(id uuid.UUID) (*models.InventoryLocation, error) {
	return s.repo.FindLocationByID(id)
}

func (s *inventoryService) UpdateLocation(id uuid.UUID, input *models.InventoryLocation) (*models.InventoryLocation, error) {
	if err := validateInventoryLocation(input); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindLocationByID(id)
	if err != nil {
		return nil, err
	}
	existing.Name = strings.TrimSpace(input.Name)
	existing.Type = strings.TrimSpace(input.Type)
	existing.CoverageID = input.CoverageID
	existing.TechnicianUserID = input.TechnicianUserID
	existing.IsActive = input.IsActive
	if err := s.repo.UpdateLocation(existing); err != nil {
		return nil, err
	}
	return s.repo.FindLocationByID(id)
}

func (s *inventoryService) DeleteLocation(id uuid.UUID) error {
	return s.repo.DeleteLocation(id)
}

func (s *inventoryService) GetStocks(itemID *uuid.UUID, locationID *uuid.UUID) ([]models.InventoryStock, error) {
	return s.repo.FindStocks(itemID, locationID)
}

func (s *inventoryService) GetMovements(itemID *uuid.UUID, locationID *uuid.UUID, movementType string, limit int) ([]models.InventoryStockMovement, error) {
	return s.repo.FindMovements(itemID, locationID, movementType, limit)
}

func (s *inventoryService) GetSerialItems(itemID *uuid.UUID, status string) ([]models.InventorySerialItem, error) {
	return s.repo.FindSerialItems(itemID, status)
}

func (s *inventoryService) CreateSupplier(supplier *models.Supplier) (*models.Supplier, error) {
	if err := validateSupplier(supplier); err != nil {
		return nil, err
	}
	if err := s.repo.CreateSupplier(supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *inventoryService) GetSuppliers(search string, active *bool) ([]models.Supplier, error) {
	return s.repo.FindSuppliers(search, active)
}

func (s *inventoryService) GetSupplierByID(id uuid.UUID) (*models.Supplier, error) {
	return s.repo.FindSupplierByID(id)
}

func (s *inventoryService) UpdateSupplier(id uuid.UUID, input *models.Supplier) (*models.Supplier, error) {
	if err := validateSupplier(input); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindSupplierByID(id)
	if err != nil {
		return nil, err
	}
	existing.Name = strings.TrimSpace(input.Name)
	existing.Phone = strings.TrimSpace(input.Phone)
	existing.Email = strings.TrimSpace(input.Email)
	existing.Address = strings.TrimSpace(input.Address)
	existing.TaxNumber = strings.TrimSpace(input.TaxNumber)
	existing.PaymentTerms = strings.TrimSpace(input.PaymentTerms)
	existing.IsActive = input.IsActive
	if err := s.repo.UpdateSupplier(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *inventoryService) DeleteSupplier(id uuid.UUID) error {
	return s.repo.DeleteSupplier(id)
}

func (s *inventoryService) CreatePurchaseOrder(po *models.PurchaseOrder) (*models.PurchaseOrder, error) {
	if err := s.preparePurchaseOrder(po); err != nil {
		return nil, err
	}
	if err := s.repo.CreatePurchaseOrder(po); err != nil {
		return nil, err
	}
	return s.repo.FindPurchaseOrderByID(po.ID)
}

func (s *inventoryService) GetPurchaseOrders(status string) ([]models.PurchaseOrder, error) {
	return s.repo.FindPurchaseOrders(status)
}

func (s *inventoryService) GetPurchaseOrderByID(id uuid.UUID) (*models.PurchaseOrder, error) {
	return s.repo.FindPurchaseOrderByID(id)
}

func (s *inventoryService) UpdatePurchaseOrder(id uuid.UUID, input *models.PurchaseOrder) (*models.PurchaseOrder, error) {
	existing, err := s.repo.FindPurchaseOrderByID(id)
	if err != nil {
		return nil, err
	}
	if existing.Status == "received" || existing.Status == "partially_received" {
		return nil, errors.New("PO yang sudah diterima tidak dapat diubah")
	}
	input.ID = id
	if err := s.preparePurchaseOrder(input); err != nil {
		return nil, err
	}
	existing.SupplierID = input.SupplierID
	existing.PONumber = input.PONumber
	existing.Status = input.Status
	existing.OrderDate = input.OrderDate
	existing.ExpectedDate = input.ExpectedDate
	existing.Subtotal = input.Subtotal
	existing.Discount = input.Discount
	existing.Tax = input.Tax
	existing.ShippingCost = input.ShippingCost
	existing.GrandTotal = input.GrandTotal
	existing.Items = input.Items
	if err := s.repo.UpdatePurchaseOrder(existing); err != nil {
		return nil, err
	}
	return s.repo.FindPurchaseOrderByID(id)
}

func (s *inventoryService) ReceiveGoods(receipt *models.GoodsReceipt) (*models.GoodsReceipt, error) {
	if receipt == nil {
		return nil, errors.New("invalid goods receipt payload")
	}
	po, err := s.repo.FindPurchaseOrderByID(receipt.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	if po.Status == "cancelled" {
		return nil, errors.New("PO cancelled tidak bisa diterima")
	}
	if err := s.prepareGoodsReceipt(receipt, po); err != nil {
		return nil, err
	}
	if err := s.repo.CreateGoodsReceiptWithStock(receipt, po); err != nil {
		return nil, err
	}
	return s.repo.FindGoodsReceiptByID(receipt.ID)
}

func (s *inventoryService) GetGoodsReceipts(status string) ([]models.GoodsReceipt, error) {
	return s.repo.FindGoodsReceipts(status)
}

func (s *inventoryService) GetGoodsReceiptByID(id uuid.UUID) (*models.GoodsReceipt, error) {
	return s.repo.FindGoodsReceiptByID(id)
}

func (s *inventoryService) CreateStockTransfer(transfer *models.StockTransfer) (*models.StockTransfer, error) {
	if err := s.prepareStockTransfer(transfer); err != nil {
		return nil, err
	}
	if err := s.repo.CreateStockTransferWithReservation(transfer); err != nil {
		return nil, err
	}
	return s.repo.FindStockTransferByID(transfer.ID)
}

func (s *inventoryService) GetStockTransfers(status string) ([]models.StockTransfer, error) {
	return s.repo.FindStockTransfers(status)
}

func (s *inventoryService) GetStockTransferByID(id uuid.UUID) (*models.StockTransfer, error) {
	return s.repo.FindStockTransferByID(id)
}

func (s *inventoryService) ReceiveStockTransfer(id uuid.UUID, receivedBy *uuid.UUID) (*models.StockTransfer, error) {
	transfer, err := s.repo.FindStockTransferByID(id)
	if err != nil {
		return nil, err
	}
	if transfer.Status != "sent" {
		return nil, errors.New("hanya transfer status sent yang dapat diterima")
	}
	if err := s.repo.ReceiveStockTransfer(transfer, receivedBy); err != nil {
		return nil, err
	}
	return s.repo.FindStockTransferByID(id)
}

func (s *inventoryService) CreateMaterialUsage(usage *models.MaterialUsage) (*models.MaterialUsage, error) {
	if err := s.prepareMaterialUsage(usage); err != nil {
		return nil, err
	}
	if err := s.repo.CreateMaterialUsageWithStock(usage); err != nil {
		return nil, err
	}
	return s.repo.FindMaterialUsageByID(usage.ID)
}

func (s *inventoryService) GetMaterialUsages(referenceType string, referenceID *uuid.UUID) ([]models.MaterialUsage, error) {
	return s.repo.FindMaterialUsages(referenceType, referenceID)
}

func (s *inventoryService) GetMaterialUsageByID(id uuid.UUID) (*models.MaterialUsage, error) {
	return s.repo.FindMaterialUsageByID(id)
}

func (s *inventoryService) GetCustomerAssets(customerID *uuid.UUID, status string) ([]models.CustomerAsset, error) {
	return s.repo.FindCustomerAssets(customerID, status)
}

func (s *inventoryService) ReturnCustomerAsset(id uuid.UUID, returnLocationID uuid.UUID, notes string) (*models.CustomerAsset, error) {
	if returnLocationID == uuid.Nil {
		return nil, errors.New("return_location_id is required")
	}
	if _, err := s.repo.FindLocationByID(returnLocationID); err != nil {
		return nil, err
	}
	asset, err := s.repo.FindCustomerAssetByID(id)
	if err != nil {
		return nil, err
	}
	if asset.Status != "installed" {
		return nil, errors.New("hanya asset installed yang dapat direturn")
	}
	if err := s.repo.ReturnCustomerAsset(asset, returnLocationID, notes); err != nil {
		return nil, err
	}
	return s.repo.FindCustomerAssetByID(id)
}

func (s *inventoryService) ReplaceCustomerAsset(id uuid.UUID, newSerialItemID uuid.UUID, locationID uuid.UUID, notes string) (*models.CustomerAsset, error) {
	if newSerialItemID == uuid.Nil || locationID == uuid.Nil {
		return nil, errors.New("new_serial_item_id and location_id are required")
	}
	asset, err := s.repo.FindCustomerAssetByID(id)
	if err != nil {
		return nil, err
	}
	if asset.Status != "installed" {
		return nil, errors.New("hanya asset installed yang dapat direplace")
	}
	serial, err := s.repo.FindSerialItemByID(newSerialItemID)
	if err != nil {
		return nil, err
	}
	if serial.CurrentLocationID == nil || *serial.CurrentLocationID != locationID {
		return nil, errors.New("serial pengganti tidak berada di lokasi sumber")
	}
	if serial.Status == "installed_customer" || serial.Status == "installed_network" || serial.Status == "lost" || serial.Status == "scrap" {
		return nil, errors.New("serial pengganti tidak available")
	}
	if err := s.repo.ReplaceCustomerAsset(asset, serial, locationID, notes); err != nil {
		return nil, err
	}
	assets, err := s.repo.FindCustomerAssets(&asset.CustomerID, "installed")
	if err != nil || len(assets) == 0 {
		return s.repo.FindCustomerAssetByID(id)
	}
	return &assets[0], nil
}

func (s *inventoryService) UpdateCustomerAssetStatus(id uuid.UUID, status string, notes string) (*models.CustomerAsset, error) {
	if !validAssetTerminalStatus(status) {
		return nil, errors.New("unsupported asset status")
	}
	asset, err := s.repo.FindCustomerAssetByID(id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCustomerAssetStatus(asset, status, notes); err != nil {
		return nil, err
	}
	return s.repo.FindCustomerAssetByID(id)
}

func (s *inventoryService) GetNetworkAssets(referenceType string, referenceID *uuid.UUID, status string) ([]models.NetworkAsset, error) {
	return s.repo.FindNetworkAssets(referenceType, referenceID, status)
}

func (s *inventoryService) UpdateNetworkAssetStatus(id uuid.UUID, status string, notes string) (*models.NetworkAsset, error) {
	if !validAssetTerminalStatus(status) {
		return nil, errors.New("unsupported asset status")
	}
	asset, err := s.repo.FindNetworkAssetByID(id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateNetworkAssetStatus(asset, status, notes); err != nil {
		return nil, err
	}
	return s.repo.FindNetworkAssetByID(id)
}

func (s *inventoryService) CreateStockOpname(opname *models.StockOpname) (*models.StockOpname, error) {
	if err := s.prepareStockOpname(opname); err != nil {
		return nil, err
	}
	if err := s.repo.CreateStockOpname(opname); err != nil {
		return nil, err
	}
	return s.repo.FindStockOpnameByID(opname.ID)
}

func (s *inventoryService) GetStockOpnames(locationID *uuid.UUID, status string) ([]models.StockOpname, error) {
	return s.repo.FindStockOpnames(locationID, status)
}

func (s *inventoryService) GetStockOpnameByID(id uuid.UUID) (*models.StockOpname, error) {
	return s.repo.FindStockOpnameByID(id)
}

func (s *inventoryService) SubmitStockOpname(id uuid.UUID, submittedBy *uuid.UUID) (*models.StockOpname, error) {
	opname, err := s.repo.FindStockOpnameByID(id)
	if err != nil {
		return nil, err
	}
	if opname.Status != "draft" {
		return nil, errors.New("hanya opname draft yang dapat disubmit")
	}
	for _, item := range opname.Items {
		if item.CountedQuantity < 0 {
			return nil, errors.New("counted_quantity cannot be negative")
		}
		variance := item.CountedQuantity - item.SystemQuantity
		if variance != 0 && strings.TrimSpace(item.Notes) == "" {
			return nil, errors.New("reason wajib diisi untuk item yang selisih")
		}
	}
	if err := s.repo.SubmitStockOpname(opname, submittedBy); err != nil {
		return nil, err
	}
	return s.repo.FindStockOpnameByID(id)
}

func (s *inventoryService) ApproveStockOpname(id uuid.UUID, approvedBy *uuid.UUID) (*models.StockOpname, error) {
	opname, err := s.repo.FindStockOpnameByID(id)
	if err != nil {
		return nil, err
	}
	if opname.Status != "submitted" {
		return nil, errors.New("hanya opname submitted yang dapat diapprove")
	}
	for _, item := range opname.Items {
		if item.CountedQuantity < 0 {
			return nil, errors.New("counted_quantity cannot be negative")
		}
		variance := item.CountedQuantity - item.SystemQuantity
		if variance != 0 && strings.TrimSpace(item.Notes) == "" {
			return nil, errors.New("reason wajib diisi untuk adjustment")
		}
	}
	if err := s.repo.ApproveStockOpname(opname, approvedBy); err != nil {
		return nil, err
	}
	return s.repo.FindStockOpnameByID(id)
}

func (s *inventoryService) GetChartOfAccounts() ([]models.ChartOfAccount, error) {
	return s.repo.FindChartOfAccounts()
}

func (s *inventoryService) GetAccountingJournals(sourceType string, sourceID *uuid.UUID) ([]models.AccountingJournal, error) {
	return s.repo.FindAccountingJournals(sourceType, sourceID)
}

func (s *inventoryService) GetInventoryValuation() ([]models.InventoryStock, error) {
	return s.repo.FindInventoryValuation()
}

func (s *inventoryService) UpsertPeriodLock(lock *models.AccountingPeriodLock) (*models.AccountingPeriodLock, error) {
	if lock == nil {
		return nil, errors.New("invalid period lock payload")
	}
	lock.Period = strings.TrimSpace(lock.Period)
	if lock.Period == "" {
		return nil, errors.New("period is required")
	}
	if len(lock.Period) != 7 {
		return nil, errors.New("period must use YYYY-MM")
	}
	if err := s.repo.UpsertPeriodLock(lock); err != nil {
		return nil, err
	}
	locks, err := s.repo.FindPeriodLocks()
	if err != nil {
		return nil, err
	}
	for i := range locks {
		if locks[i].Period == lock.Period {
			return &locks[i], nil
		}
	}
	return lock, nil
}

func (s *inventoryService) GetPeriodLocks() ([]models.AccountingPeriodLock, error) {
	return s.repo.FindPeriodLocks()
}

func (s *inventoryService) ensureSKUUnique(sku string, exceptID *uuid.UUID) error {
	existing, err := s.repo.FindItemBySKU(sku)
	if err == nil {
		if exceptID != nil && existing.ID == *exceptID {
			return nil
		}
		return errors.New("SKU sudah digunakan")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func validateInventoryItem(item *models.InventoryItem) error {
	if item == nil {
		return errors.New("invalid inventory item payload")
	}
	item.SKU = strings.TrimSpace(item.SKU)
	item.Name = strings.TrimSpace(item.Name)
	item.AccountingType = strings.TrimSpace(item.AccountingType)
	item.Unit = strings.TrimSpace(item.Unit)
	if item.SKU == "" {
		return errors.New("sku is required")
	}
	if item.Name == "" {
		return errors.New("name is required")
	}
	if item.AccountingType == "" {
		return errors.New("accounting_type is required")
	}
	if !validInventoryAccountingType(item.AccountingType) {
		return errors.New("unsupported accounting_type")
	}
	if item.Unit == "" {
		item.Unit = "pcs"
	}
	if item.MinimumStock < 0 || item.DefaultCost < 0 || item.SalePrice < 0 {
		return errors.New("minimum_stock, default_cost, and sale_price cannot be negative")
	}
	return nil
}

func validateInventoryLocation(location *models.InventoryLocation) error {
	if location == nil {
		return errors.New("invalid inventory location payload")
	}
	location.Name = strings.TrimSpace(location.Name)
	location.Type = strings.TrimSpace(location.Type)
	if location.Name == "" {
		return errors.New("name is required")
	}
	if location.Type == "" {
		location.Type = "warehouse"
	}
	if !validInventoryLocationType(location.Type) {
		return errors.New("unsupported location type")
	}
	return nil
}

func validateSupplier(supplier *models.Supplier) error {
	if supplier == nil {
		return errors.New("invalid supplier payload")
	}
	supplier.Name = strings.TrimSpace(supplier.Name)
	supplier.Phone = strings.TrimSpace(supplier.Phone)
	supplier.Email = strings.TrimSpace(supplier.Email)
	if supplier.Name == "" {
		return errors.New("supplier name is required")
	}
	return nil
}

func (s *inventoryService) preparePurchaseOrder(po *models.PurchaseOrder) error {
	if po == nil {
		return errors.New("invalid purchase order payload")
	}
	if po.SupplierID == uuid.Nil {
		return errors.New("supplier_id is required")
	}
	if po.ID == uuid.Nil {
		po.ID = uuid.New()
	}
	if _, err := s.repo.FindSupplierByID(po.SupplierID); err != nil {
		return err
	}
	po.PONumber = strings.TrimSpace(po.PONumber)
	if po.PONumber == "" {
		po.PONumber = fmt.Sprintf("PO-%s", time.Now().Format("20060102150405"))
	}
	po.Status = strings.TrimSpace(po.Status)
	if po.Status == "" {
		po.Status = "draft"
	}
	if !validPurchaseOrderStatus(po.Status) {
		return errors.New("unsupported purchase order status")
	}
	if po.OrderDate.IsZero() {
		po.OrderDate = time.Now()
	}
	if len(po.Items) == 0 {
		return errors.New("purchase order requires at least one item")
	}
	subtotal := 0.0
	for i := range po.Items {
		item := &po.Items[i]
		if item.ItemID == uuid.Nil {
			return errors.New("item_id is required")
		}
		if _, err := s.repo.FindItemByID(item.ItemID); err != nil {
			return err
		}
		if item.Quantity <= 0 {
			return errors.New("quantity must be greater than zero")
		}
		if item.UnitCost < 0 || item.Discount < 0 || item.Tax < 0 {
			return errors.New("unit_cost, discount, and tax cannot be negative")
		}
		item.PurchaseOrderID = po.ID
		item.ReceivedQuantity = 0
		item.Total = (item.Quantity * item.UnitCost) - item.Discount + item.Tax
		subtotal += item.Quantity * item.UnitCost
	}
	po.Subtotal = subtotal
	if po.Discount < 0 || po.Tax < 0 || po.ShippingCost < 0 {
		return errors.New("discount, tax, and shipping_cost cannot be negative")
	}
	po.GrandTotal = po.Subtotal - po.Discount + po.Tax + po.ShippingCost
	return nil
}

func (s *inventoryService) prepareGoodsReceipt(receipt *models.GoodsReceipt, po *models.PurchaseOrder) error {
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	if receipt.WarehouseLocationID == uuid.Nil {
		return errors.New("warehouse_location_id is required")
	}
	if _, err := s.repo.FindLocationByID(receipt.WarehouseLocationID); err != nil {
		return err
	}
	receipt.SupplierID = po.SupplierID
	receipt.ReceiptNumber = strings.TrimSpace(receipt.ReceiptNumber)
	if receipt.ReceiptNumber == "" {
		receipt.ReceiptNumber = fmt.Sprintf("GR-%s", time.Now().Format("20060102150405"))
	}
	if receipt.ReceivedAt.IsZero() {
		receipt.ReceivedAt = time.Now()
	}
	receipt.Status = strings.TrimSpace(receipt.Status)
	if receipt.Status == "" {
		receipt.Status = "received"
	}
	if receipt.Status != "received" {
		return errors.New("goods receipt status must be received")
	}
	if len(receipt.Items) == 0 {
		return errors.New("goods receipt requires at least one item")
	}

	poItemByItemID := map[uuid.UUID]*models.PurchaseOrderItem{}
	for i := range po.Items {
		poItemByItemID[po.Items[i].ItemID] = &po.Items[i]
	}
	for i := range receipt.Items {
		item := &receipt.Items[i]
		if item.ItemID == uuid.Nil {
			return errors.New("item_id is required")
		}
		poItem, ok := poItemByItemID[item.ItemID]
		if !ok {
			return errors.New("receipt item tidak ada di purchase order")
		}
		if item.Quantity <= 0 {
			return errors.New("received quantity must be greater than zero")
		}
		remaining := poItem.Quantity - poItem.ReceivedQuantity
		if item.Quantity > remaining {
			return fmt.Errorf("received quantity untuk item %s melebihi sisa PO", item.ItemID)
		}
		if item.UnitCost < 0 {
			return errors.New("unit_cost cannot be negative")
		}
		if item.UnitCost == 0 {
			item.UnitCost = poItem.UnitCost
		}
		item.Total = item.Quantity * item.UnitCost
		item.GoodsReceiptID = receipt.ID

		inventoryItem, err := s.repo.FindItemByID(item.ItemID)
		if err != nil {
			return err
		}
		serialNumbers := splitServiceCSV(item.SerialNumbers)
		macAddresses := splitServiceCSV(item.MACAddresses)
		if inventoryItem.TrackSerial {
			if len(serialNumbers) != int(math.Round(item.Quantity)) {
				return fmt.Errorf("serial number item %s harus sama dengan quantity", inventoryItem.Name)
			}
			for _, serial := range serialNumbers {
				if _, err := s.repo.FindSerialItemByItemAndSerial(item.ItemID, serial); err == nil {
					return fmt.Errorf("serial %s sudah digunakan", serial)
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
		}
		if inventoryItem.TrackMACAddress {
			if len(macAddresses) == 0 {
				return fmt.Errorf("MAC address item %s wajib diisi", inventoryItem.Name)
			}
			if len(serialNumbers) > 0 && len(macAddresses) != len(serialNumbers) {
				return fmt.Errorf("jumlah MAC address item %s harus sama dengan serial", inventoryItem.Name)
			}
		}
	}
	return nil
}

func (s *inventoryService) prepareStockTransfer(transfer *models.StockTransfer) error {
	if transfer == nil {
		return errors.New("invalid stock transfer payload")
	}
	if transfer.ID == uuid.Nil {
		transfer.ID = uuid.New()
	}
	if transfer.FromLocationID == uuid.Nil {
		return errors.New("from_location_id is required")
	}
	if transfer.ToLocationID == uuid.Nil {
		return errors.New("to_location_id is required")
	}
	if transfer.FromLocationID == transfer.ToLocationID {
		return errors.New("lokasi asal dan tujuan tidak boleh sama")
	}
	if _, err := s.repo.FindLocationByID(transfer.FromLocationID); err != nil {
		return err
	}
	if _, err := s.repo.FindLocationByID(transfer.ToLocationID); err != nil {
		return err
	}
	transfer.TransferNumber = strings.TrimSpace(transfer.TransferNumber)
	if transfer.TransferNumber == "" {
		transfer.TransferNumber = fmt.Sprintf("TRF-%s", time.Now().Format("20060102150405"))
	}
	transfer.Status = strings.TrimSpace(transfer.Status)
	if transfer.Status == "" {
		transfer.Status = "sent"
	}
	if transfer.Status != "sent" {
		return errors.New("stock transfer baru harus berstatus sent")
	}
	now := time.Now()
	if transfer.SentAt == nil {
		transfer.SentAt = &now
	}
	if len(transfer.Items) == 0 {
		return errors.New("stock transfer requires at least one item")
	}

	for i := range transfer.Items {
		item := &transfer.Items[i]
		if item.ItemID == uuid.Nil {
			return errors.New("item_id is required")
		}
		inventoryItem, err := s.repo.FindItemByID(item.ItemID)
		if err != nil {
			return err
		}
		if item.Quantity <= 0 {
			return errors.New("quantity must be greater than zero")
		}
		if item.SerialItemID != nil {
			serial, err := s.repo.FindSerialItemByID(*item.SerialItemID)
			if err != nil {
				return err
			}
			if serial.ItemID != item.ItemID {
				return errors.New("serial item tidak sesuai dengan item")
			}
			if serial.CurrentLocationID == nil || *serial.CurrentLocationID != transfer.FromLocationID {
				return fmt.Errorf("serial %s tidak berada di lokasi asal", serial.SerialNumber)
			}
			item.Quantity = 1
		} else if inventoryItem.TrackSerial {
			return fmt.Errorf("item %s wajib memilih serial item", inventoryItem.Name)
		}
		stocks, err := s.repo.FindStocks(&item.ItemID, &transfer.FromLocationID)
		if err != nil {
			return err
		}
		if len(stocks) == 0 {
			return fmt.Errorf("stok item %s di lokasi asal tidak ditemukan", inventoryItem.Name)
		}
		available := stocks[0].QuantityOnHand - stocks[0].QuantityReserved
		if available < item.Quantity {
			return fmt.Errorf("stok item %s di lokasi asal tidak cukup", inventoryItem.Name)
		}
		item.StockTransferID = transfer.ID
		item.UnitCost = stocks[0].AverageCost
		item.TotalCost = item.Quantity * item.UnitCost
	}
	return nil
}

func (s *inventoryService) prepareMaterialUsage(usage *models.MaterialUsage) error {
	if usage == nil {
		return errors.New("invalid material usage payload")
	}
	if usage.ID == uuid.Nil {
		usage.ID = uuid.New()
	}
	usage.UsageNumber = strings.TrimSpace(usage.UsageNumber)
	if usage.UsageNumber == "" {
		usage.UsageNumber = fmt.Sprintf("MU-%s", time.Now().Format("20060102150405"))
	}
	usage.UsageType = strings.TrimSpace(usage.UsageType)
	if usage.UsageType == "" {
		usage.UsageType = "maintenance"
	}
	if !validMaterialUsageType(usage.UsageType) {
		return errors.New("unsupported material usage type")
	}
	usage.Status = strings.TrimSpace(usage.Status)
	if usage.Status == "" {
		usage.Status = "posted"
	}
	if usage.Status != "posted" {
		return errors.New("material usage baru harus berstatus posted")
	}
	if usage.LocationID == uuid.Nil {
		return errors.New("location_id is required")
	}
	if _, err := s.repo.FindLocationByID(usage.LocationID); err != nil {
		return err
	}
	if usage.UsedAt.IsZero() {
		usage.UsedAt = time.Now()
	}
	if len(usage.Items) == 0 {
		return errors.New("material usage requires at least one item")
	}

	for i := range usage.Items {
		item := &usage.Items[i]
		if item.ItemID == uuid.Nil {
			return errors.New("item_id is required")
		}
		inventoryItem, err := s.repo.FindItemByID(item.ItemID)
		if err != nil {
			return err
		}
		if item.Quantity <= 0 {
			return errors.New("quantity must be greater than zero")
		}
		if item.SerialItemID != nil {
			serial, err := s.repo.FindSerialItemByID(*item.SerialItemID)
			if err != nil {
				return err
			}
			if serial.ItemID != item.ItemID {
				return errors.New("serial item tidak sesuai dengan item")
			}
			if serial.CurrentLocationID == nil || *serial.CurrentLocationID != usage.LocationID {
				return fmt.Errorf("serial %s tidak berada di lokasi pemakaian", serial.SerialNumber)
			}
			item.Quantity = 1
		} else if inventoryItem.TrackSerial {
			return fmt.Errorf("item %s wajib memilih serial item", inventoryItem.Name)
		}
		if inventoryItem.AccountingType == "customer_asset" && usage.CustomerID == nil {
			return fmt.Errorf("item customer asset %s wajib link customer", inventoryItem.Name)
		}
		stocks, err := s.repo.FindStocks(&item.ItemID, &usage.LocationID)
		if err != nil {
			return err
		}
		if len(stocks) == 0 {
			return fmt.Errorf("stok item %s di lokasi pemakaian tidak ditemukan", inventoryItem.Name)
		}
		available := stocks[0].QuantityOnHand - stocks[0].QuantityReserved
		if available < item.Quantity {
			return fmt.Errorf("stok item %s di lokasi pemakaian tidak cukup", inventoryItem.Name)
		}
		item.MaterialUsageID = usage.ID
		item.UnitCost = stocks[0].AverageCost
		item.TotalCost = item.Quantity * item.UnitCost
		if item.CustomerChargeAmount < 0 {
			return errors.New("customer_charge_amount cannot be negative")
		}
	}
	return nil
}

func (s *inventoryService) prepareStockOpname(opname *models.StockOpname) error {
	if opname == nil {
		return errors.New("invalid stock opname payload")
	}
	if opname.ID == uuid.Nil {
		opname.ID = uuid.New()
	}
	if opname.LocationID == uuid.Nil {
		return errors.New("location_id is required")
	}
	if _, err := s.repo.FindLocationByID(opname.LocationID); err != nil {
		return err
	}
	opname.OpnameNumber = strings.TrimSpace(opname.OpnameNumber)
	if opname.OpnameNumber == "" {
		opname.OpnameNumber = fmt.Sprintf("OPN-%s", time.Now().Format("20060102150405"))
	}
	opname.Status = strings.TrimSpace(opname.Status)
	if opname.Status == "" {
		opname.Status = "draft"
	}
	if opname.Status != "draft" {
		return errors.New("stock opname baru harus berstatus draft")
	}
	if opname.StartedAt.IsZero() {
		opname.StartedAt = time.Now()
	}
	if len(opname.Items) == 0 {
		return errors.New("stock opname requires at least one item")
	}
	for i := range opname.Items {
		item := &opname.Items[i]
		if item.ItemID == uuid.Nil {
			return errors.New("item_id is required")
		}
		if _, err := s.repo.FindItemByID(item.ItemID); err != nil {
			return err
		}
		if item.CountedQuantity < 0 {
			return errors.New("counted_quantity cannot be negative")
		}
		if item.SerialItemID != nil {
			serial, err := s.repo.FindSerialItemByID(*item.SerialItemID)
			if err != nil {
				return err
			}
			if serial.ItemID != item.ItemID {
				return errors.New("serial item tidak sesuai dengan item")
			}
			if serial.CurrentLocationID == nil || *serial.CurrentLocationID != opname.LocationID {
				return fmt.Errorf("serial %s tidak berada di lokasi opname", serial.SerialNumber)
			}
		}
		stocks, err := s.repo.FindStocks(&item.ItemID, &opname.LocationID)
		if err != nil {
			return err
		}
		if len(stocks) > 0 {
			item.SystemQuantity = stocks[0].QuantityOnHand
			item.VarianceCost = (item.CountedQuantity - item.SystemQuantity) * stocks[0].AverageCost
		} else {
			item.SystemQuantity = 0
			item.VarianceCost = 0
		}
		item.StockOpnameID = opname.ID
		item.VarianceQuantity = item.CountedQuantity - item.SystemQuantity
		if item.VarianceQuantity != 0 && strings.TrimSpace(item.Notes) == "" {
			return errors.New("reason wajib diisi untuk item yang selisih")
		}
	}
	return nil
}

func validPurchaseOrderStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case "draft", "submitted", "approved", "partially_received", "received", "cancelled":
		return true
	default:
		return false
	}
}

func validMaterialUsageType(value string) bool {
	switch strings.TrimSpace(value) {
	case "new_installation", "maintenance", "upgrade", "network_build", "network_repair", "internal_use":
		return true
	default:
		return false
	}
}

func validAssetTerminalStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case "damaged", "lost", "scrap":
		return true
	default:
		return false
	}
}

func splitServiceCSV(value string) []string {
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

func validInventoryAccountingType(value string) bool {
	switch strings.TrimSpace(value) {
	case "stock_consumable", "stock_saleable", "customer_asset", "network_asset", "tool_asset", "expense_item":
		return true
	default:
		return false
	}
}

func validInventoryLocationType(value string) bool {
	switch strings.TrimSpace(value) {
	case "warehouse", "technician", "vehicle", "network_site", "customer", "scrap":
		return true
	default:
		return false
	}
}
