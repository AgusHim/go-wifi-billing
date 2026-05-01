package controllers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/Agushim/go_wifi_billing/dto"
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type CustomerController struct {
	service services.CustomerService
}

func NewCustomerController(service services.CustomerService) *CustomerController {
	return &CustomerController{service}
}

func (c *CustomerController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/customers")
	r.Get("/export", middlewares.UserProtected(), c.ExportCSV)
	r.Post("/import", middlewares.UserProtected(), c.ImportCSV)
	r.Get("/", middlewares.UserProtected(), c.GetAll)
	r.Get("/by_user/:user_id", c.GetByUserID)
	r.Get("/:id", c.GetByID)
	r.Post("/", c.Create)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
}

func (c *CustomerController) Create(ctx *fiber.Ctx) error {
	var customer dto.CreateCustomerDTO
	if err := ctx.BodyParser(&customer); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	new_customer, err := c.service.Create(&customer)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": new_customer, "message": "Customer created"})
}

func (c *CustomerController) GetAll(ctx *fiber.Ctx) error {
	pageStr := ctx.Query("page", "1")
	limitStr := ctx.Query("limit", "10")
	search := ctx.Query("search", "")
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	coverageID := strings.TrimSpace(ctx.Query("coverage_id", ""))
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		userID, _ := userClaims["user_id"].(string)
		if strings.ToLower(strings.TrimSpace(role)) == "loket" && strings.TrimSpace(userID) != "" {
			adminID = userID
		}
	}

	customers, total, err := c.service.GetAll(page, limit, search, adminID, coverageID)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid admin_id") || strings.Contains(msg, "invalid coverage_id") {
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
		"data":    customers,
		"message": "Success get data",
	})
}

func (c *CustomerController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	customer, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Customer not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": customer, "message": "Success get data"})
}

func (c *CustomerController) GetByUserID(ctx *fiber.Ctx) error {
	userIDStr := ctx.Params("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user_id",
		})
	}

	customer, err := c.service.FindByUserID(userID)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"success": false,
			"message": "Customer not found",
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    customer,
		"message": "Success get data",
	})
}

func (c *CustomerController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	var customer dto.CreateCustomerDTO
	if err = ctx.BodyParser(&customer); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	updated, err := c.service.Update(id, &customer)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": updated, "message": "Customer updated"})
}

func (c *CustomerController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}

	if err := c.service.Delete(id); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Customer deleted"})
}

