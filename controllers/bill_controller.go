package controllers

import (
	"strconv"
	"strings"
	"time"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type BillController struct {
	service services.BillService
}

func (c *BillController) RegisterRoutes(router fiber.Router) {
	api := router.Group("/api/bills")
	api.Get("/:public_id", c.GetByPublicID)

	user_api := router.Group("/user_api/bills")
	user_api.Get("/", middlewares.UserProtected(), c.GetAll)
	user_api.Get("/me", middlewares.UserProtected(), c.GetByUserID)
	user_api.Get("/public/:public_id", c.GetByPublicID)
	admin_api := router.Group("/admin_api/bills")
	adminOnly := middlewares.RequireRoles("root", "admin")
	billingOps := middlewares.RequireRoles("root", "admin", "loket", "petugas")
	admin_api.Get("/dashboard/stats", middlewares.UserProtected(), billingOps, c.GetDashboardStats)
	admin_api.Get("/dashboard/charts", middlewares.UserProtected(), billingOps, c.GetDashboardCharts)
	admin_api.Get("/recent/paid", middlewares.UserProtected(), billingOps, c.GetRecentPaidBills)
	admin_api.Get("/missing/current-month", middlewares.UserProtected(), billingOps, c.GetMissingCurrentMonthBills)
	admin_api.Get("/generate", middlewares.UserProtected(), billingOps, c.GenerateMonthlyBills)
	admin_api.Post("/generate", middlewares.UserProtected(), billingOps, c.GenerateMonthlyBills)
	admin_api.Post("/generate/dry-run", middlewares.UserProtected(), billingOps, c.PreviewMonthlyBills)
	admin_api.Post("/mark-overdue", middlewares.UserProtected(), billingOps, c.MarkOverdueBills)
	admin_api.Get("/send-reminders", middlewares.UserProtected(), billingOps, c.SendReminders)
	admin_api.Post("/create", middlewares.UserProtected(), billingOps, c.Create)
	admin_api.Get("/", middlewares.UserProtected(), billingOps, c.GetAll)
	admin_api.Get("/:id", middlewares.UserProtected(), billingOps, c.GetByID)
	admin_api.Put("/:id", middlewares.UserProtected(), billingOps, c.Update)
	admin_api.Delete("/generated/current-month/unpaid", middlewares.UserProtected(), adminOnly, c.DeleteCurrentMonthUnpaidBills)
	admin_api.Delete("/:id", middlewares.UserProtected(), adminOnly, c.Delete)
}

func NewBillController(service services.BillService) *BillController {
	return &BillController{service}
}

func (c *BillController) GetAll(ctx *fiber.Ctx) error {
	pageStr := ctx.Query("page", "1")
	limitStr := ctx.Query("limit", "10")
	search := ctx.Query("search", "")
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	status := strings.TrimSpace(ctx.Query("status", ""))
	startAt := strings.TrimSpace(ctx.Query("start_at", ""))
	endAt := strings.TrimSpace(ctx.Query("end_at", ""))
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	// Check if user is root, if yes, get all data without pagination
	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		if strings.ToLower(strings.TrimSpace(role)) == "root" {
			limit = 999999
			page = 1
		}
	}

	// Parse coverage_ids (both coverage_ids and coverage_ids[] formats) manually
	var coverageIDs []string

	// Get all query args
	ctx.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		// Check for both coverage_ids and coverage_ids[]
		if k == "coverage_ids" || k == "coverage_ids[]" {
			if v != "" {
				coverageIDs = append(coverageIDs, strings.TrimSpace(v))
			}
		}
	})

	// Fallback to single coverage_id if no array found
	if len(coverageIDs) == 0 {
		coverageID := strings.TrimSpace(ctx.Query("coverage_id", ""))
		if coverageID != "" {
			coverageIDs = []string{coverageID}
		}
	}

	// If endpoint is accessed with authenticated non-admin user, force filter to own user ID.
	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		userID, _ := userClaims["user_id"].(string)
		if strings.TrimSpace(role) != "" && strings.ToLower(role) != "admin" && strings.TrimSpace(userID) != "" {
			adminID = userID
		}
	}

	data, total, err := c.service.GetAll(page, limit, search, adminID, status, startAt, endAt, coverageIDs)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid admin_id") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid status") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid start_at format") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid end_at format") ||
			strings.Contains(strings.ToLower(err.Error()), "start_at must be before or equal end_at") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid coverage_id") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"meta": fiber.Map{
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": int((total + int64(limit) - 1) / int64(limit)),
			},
		},
		"data":    data,
		"message": "Success get all bills",
	})
}

