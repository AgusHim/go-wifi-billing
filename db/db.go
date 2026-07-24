package db

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Agushim/go_wifi_billing/models"
)

func InitDB(postgresDsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if postgresDsn != "" {
		dialector = postgres.Open(postgresDsn)
	} else {
		// fallback to sqlite
		dialector = sqlite.Open("test.db")
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Enable UUID extension for PostgreSQL
	if postgresDsn != "" {
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
			return nil, err
		}
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	return db.AutoMigrate(
		&models.Coverage{},
		&models.Package{},
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.RolePermission{},
		&models.UserPermissionOverride{},
		&models.AccessAuditLog{},
		&models.Odc{},
		&models.Odp{},
		&models.Router{},
		&models.RouterSnapshot{},
		&models.RouterInterfaceSnapshot{},
		&models.ServiceSessionSnapshot{},
		&models.RouterEventLog{},
		&models.RouterMetricAggregate{},
		&models.AlertRule{},
		&models.AlertEvent{},
		&models.AlertNotification{},
		&models.IncidentTicket{},
		&models.RouterImportBatch{},
		&models.RouterImportItem{},
		&models.NetworkPlan{},
		&models.VoucherBatch{},
		&models.Voucher{},
		&models.Subscription{},
		&models.SubscriptionRenewalHistory{},
		&models.ServiceAccount{},
		&models.ServiceStatusHistory{},
		&models.ReconciliationFinding{},
		&models.ProvisioningJob{},
		&models.ProvisioningLog{},
		&models.Bill{},
		&models.Payment{},
		&models.PaymentCallbackLog{},
		&models.BillingAutomationRun{},
		&models.Customer{},
		&models.Complain{},
		&models.WhatsAppTemplate{},
		&models.Expense{},
		&models.Setting{},
		&models.InventoryItem{},
		&models.InventoryLocation{},
		&models.InventoryStock{},
		&models.InventoryStockMovement{},
		&models.InventorySerialItem{},
		&models.Supplier{},
		&models.PurchaseOrder{},
		&models.PurchaseOrderItem{},
		&models.GoodsReceipt{},
		&models.GoodsReceiptItem{},
		&models.StockTransfer{},
		&models.StockTransferItem{},
		&models.MaterialUsage{},
		&models.MaterialUsageItem{},
		&models.CustomerAsset{},
		&models.NetworkAsset{},
		&models.StockOpname{},
		&models.StockOpnameItem{},
		&models.ChartOfAccount{},
		&models.AccountingJournal{},
		&models.AccountingJournalLine{},
		&models.AccountingPeriodLock{},
		&models.SupplierInvoice{},
		&models.SupplierPayment{},
	)
}
