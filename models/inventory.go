package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryItem struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	SKU             string         `json:"sku" gorm:"uniqueIndex;not null"`
	Name            string         `json:"name" gorm:"not null"`
	Category        string         `json:"category"`
	AccountingType  string         `json:"accounting_type" gorm:"index"`
	Unit            string         `json:"unit"`
	TrackSerial     bool           `json:"track_serial"`
	TrackMACAddress bool           `json:"track_mac_address"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	MinimumStock    float64        `json:"minimum_stock"`
	DefaultCost     float64        `json:"default_cost"`
	SalePrice       float64        `json:"sale_price"`
	Description     string         `json:"description"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (i *InventoryItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type InventoryLocation struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name             string         `json:"name" gorm:"not null"`
	Type             string         `json:"type" gorm:"index"`
	CoverageID       *uuid.UUID     `json:"coverage_id" gorm:"type:uuid"`
	TechnicianUserID *uuid.UUID     `json:"technician_user_id" gorm:"type:uuid"`
	IsActive         bool           `json:"is_active" gorm:"default:true"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Coverage       *Coverage `json:"coverage,omitempty" gorm:"foreignKey:CoverageID"`
	TechnicianUser *User     `json:"technician_user,omitempty" gorm:"foreignKey:TechnicianUserID"`
}

func (i *InventoryLocation) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type InventoryStock struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	ItemID           uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;uniqueIndex:idx_inventory_stock_item_location"`
	LocationID       uuid.UUID      `json:"location_id" gorm:"type:uuid;not null;uniqueIndex:idx_inventory_stock_item_location"`
	QuantityOnHand   float64        `json:"quantity_on_hand"`
	QuantityReserved float64        `json:"quantity_reserved"`
	AverageCost      float64        `json:"average_cost"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Item     *InventoryItem     `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	Location *InventoryLocation `json:"location,omitempty" gorm:"foreignKey:LocationID"`
}

func (i *InventoryStock) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type InventoryStockMovement struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	ItemID         uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	SerialItemID   *uuid.UUID     `json:"serial_item_id" gorm:"type:uuid;index"`
	FromLocationID *uuid.UUID     `json:"from_location_id" gorm:"type:uuid;index"`
	ToLocationID   *uuid.UUID     `json:"to_location_id" gorm:"type:uuid;index"`
	MovementType   string         `json:"movement_type" gorm:"index"`
	Quantity       float64        `json:"quantity"`
	UnitCost       float64        `json:"unit_cost"`
	TotalCost      float64        `json:"total_cost"`
	ReferenceType  string         `json:"reference_type"`
	ReferenceID    *uuid.UUID     `json:"reference_id" gorm:"type:uuid;index"`
	Notes          string         `json:"notes"`
	CreatedBy      *uuid.UUID     `json:"created_by" gorm:"type:uuid"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Item         *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	SerialItem   *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
	FromLocation *InventoryLocation   `json:"from_location,omitempty" gorm:"foreignKey:FromLocationID"`
	ToLocation   *InventoryLocation   `json:"to_location,omitempty" gorm:"foreignKey:ToLocationID"`
	Creator      *User                `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (i *InventoryStockMovement) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type InventorySerialItem struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	ItemID            uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_inventory_serial_item_serial"`
	SerialNumber      string         `json:"serial_number" gorm:"uniqueIndex:idx_inventory_serial_item_serial"`
	MACAddress        string         `json:"mac_address" gorm:"index"`
	CurrentLocationID *uuid.UUID     `json:"current_location_id" gorm:"type:uuid;index"`
	Status            string         `json:"status" gorm:"index"`
	CustomerID        *uuid.UUID     `json:"customer_id" gorm:"type:uuid;index"`
	ServiceAccountID  *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	NetworkAssetID    *uuid.UUID     `json:"network_asset_id" gorm:"type:uuid;index"`
	PurchaseReceiptID *uuid.UUID     `json:"purchase_receipt_id" gorm:"type:uuid;index"`
	WarrantyExpiredAt *time.Time     `json:"warranty_expired_at"`
	Notes             string         `json:"notes"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Item            *InventoryItem     `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	CurrentLocation *InventoryLocation `json:"current_location,omitempty" gorm:"foreignKey:CurrentLocationID"`
	Customer        *Customer          `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	ServiceAccount  *ServiceAccount    `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
}