func (c *BillController) GetMissingCurrentMonthBills(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	search := strings.TrimSpace(ctx.Query("search", ""))
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	coverageIDs := make([]string, 0)
	ctx.Context().QueryArgs().VisitAll(func(key, value []byte) {
		name := string(key)
		id := strings.TrimSpace(string(value))
		if (name == "coverage_ids" || name == "coverage_ids[]") && id != "" {
			coverageIDs = append(coverageIDs, id)
		}
	})
	if len(coverageIDs) == 0 {
		if coverageID := strings.TrimSpace(ctx.Query("coverage_id", "")); coverageID != "" {
			coverageIDs = []string{coverageID}
		}
	}

	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		userID, _ := userClaims["user_id"].(string)
		role = strings.ToLower(strings.TrimSpace(role))
		if role != "admin" && role != "root" && strings.TrimSpace(userID) != "" {
			adminID = userID
		}
	}

	result, total, err := c.service.GetMissingCurrentMonthBills(page, limit, search, adminID, coverageIDs)
	if err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "invalid admin_id") || strings.Contains(message, "invalid coverage_id") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"meta": fiber.Map{
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_items": total,
				"total_pages": int((total + int64(limit) - 1) / int64(limit)),
			},
		},
		"message": "Success get subscriptions without current month bill",
	})
}

func (c *BillController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Bill not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get bill"})
}
func (c *BillController) GetByPublicID(ctx *fiber.Ctx) error {
	publicID := ctx.Params("public_id")
	if publicID == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Public ID wajib diisi",
		})
	}

	data, err := c.service.GetByPublicID(publicID)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"success": false,
			"message": "Tagihan tidak ditemukan",
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    data,
		"message": "Success get bill by public ID",
	})
}
func (c *BillController) GetByUserID(ctx *fiber.Ctx) error {
	userClaims, ok := ctx.Locals("user").(jwt.MapClaims)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}
	userID, ok := userClaims["user_id"].(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}

	data, err := c.service.GetByUserID(userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    data,
		"message": "Success get bills by user ID",
	})
}

func (c *BillController) Create(ctx *fiber.Ctx) error {
	var input models.Bill
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	data, err := c.service.Create(input)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Bill created successfully"})
}

func (c *BillController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input models.Bill
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	data, err := c.service.Update(id, input)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Bill updated successfully"})
}

func (c *BillController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Bill deleted successfully"})
}

func (c *BillController) DeleteCurrentMonthUnpaidBills(ctx *fiber.Ctx) error {
	deleted, err := c.service.DeleteCurrentMonthUnpaidBills()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"deleted_count": deleted,
		},
		"message": "Current month non-paid generated bills deleted successfully",
	})
}

func (c *BillController) GenerateMonthlyBills(ctx *fiber.Ctx) error {
	var input struct {
		Period string `json:"period"`
	}
	if len(ctx.Body()) > 0 {
		if err := ctx.BodyParser(&input); err != nil {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
	}

	result, err := c.service.GenerateMonthlyBillsForPeriod(input.Period)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid period format") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Monthly bills generated successfully"})
}

func (c *BillController) PreviewMonthlyBills(ctx *fiber.Ctx) error {
	var input struct {
		Period string `json:"period"`
	}
	if len(ctx.Body()) > 0 {
		if err := ctx.BodyParser(&input); err != nil {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
	}

	result, err := c.service.PreviewMonthlyBills(input.Period)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid period format") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Monthly bill dry-run completed"})
}

func (c *BillController) MarkOverdueBills(ctx *fiber.Ctx) error {
	updated, err := c.service.MarkOverdueBills(time.Now(), 1000)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"updated_count": updated,
		},
		"message": "Overdue bills marked successfully",
	})
}

func (c *BillController) SendReminders(ctx *fiber.Ctx) error {
	result, err := c.service.SendReminders()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Reminders sent successfully"})
}

func (c *BillController) GetDashboardStats(ctx *fiber.Ctx) error {
	monthStr := strings.TrimSpace(ctx.Query("month", "0"))
	yearStr := strings.TrimSpace(ctx.Query("year", "0"))
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))

	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		userID, _ := userClaims["user_id"].(string)
		if strings.TrimSpace(role) != "" && strings.ToLower(role) != "admin" && strings.TrimSpace(userID) != "" {
			adminID = userID
		}
	}

	stats, err := c.service.GetDashboardStats(month, year, adminID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": stats, "message": "Dashboard stats retrieved successfully"})
}

func (c *BillController) GetDashboardCharts(ctx *fiber.Ctx) error {
	monthsStr := strings.TrimSpace(ctx.Query("months", "6"))
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	months, err := strconv.Atoi(monthsStr)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "invalid months"})
	}

	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		userID, _ := userClaims["user_id"].(string)
		if strings.TrimSpace(role) != "" && strings.ToLower(role) != "admin" && strings.TrimSpace(userID) != "" {
			adminID = userID
		}
	}

	charts, err := c.service.GetDashboardCharts(months, adminID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid admin_id") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": charts, "message": "Dashboard chart data retrieved successfully"})
}

func (c *BillController) GetRecentPaidBills(ctx *fiber.Ctx) error {
	limitStr := ctx.Query("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	bills, err := c.service.GetRecentPaidBills(limit)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": bills, "message": "Recent paid bills retrieved successfully"})
}
