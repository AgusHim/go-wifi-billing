package controllers

import (
	"strings"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type InventoryController struct {
	service services.InventoryService
}

func NewInventoryController(service services.InventoryService) *InventoryController {
	return &InventoryController{service: service}
}

func (c *InventoryController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/inventory")
	r.Get("/items", c.GetItems)
	r.Post("/items", c.CreateItem)
	r.Get("/items/:id", c.GetItemByID)
	r.Put("/items/:id", c.UpdateItem)
	r.Delete("/items/:id", c.DeleteItem)

	r.Get("/locations", c.GetLocations)
	r.Post("/locations", c.CreateLocation)
	r.Get("/locations/:id", c.GetLocationByID)
	r.Put("/locations/:id", c.UpdateLocation)
	r.Delete("/locations/:id", c.DeleteLocation)

	r.Get("/stocks", c.GetStocks)
	r.Get("/movements", c.GetMovements)
	r.Get("/serial-items", c.GetSerialItems)

	r.Get("/suppliers", c.GetSuppliers)
	r.Post("/suppliers", c.CreateSupplier)
	r.Get("/suppliers/:id", c.GetSupplierByID)
	r.Put("/suppliers/:id", c.UpdateSupplier)
	r.Delete("/suppliers/:id", c.DeleteSupplier)

	r.Get("/purchase-orders", c.GetPurchaseOrders)
	r.Post("/purchase-orders", c.CreatePurchaseOrder)
	r.Get("/purchase-orders/:id", c.GetPurchaseOrderByID)
	r.Put("/purchase-orders/:id", c.UpdatePurchaseOrder)

	r.Get("/goods-receipts", c.GetGoodsReceipts)
	r.Post("/goods-receipts", c.CreateGoodsReceipt)
	r.Get("/goods-receipts/:id", c.GetGoodsReceiptByID)

	r.Get("/stock-transfers", c.GetStockTransfers)
	r.Post("/stock-transfers", c.CreateStockTransfer)
	r.Get("/stock-transfers/:id", c.GetStockTransferByID)
	r.Post("/stock-transfers/:id/receive", c.ReceiveStockTransfer)

	r.Get("/material-usages", c.GetMaterialUsages)
	r.Post("/material-usages", c.CreateMaterialUsage)
	r.Get("/material-usages/:id", c.GetMaterialUsageByID)

	r.Get("/customer-assets", c.GetCustomerAssets)
	r.Post("/customer-assets/:id/return", c.ReturnCustomerAsset)
	r.Post("/customer-assets/:id/replace", c.ReplaceCustomerAsset)
	r.Post("/customer-assets/:id/status", c.UpdateCustomerAssetStatus)

	r.Get("/network-assets", c.GetNetworkAssets)
	r.Post("/network-assets/:id/status", c.UpdateNetworkAssetStatus)

	r.Get("/stock-opnames", c.GetStockOpnames)
	r.Post("/stock-opnames", c.CreateStockOpname)
	r.Get("/stock-opnames/:id", c.GetStockOpnameByID)
	r.Post("/stock-opnames/:id/submit", c.SubmitStockOpname)
	r.Post("/stock-opnames/:id/approve", c.ApproveStockOpname)

	r.Get("/accounting/accounts", c.GetChartOfAccounts)
	r.Get("/accounting/journals", c.GetAccountingJournals)
	r.Get("/accounting/valuation", c.GetInventoryValuation)
	r.Get("/accounting/period-locks", c.GetPeriodLocks)
	r.Post("/accounting/period-locks", c.UpsertPeriodLock)
}

func (c *InventoryController) CreateItem(ctx *fiber.Ctx) error {
	var input models.InventoryItem
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateItem(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Inventory item created"})
}

func (c *InventoryController) GetItems(ctx *fiber.Ctx) error {
	active, err := optionalBoolQuery(ctx.Query("active", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	items, err := c.service.GetItems(ctx.Query("search", ""), active)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Inventory items retrieved"})
}

func (c *InventoryController) GetItemByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetItemByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Inventory item retrieved"})
}

func (c *InventoryController) UpdateItem(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input models.InventoryItem
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpdateItem(id, &input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Inventory item updated"})
}

func (c *InventoryController) DeleteItem(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.DeleteItem(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Inventory item deleted"})
}

func (c *InventoryController) CreateLocation(ctx *fiber.Ctx) error {
	var input models.InventoryLocation
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateLocation(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Inventory location created"})
}

func (c *InventoryController) GetLocations(ctx *fiber.Ctx) error {
	active, err := optionalBoolQuery(ctx.Query("active", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	items, err := c.service.GetLocations(ctx.Query("type", ""), active)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Inventory locations retrieved"})
}

func (c *InventoryController) GetLocationByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetLocationByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Inventory location retrieved"})
}

func (c *InventoryController) UpdateLocation(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input models.InventoryLocation
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpdateLocation(id, &input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Inventory location updated"})
}

func (c *InventoryController) DeleteLocation(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.DeleteLocation(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Inventory location deleted"})
}

func (c *InventoryController) GetStocks(ctx *fiber.Ctx) error {
	itemID, err := optionalUUIDQuery(ctx.Query("item_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid item_id"})
	}
	locationID, err := optionalUUIDQuery(ctx.Query("location_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid location_id"})
	}
	items, err := c.service.GetStocks(itemID, locationID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Inventory stocks retrieved"})
}

func (c *InventoryController) GetMovements(ctx *fiber.Ctx) error {
	itemID, err := optionalUUIDQuery(ctx.Query("item_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid item_id"})
	}
	locationID, err := optionalUUIDQuery(ctx.Query("location_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid location_id"})
	}
	items, err := c.service.GetMovements(itemID, locationID, ctx.Query("movement_type", ""), ctx.QueryInt("limit", 200))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Inventory movements retrieved"})
}

func (c *InventoryController) GetSerialItems(ctx *fiber.Ctx) error {
	itemID, err := optionalUUIDQuery(ctx.Query("item_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid item_id"})
	}
	items, err := c.service.GetSerialItems(itemID, ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Inventory serial items retrieved"})
}

func (c *InventoryController) CreateSupplier(ctx *fiber.Ctx) error {
	var input models.Supplier
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateSupplier(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Supplier created"})
}

func (c *InventoryController) GetSuppliers(ctx *fiber.Ctx) error {
	active, err := optionalBoolQuery(ctx.Query("active", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	items, err := c.service.GetSuppliers(ctx.Query("search", ""), active)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Suppliers retrieved"})
}

func (c *InventoryController) GetSupplierByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetSupplierByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Supplier retrieved"})
}

func (c *InventoryController) UpdateSupplier(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input models.Supplier
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpdateSupplier(id, &input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Supplier updated"})
}

func (c *InventoryController) DeleteSupplier(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.DeleteSupplier(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Supplier deleted"})
}

func (c *InventoryController) CreatePurchaseOrder(ctx *fiber.Ctx) error {
	var input models.PurchaseOrder
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreatePurchaseOrder(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Purchase order created"})
}

func (c *InventoryController) GetPurchaseOrders(ctx *fiber.Ctx) error {
	items, err := c.service.GetPurchaseOrders(ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Purchase orders retrieved"})
}

func (c *InventoryController) GetPurchaseOrderByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetPurchaseOrderByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Purchase order retrieved"})
}

func (c *InventoryController) UpdatePurchaseOrder(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input models.PurchaseOrder
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpdatePurchaseOrder(id, &input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Purchase order updated"})
}

func (c *InventoryController) CreateGoodsReceipt(ctx *fiber.Ctx) error {
	var input models.GoodsReceipt
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.ReceiveGoods(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Goods receipt created"})
}

func (c *InventoryController) GetGoodsReceipts(ctx *fiber.Ctx) error {
	items, err := c.service.GetGoodsReceipts(ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Goods receipts retrieved"})
}

func (c *InventoryController) GetGoodsReceiptByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetGoodsReceiptByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Goods receipt retrieved"})
}

func (c *InventoryController) CreateStockTransfer(ctx *fiber.Ctx) error {
	var input models.StockTransfer
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateStockTransfer(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Stock transfer created"})
}

func (c *InventoryController) GetStockTransfers(ctx *fiber.Ctx) error {
	items, err := c.service.GetStockTransfers(ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Stock transfers retrieved"})
}

func (c *InventoryController) GetStockTransferByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetStockTransferByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Stock transfer retrieved"})
}

func (c *InventoryController) ReceiveStockTransfer(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		ReceivedBy *uuid.UUID `json:"received_by"`
	}
	_ = ctx.BodyParser(&input)
	item, err := c.service.ReceiveStockTransfer(id, input.ReceivedBy)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Stock transfer received"})
}

func (c *InventoryController) CreateMaterialUsage(ctx *fiber.Ctx) error {
	var input models.MaterialUsage
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateMaterialUsage(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Material usage created"})
}

func (c *InventoryController) GetMaterialUsages(ctx *fiber.Ctx) error {
	referenceID, err := optionalUUIDQuery(ctx.Query("reference_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid reference_id"})
	}
	items, err := c.service.GetMaterialUsages(ctx.Query("reference_type", ""), referenceID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Material usages retrieved"})
}

func (c *InventoryController) GetMaterialUsageByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetMaterialUsageByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Material usage retrieved"})
}

func (c *InventoryController) GetCustomerAssets(ctx *fiber.Ctx) error {
	customerID, err := optionalUUIDQuery(ctx.Query("customer_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid customer_id"})
	}
	items, err := c.service.GetCustomerAssets(customerID, ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Customer assets retrieved"})
}

func (c *InventoryController) ReturnCustomerAsset(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		ReturnLocationID uuid.UUID `json:"return_location_id"`
		Notes            string    `json:"notes"`
	}
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.ReturnCustomerAsset(id, input.ReturnLocationID, input.Notes)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Customer asset returned"})
}

func (c *InventoryController) ReplaceCustomerAsset(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		NewSerialItemID uuid.UUID `json:"new_serial_item_id"`
		LocationID      uuid.UUID `json:"location_id"`
		Notes           string    `json:"notes"`
	}
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.ReplaceCustomerAsset(id, input.NewSerialItemID, input.LocationID, input.Notes)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Customer asset replaced"})
}

func (c *InventoryController) UpdateCustomerAssetStatus(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpdateCustomerAssetStatus(id, input.Status, input.Notes)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Customer asset status updated"})
}

func (c *InventoryController) GetNetworkAssets(ctx *fiber.Ctx) error {
	referenceID, err := optionalUUIDQuery(ctx.Query("reference_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid reference_id"})
	}
	items, err := c.service.GetNetworkAssets(ctx.Query("reference_type", ""), referenceID, ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Network assets retrieved"})
}

func (c *InventoryController) UpdateNetworkAssetStatus(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpdateNetworkAssetStatus(id, input.Status, input.Notes)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Network asset status updated"})
}

func (c *InventoryController) CreateStockOpname(ctx *fiber.Ctx) error {
	var input models.StockOpname
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.CreateStockOpname(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item, "message": "Stock opname created"})
}

func (c *InventoryController) GetStockOpnames(ctx *fiber.Ctx) error {
	locationID, err := optionalUUIDQuery(ctx.Query("location_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid location_id"})
	}
	items, err := c.service.GetStockOpnames(locationID, ctx.Query("status", ""))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Stock opnames retrieved"})
}

func (c *InventoryController) GetStockOpnameByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetStockOpnameByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Stock opname retrieved"})
}

func (c *InventoryController) SubmitStockOpname(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		SubmittedBy *uuid.UUID `json:"submitted_by"`
	}
	_ = ctx.BodyParser(&input)
	item, err := c.service.SubmitStockOpname(id, input.SubmittedBy)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Stock opname submitted"})
}

func (c *InventoryController) ApproveStockOpname(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input struct {
		ApprovedBy *uuid.UUID `json:"approved_by"`
	}
	_ = ctx.BodyParser(&input)
	item, err := c.service.ApproveStockOpname(id, input.ApprovedBy)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Stock opname approved"})
}

func (c *InventoryController) GetChartOfAccounts(ctx *fiber.Ctx) error {
	items, err := c.service.GetChartOfAccounts()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Chart of accounts retrieved"})
}

func (c *InventoryController) GetAccountingJournals(ctx *fiber.Ctx) error {
	sourceID, err := optionalUUIDQuery(ctx.Query("source_id", ""))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid source_id"})
	}
	items, err := c.service.GetAccountingJournals(ctx.Query("source_type", ""), sourceID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Accounting journals retrieved"})
}

func (c *InventoryController) GetInventoryValuation(ctx *fiber.Ctx) error {
	items, err := c.service.GetInventoryValuation()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Inventory valuation retrieved"})
}

func (c *InventoryController) UpsertPeriodLock(ctx *fiber.Ctx) error {
	var input models.AccountingPeriodLock
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	item, err := c.service.UpsertPeriodLock(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Period lock saved"})
}

func (c *InventoryController) GetPeriodLocks(ctx *fiber.Ctx) error {
	items, err := c.service.GetPeriodLocks()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": items, "message": "Period locks retrieved"})
}

func optionalUUIDQuery(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func optionalBoolQuery(value string) (*bool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "true" || normalized == "1" || normalized == "yes" {
		result := true
		return &result, nil
	}
	if normalized == "false" || normalized == "0" || normalized == "no" {
		result := false
		return &result, nil
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, "invalid boolean query")
}