func (i *InventorySerialItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type Supplier struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name         string         `json:"name" gorm:"not null"`
	Phone        string         `json:"phone"`
	Email        string         `json:"email"`
	Address      string         `json:"address"`
	TaxNumber    string         `json:"tax_number"`
	PaymentTerms string         `json:"payment_terms"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (s *Supplier) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type PurchaseOrder struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	SupplierID   uuid.UUID      `json:"supplier_id" gorm:"type:uuid;not null;index"`
	PONumber     string         `json:"po_number" gorm:"uniqueIndex;not null"`
	Status       string         `json:"status" gorm:"index"`
	OrderDate    time.Time      `json:"order_date"`
	ExpectedDate *time.Time     `json:"expected_date"`
	Subtotal     float64        `json:"subtotal"`
	Discount     float64        `json:"discount"`
	Tax          float64        `json:"tax"`
	ShippingCost float64        `json:"shipping_cost"`
	GrandTotal   float64        `json:"grand_total"`
	CreatedBy    *uuid.UUID     `json:"created_by" gorm:"type:uuid"`
	ApprovedBy   *uuid.UUID     `json:"approved_by" gorm:"type:uuid"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Supplier *Supplier           `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
	Items    []PurchaseOrderItem `json:"items,omitempty" gorm:"foreignKey:PurchaseOrderID"`
	Receipts []GoodsReceipt      `json:"receipts,omitempty" gorm:"foreignKey:PurchaseOrderID"`
	Creator  *User               `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Approver *User               `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
}

func (p *PurchaseOrder) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PurchaseOrderItem struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PurchaseOrderID  uuid.UUID      `json:"purchase_order_id" gorm:"type:uuid;not null;index"`
	ItemID           uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	Quantity         float64        `json:"quantity"`
	ReceivedQuantity float64        `json:"received_quantity"`
	UnitCost         float64        `json:"unit_cost"`
	Discount         float64        `json:"discount"`
	Tax              float64        `json:"tax"`
	Total            float64        `json:"total"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	PurchaseOrder *PurchaseOrder `json:"purchase_order,omitempty" gorm:"foreignKey:PurchaseOrderID"`
	Item          *InventoryItem `json:"item,omitempty" gorm:"foreignKey:ItemID"`
}

func (p *PurchaseOrderItem) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type GoodsReceipt struct {
	ID                    uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PurchaseOrderID       uuid.UUID      `json:"purchase_order_id" gorm:"type:uuid;not null;index"`
	SupplierID            uuid.UUID      `json:"supplier_id" gorm:"type:uuid;not null;index"`
	ReceiptNumber         string         `json:"receipt_number" gorm:"uniqueIndex;not null"`
	ReceivedAt            time.Time      `json:"received_at"`
	ReceivedBy            *uuid.UUID     `json:"received_by" gorm:"type:uuid"`
	WarehouseLocationID   uuid.UUID      `json:"warehouse_location_id" gorm:"type:uuid;not null;index"`
	Status                string         `json:"status" gorm:"index"`
	SupplierInvoiceNumber string         `json:"supplier_invoice_number"`
	Notes                 string         `json:"notes"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	PurchaseOrder     *PurchaseOrder     `json:"purchase_order,omitempty" gorm:"foreignKey:PurchaseOrderID"`
	Supplier          *Supplier          `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
	ReceivedByUser    *User              `json:"received_by_user,omitempty" gorm:"foreignKey:ReceivedBy"`
	WarehouseLocation *InventoryLocation `json:"warehouse_location,omitempty" gorm:"foreignKey:WarehouseLocationID"`
	Items             []GoodsReceiptItem `json:"items,omitempty" gorm:"foreignKey:GoodsReceiptID"`
}

func (g *GoodsReceipt) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

type GoodsReceiptItem struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoodsReceiptID uuid.UUID      `json:"goods_receipt_id" gorm:"type:uuid;not null;index"`
	ItemID         uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	Quantity       float64        `json:"quantity"`
	UnitCost       float64        `json:"unit_cost"`
	Total          float64        `json:"total"`
	SerialNumbers  string         `json:"serial_numbers"`
	MACAddresses   string         `json:"mac_addresses"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	GoodsReceipt *GoodsReceipt  `json:"goods_receipt,omitempty" gorm:"foreignKey:GoodsReceiptID"`
	Item         *InventoryItem `json:"item,omitempty" gorm:"foreignKey:ItemID"`
}

func (g *GoodsReceiptItem) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

type StockTransfer struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TransferNumber string         `json:"transfer_number" gorm:"uniqueIndex;not null"`
	FromLocationID uuid.UUID      `json:"from_location_id" gorm:"type:uuid;not null;index"`
	ToLocationID   uuid.UUID      `json:"to_location_id" gorm:"type:uuid;not null;index"`
	Status         string         `json:"status" gorm:"index"`
	RequestedBy    *uuid.UUID     `json:"requested_by" gorm:"type:uuid"`
	SentBy         *uuid.UUID     `json:"sent_by" gorm:"type:uuid"`
	ReceivedBy     *uuid.UUID     `json:"received_by" gorm:"type:uuid"`
	SentAt         *time.Time     `json:"sent_at"`
	ReceivedAt     *time.Time     `json:"received_at"`
	Notes          string         `json:"notes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	FromLocation *InventoryLocation  `json:"from_location,omitempty" gorm:"foreignKey:FromLocationID"`
	ToLocation   *InventoryLocation  `json:"to_location,omitempty" gorm:"foreignKey:ToLocationID"`
	Items        []StockTransferItem `json:"items,omitempty" gorm:"foreignKey:StockTransferID"`
	Requester    *User               `json:"requester,omitempty" gorm:"foreignKey:RequestedBy"`
	Sender       *User               `json:"sender,omitempty" gorm:"foreignKey:SentBy"`
	Receiver     *User               `json:"receiver,omitempty" gorm:"foreignKey:ReceivedBy"`
}

