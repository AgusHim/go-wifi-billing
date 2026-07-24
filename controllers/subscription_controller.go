package controllers

import (
	"log"
	"strconv"
	"strings"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type SubscriptionController struct {
	service         services.SubscriptionService
	customerService services.CustomerService
	renewalService  services.RenewalService
}

func NewSubscriptionController(service services.SubscriptionService, customerService services.CustomerService, renewalService services.RenewalService) *SubscriptionController {
	return &SubscriptionController{service, customerService, renewalService}
}

func (c *SubscriptionController) RegisterRoutes(router fiber.Router) {
	// User API
	userGroup := router.Group("/user_api/subscriptions", middlewares.UserProtected())
	userGroup.Get("/me", c.GetMySubscription)

	r := router.Group("/admin_api/subscriptions", middlewares.UserProtected())
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Get("/customer/:customer_id", c.GetByCustomerID)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
}

func (c *SubscriptionController) Create(ctx *fiber.Ctx) error {
	var subscription models.Subscription
	if err := ctx.BodyParser(&subscription); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.Create(&subscription); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	if c.renewalService != nil {
		if err := c.renewalService.SyncRecurringProfile(subscription.ID); err != nil {
			log.Printf("[renewal] failed to sync recurring profile after create for subscription %s: %v", subscription.ID, err)
		}
	}

	return ctx.JSON(fiber.Map{"success": true, "data": subscription, "message": "Subscription created"})
}

func (c *SubscriptionController) GetAll(ctx *fiber.Ctx) error {
	customerID := ctx.Query("customer_id", "")
	status := ctx.Query("status", "")
	pageStr := ctx.Query("page", "1")
	limitStr := ctx.Query("limit", "10")
	search := ctx.Query("search", "")
	customerDeleted := ctx.Query("customer_deleted", "")
	endDate := ctx.Query("end_date", "")
	var coverageIDs []string
	ctx.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		v := strings.TrimSpace(string(value))
		if (k == "coverage_ids" || k == "coverage_ids[]") && v != "" {
			coverageIDs = append(coverageIDs, v)
		}
	})
	if len(coverageIDs) == 0 {
		if coverageID := strings.TrimSpace(ctx.Query("coverage_id", "")); coverageID != "" {
			coverageIDs = []string{coverageID}
		}
	}
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	var endDateFilter *string
	if endDate != "" {
		endDateFilter = &endDate
	}

	subscriptions, total, err := c.service.GetAll(page, limit, search, &customerID, &status, &customerDeleted, endDateFilter, coverageIDs)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid coverage_id") {
			return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true, "meta": fiber.Map{
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": int((total + int64(limit) - 1) / int64(limit)),
			},
		},
		"data":    subscriptions,
		"message": "Success get data",
	})
}

func (c *SubscriptionController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	subscription, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Subscription not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": subscription, "message": "Success get data"})
}
func (c *SubscriptionController) GetByCustomerID(ctx *fiber.Ctx) error {
	customerID := ctx.Params("customer_id")
	if customerID == "" {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Customer ID is required"})
	}

	subscriptions, err := c.service.FindByCustomerID(customerID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    subscriptions,
		"message": "Success get subscriptions by customer",
	})
}

func (c *SubscriptionController) GetMySubscription(ctx *fiber.Ctx) error {
	userClaims, ok := ctx.Locals("user").(jwt.MapClaims)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "Unauthorized"})
	}
	userIDStr, ok := userClaims["user_id"].(string)
	if !ok || userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "Invalid token"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid User ID"})
	}

	customer, err := c.customerService.FindByUserID(userID)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Customer profile not found"})
	}

	subscriptions, err := c.service.FindByCustomerID(customer.ID.String())
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    subscriptions,
		"message": "Success get my subscriptions",
	})
}

func (c *SubscriptionController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	var subscription models.Subscription
	if err = ctx.BodyParser(&subscription); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	n_subs, err := c.service.Update(id, &subscription)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	if c.renewalService != nil {
		if err := c.renewalService.SyncRecurringProfile(n_subs.ID); err != nil {
			log.Printf("[renewal] failed to sync recurring profile after update for subscription %s: %v", n_subs.ID, err)
		}
	}

	return ctx.JSON(fiber.Map{"success": true, "data": n_subs, "message": "Subscription updated"})
}

func (c *SubscriptionController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Subscription deleted"})
}