// ExportCSV exports all customers as a CSV file.
func (c *CustomerController) ExportCSV(ctx *fiber.Ctx) error {
	search := ctx.Query("search", "")
	adminID := strings.TrimSpace(ctx.Query("admin_id", ""))
	coverageID := strings.TrimSpace(ctx.Query("coverage_id", ""))

	if userClaims, ok := ctx.Locals("user").(jwt.MapClaims); ok {
		role, _ := userClaims["role"].(string)
		userID, _ := userClaims["user_id"].(string)
		if strings.ToLower(strings.TrimSpace(role)) == "loket" && strings.TrimSpace(userID) != "" {
			adminID = userID
		}
	}

	customers, _, err := c.service.GetAll(1, 100000, search, adminID, coverageID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{
		"service_number", "name", "email", "phone", "status", "address",
		"coverage", "odc", "odp", "port_odp",
		"latitude", "longitude", "mode", "id_ppoe", "profile_ppoe",
		"admin", "created_at",
	})

	for _, cu := range customers {
		name, email, phone := "", "", ""
		if cu.User != nil {
			name = cu.User.Name
			email = cu.User.Email
			phone = cu.User.Phone
		}
		coverageName := ""
		if cu.Coverage != nil {
			coverageName = cu.Coverage.Name
		}
		odcCode := ""
		if cu.Odc != nil {
			odcCode = cu.Odc.Code
		}
		odpCode := ""
		if cu.Odp != nil {
			odpCode = cu.Odp.Code
		}
		adminName := ""
		if cu.Admin != nil {
			adminName = cu.Admin.Name
		}

		_ = w.Write([]string{
			cu.ServiceNumber, name, email, phone, cu.Status, cu.Address,
			coverageName, odcCode, odpCode, cu.PortOdp,
			strconv.FormatFloat(cu.Latitude, 'f', 6, 64),
			strconv.FormatFloat(cu.Longitude, 'f', 6, 64),
			cu.Mode, cu.IDPPOE, cu.ProfilePPOE, adminName,
			cu.CreatedAt.Format("2006-01-02"),
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": "gagal membuat CSV"})
	}

	ctx.Set("Content-Type", "text/csv; charset=utf-8")
	ctx.Set("Content-Disposition", `attachment; filename="customers.csv"`)
	return ctx.Send(buf.Bytes())
}

// ImportCSV processes a CSV file upload and creates customers in bulk.
// CSV columns (0-indexed):
// 0:name 1:email 2:phone 3:password 4:service_number 5:card 6:id_card
// 7:status 8:address 9:coverage_id 10:odc_id 11:odp_id 12:port_odp
// 13:latitude 14:longitude 15:mode 16:id_ppoe 17:profile_ppoe
// 18:admin_id 19:is_send_wa 20:description(optional)
func (c *CustomerController) ImportCSV(ctx *fiber.Ctx) error {
	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "file wajib diupload"})
	}

	f, err := file.Open()
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "gagal membuka file"})
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "format CSV tidak valid"})
	}

	if len(records) < 2 {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "CSV tidak memiliki data (hanya header)"})
	}

	successCount := 0
	failCount := 0
	var errorList []string

	for i, row := range records[1:] {
		lineNum := i + 2

		if len(row) < 20 {
			failCount++
			errorList = append(errorList, fmt.Sprintf("baris %d: jumlah kolom tidak mencukupi (minimal 20)", lineNum))
			continue
		}

		// Validate required string fields
		required := []struct {
			name string
			idx  int
		}{
			{"name", 0}, {"email", 1}, {"phone", 2}, {"password", 3},
			{"service_number", 4}, {"card", 5}, {"id_card", 6},
			{"status", 7}, {"address", 8}, {"coverage_id", 9},
			{"admin_id", 18},
		}
		hasError := false
		for _, req := range required {
			if strings.TrimSpace(row[req.idx]) == "" {
				failCount++
				errorList = append(errorList, fmt.Sprintf("baris %d: kolom '%s' wajib diisi", lineNum, req.name))
				hasError = true
				break
			}
		}
		if hasError {
			continue
		}

		lat, _ := strconv.ParseFloat(strings.TrimSpace(row[13]), 64)
		lng, _ := strconv.ParseFloat(strings.TrimSpace(row[14]), 64)
		isSendWA := strings.ToLower(strings.TrimSpace(row[19])) == "true" || strings.TrimSpace(row[19]) == "1"

		description := ""
		if len(row) > 20 {
			description = strings.TrimSpace(row[20])
		}

		pStr := func(s string) *string { v := strings.TrimSpace(s); return &v }
		pStrOpt := func(s string) *string {
			v := strings.TrimSpace(s)
			if v == "" {
				return nil
			}
			return &v
		}

		input := &dto.CreateCustomerDTO{
			Name:          pStr(row[0]),
			Email:         pStr(row[1]),
			Phone:         pStr(row[2]),
			Password:      pStr(row[3]),
			ServiceNumber: pStr(row[4]),
			Card:          pStr(row[5]),
			IDCard:        pStr(row[6]),
			Status:        pStr(row[7]),
			Address:       pStr(row[8]),
			CoverageID:    pStr(row[9]),
			OdcID:         pStrOpt(row[10]),
			OdpID:         pStrOpt(row[11]),
			PortOdp:       pStrOpt(row[12]),
			Latitude:      &lat,
			Longitude:     &lng,
			Mode:          pStr(row[15]),
			IDPPOE:        pStrOpt(row[16]),
			ProfilePPOE:   pStrOpt(row[17]),
			AdminID:       pStr(row[18]),
			IsSendWA:      &isSendWA,
			Description:   &description,
		}

		_, createErr := c.service.Create(input)
		if createErr != nil {
			failCount++
			errorList = append(errorList, fmt.Sprintf("baris %d (%s): %s", lineNum, strings.TrimSpace(row[1]), createErr.Error()))
			continue
		}
		successCount++
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"success_count": successCount,
			"fail_count":    failCount,
			"errors":        errorList,
		},
		"message": fmt.Sprintf("Import selesai: %d berhasil, %d gagal", successCount, failCount),
	})
}