func (s *StockTransfer) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type StockTransferItem struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	StockTransferID  uuid.UUID      `json:"stock_transfer_id" gorm:"type:uuid;not null;index"`
	ItemID           uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	SerialItemID     *uuid.UUID     `json:"serial_item_id" gorm:"type:uuid;index"`
	Quantity         float64        `json:"quantity"`
	ReceivedQuantity float64        `json:"received_quantity"`
	UnitCost         float64        `json:"unit_cost"`
	TotalCost        float64        `json:"total_cost"`
	Notes            string         `json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	StockTransfer *StockTransfer       `json:"stock_transfer,omitempty" gorm:"foreignKey:StockTransferID"`
	Item          *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	SerialItem    *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
}

func (s *StockTransferItem) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type MaterialUsage struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UsageNumber      string         `json:"usage_number" gorm:"uniqueIndex;not null"`
	UsageType        string         `json:"usage_type" gorm:"index"`
	CustomerID       *uuid.UUID     `json:"customer_id" gorm:"type:uuid;index"`
	SubscriptionID   *uuid.UUID     `json:"subscription_id" gorm:"type:uuid;index"`
	ServiceAccountID *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	ComplainID       *uuid.UUID     `json:"complain_id" gorm:"type:uuid;index"`
	CoverageID       *uuid.UUID     `json:"coverage_id" gorm:"type:uuid;index"`
	OdcID            *uuid.UUID     `json:"odc_id" gorm:"type:uuid;index"`
	OdpID            *uuid.UUID     `json:"odp_id" gorm:"type:uuid;index"`
	RouterID         *uuid.UUID     `json:"router_id" gorm:"type:uuid;index"`
	TechnicianUserID *uuid.UUID     `json:"technician_user_id" gorm:"type:uuid;index"`
	LocationID       uuid.UUID      `json:"location_id" gorm:"type:uuid;not null;index"`
	Status           string         `json:"status" gorm:"index"`
	UsedAt           time.Time      `json:"used_at"`
	Notes            string         `json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Customer       *Customer           `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	Subscription   *Subscription       `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
	ServiceAccount *ServiceAccount     `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
	Complain       *Complain           `json:"complain,omitempty" gorm:"foreignKey:ComplainID"`
	Coverage       *Coverage           `json:"coverage,omitempty" gorm:"foreignKey:CoverageID"`
	Odc            *Odc                `json:"odc,omitempty" gorm:"foreignKey:OdcID"`
	Odp            *Odp                `json:"odp,omitempty" gorm:"foreignKey:OdpID"`
	Router         *Router             `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	Technician     *User               `json:"technician,omitempty" gorm:"foreignKey:TechnicianUserID"`
	Location       *InventoryLocation  `json:"location,omitempty" gorm:"foreignKey:LocationID"`
	Items          []MaterialUsageItem `json:"items,omitempty" gorm:"foreignKey:MaterialUsageID"`
}

func (m *MaterialUsage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type MaterialUsageItem struct {
	ID                   uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	MaterialUsageID      uuid.UUID      `json:"material_usage_id" gorm:"type:uuid;not null;index"`
	ItemID               uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	SerialItemID         *uuid.UUID     `json:"serial_item_id" gorm:"type:uuid;index"`
	Quantity             float64        `json:"quantity"`
	UnitCost             float64        `json:"unit_cost"`
	TotalCost            float64        `json:"total_cost"`
	ChargeToCustomer     bool           `json:"charge_to_customer"`
	CustomerChargeAmount float64        `json:"customer_charge_amount"`
	Notes                string         `json:"notes"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	MaterialUsage *MaterialUsage       `json:"material_usage,omitempty" gorm:"foreignKey:MaterialUsageID"`
	Item          *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	SerialItem    *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
}

func (m *MaterialUsageItem) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type CustomerAsset struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	CustomerID       uuid.UUID      `json:"customer_id" gorm:"type:uuid;not null;index"`
	SubscriptionID   *uuid.UUID     `json:"subscription_id" gorm:"type:uuid;index"`
	ServiceAccountID *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	SerialItemID     *uuid.UUID     `json:"serial_item_id" gorm:"type:uuid;index"`
	ItemID           uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	InstalledAt      time.Time      `json:"installed_at"`
	ReturnedAt       *time.Time     `json:"returned_at"`
	Status           string         `json:"status" gorm:"index"`
	OwnershipType    string         `json:"ownership_type" gorm:"index"`
	DepositAmount    float64        `json:"deposit_amount"`
	Notes            string         `json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Customer       *Customer            `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	Subscription   *Subscription        `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
	ServiceAccount *ServiceAccount      `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
	SerialItem     *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
	Item           *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
}

func (c *CustomerAsset) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type NetworkAsset struct {
	ID                 uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	ItemID             uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	SerialItemID       *uuid.UUID     `json:"serial_item_id" gorm:"type:uuid;index"`
	AssetCode          string         `json:"asset_code" gorm:"uniqueIndex"`
	AssetType          string         `json:"asset_type" gorm:"index"`
	CoverageID         *uuid.UUID     `json:"coverage_id" gorm:"type:uuid;index"`
	OdcID              *uuid.UUID     `json:"odc_id" gorm:"type:uuid;index"`
	OdpID              *uuid.UUID     `json:"odp_id" gorm:"type:uuid;index"`
	RouterID           *uuid.UUID     `json:"router_id" gorm:"type:uuid;index"`
	InstalledAt        time.Time      `json:"installed_at"`
	Status             string         `json:"status" gorm:"index"`
	AcquisitionCost    float64        `json:"acquisition_cost"`
	DepreciationMethod string         `json:"depreciation_method"`
	UsefulLifeMonths   int            `json:"useful_life_months"`
	Notes              string         `json:"notes"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Item       *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	SerialItem *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
	Coverage   *Coverage            `json:"coverage,omitempty" gorm:"foreignKey:CoverageID"`
	Odc        *Odc                 `json:"odc,omitempty" gorm:"foreignKey:OdcID"`
	Odp        *Odp                 `json:"odp,omitempty" gorm:"foreignKey:OdpID"`
	Router     *Router              `json:"router,omitempty" gorm:"foreignKey:RouterID"`
}

func (n *NetworkAsset) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

type StockOpname struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	LocationID   uuid.UUID      `json:"location_id" gorm:"type:uuid;not null;index"`
	OpnameNumber string         `json:"opname_number" gorm:"uniqueIndex;not null"`
	Status       string         `json:"status" gorm:"index"`
	StartedAt    time.Time      `json:"started_at"`
	FinishedAt   *time.Time     `json:"finished_at"`
	CreatedBy    *uuid.UUID     `json:"created_by" gorm:"type:uuid"`
	SubmittedBy  *uuid.UUID     `json:"submitted_by" gorm:"type:uuid"`
	ApprovedBy   *uuid.UUID     `json:"approved_by" gorm:"type:uuid"`
	ApprovedAt   *time.Time     `json:"approved_at"`
	Notes        string         `json:"notes"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Location  *InventoryLocation `json:"location,omitempty" gorm:"foreignKey:LocationID"`
	Creator   *User              `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Submitter *User              `json:"submitter,omitempty" gorm:"foreignKey:SubmittedBy"`
	Approver  *User              `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
	Items     []StockOpnameItem  `json:"items,omitempty" gorm:"foreignKey:StockOpnameID"`
}

func (s *StockOpname) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type StockOpnameItem struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	StockOpnameID    uuid.UUID      `json:"stock_opname_id" gorm:"type:uuid;not null;index"`
	ItemID           uuid.UUID      `json:"item_id" gorm:"type:uuid;not null;index"`
	SerialItemID     *uuid.UUID     `json:"serial_item_id" gorm:"type:uuid;index"`
	SystemQuantity   float64        `json:"system_quantity"`
	CountedQuantity  float64        `json:"counted_quantity"`
	VarianceQuantity float64        `json:"variance_quantity"`
	VarianceCost     float64        `json:"variance_cost"`
	Notes            string         `json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	StockOpname *StockOpname         `json:"stock_opname,omitempty" gorm:"foreignKey:StockOpnameID"`
	Item        *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	SerialItem  *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
}

func (s *StockOpnameItem) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
