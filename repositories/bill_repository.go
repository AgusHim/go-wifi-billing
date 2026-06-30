package repositories

import (
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillRepository interface {
	FindAll() ([]models.Bill, error)
	FindAllPaginated(page, limit int, search string, adminID *uuid.UUID, status string, startDate *time.Time, endDate *time.Time, coverageIDs []uuid.UUID) ([]models.Bill, int64, error)
	FindByID(id string) (models.Bill, error)
	FindByPublicID(publicID string) (*models.Bill, error)
	FindByUserID(userID string) ([]models.Bill, error)
	Create(bill *models.Bill) error
	Update(bill *models.Bill) error
	Delete(id string) error
	DeleteUnpaidByBillDateRange(startDate time.Time, endDate time.Time) (int64, error)
	FindBillByCustomerAndMonth(customerID string, month int, year int) (*models.Bill, error)
	FindBillBySubscriptionAndMonth(subscriptionID uuid.UUID, month int, year int) (*models.Bill, error)
	FindUnpaidBills() ([]models.Bill, error)
	GetDashboardStats(month, year int, adminID *uuid.UUID) (map[string]int64, error)
	GetRecentPaidBills(limit int) ([]models.Bill, error)
	GetDashboardChartRows(fromDate time.Time, adminID *uuid.UUID) ([]models.Bill, error)
}

type billRepository struct {
	db *gorm.DB
}

func NewBillRepository(db *gorm.DB) BillRepository {
	return &billRepository{db}
}

func (r *billRepository) FindAll() ([]models.Bill, error) {
	var bills []models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		Find(&bills).Error
	return bills, err
}

func (r *billRepository) FindAllPaginated(
	page, limit int,
	search string,
	adminID *uuid.UUID,
	status string,
	startDate *time.Time,
	endDate *time.Time,
	coverageIDs []uuid.UUID,
) ([]models.Bill, int64, error) {
	var bills []models.Bill
	var total int64

	query := r.db.Model(&models.Bill{}).
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package")

	// Join customers table when needed for search, coverage, or admin filter
	if search != "" || len(coverageIDs) > 0 || adminID != nil {
		query = query.Joins("JOIN customers ON customers.id = bills.customer_id")
	}
	if search != "" {
		query = query.Joins("JOIN users ON users.id = customers.user_id").
			Where("LOWER(users.name) LIKE LOWER(?)", "%"+search+"%")
	}
	if len(coverageIDs) > 0 {
		query = query.Where("customers.coverage_id IN ?", coverageIDs)
	}
	if adminID != nil {
		query = query.Where("customers.admin_id = ?", *adminID)
	}
	if status != "" {
		if strings.ToLower(status) == "overdue" {
			// For overdue status, include both:
			// 1. Bills explicitly marked as overdue
			// 2. Unpaid bills that are past due date
			now := time.Now()
			query = query.Where("(LOWER(bills.status) = ? OR (LOWER(bills.status) = ? AND bills.due_date < ?))", "overdue", "unpaid", now)
		} else {
			query = query.Where("LOWER(bills.status) = LOWER(?)", status)
		}
	}
	if startDate != nil {
		query = query.Where("bills.due_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("bills.due_date < ?", *endDate)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Find(&bills).Error

	return bills, total, err
}

func (r *billRepository) FindByID(id string) (models.Bill, error) {
	var bill models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		First(&bill, "id = ?", id).Error
	return bill, err
}
func (r *billRepository) FindByUserID(userID string) ([]models.Bill, error) {
	var bills []models.Bill
	err := r.db.
		Joins("JOIN customers ON customers.id = bills.customer_id").
		Joins("JOIN users ON users.id = customers.user_id").
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		Where("users.id = ?", userID).
		Find(&bills).Error

	return bills, err
}

func (r *billRepository) FindByPublicID(publicID string) (*models.Bill, error) {
	var bill models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		First(&bill, "LOWER(public_id) = LOWER(?)", publicID).Error

	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *billRepository) Create(bill *models.Bill) error {
	return r.db.Create(bill).Error
}

func (r *billRepository) Update(bill *models.Bill) error {
	return r.db.Omit("Subscription", "Customer", "Customer.User").Save(bill).Error
}

func (r *billRepository) Delete(id string) error {
	return r.db.Delete(&models.Bill{}, "id = ?", id).Error
}

func (r *billRepository) DeleteUnpaidByBillDateRange(startDate time.Time, endDate time.Time) (int64, error) {
	result := r.db.
		Where("LOWER(status) = ? AND bill_date >= ? AND bill_date < ?", "unpaid", startDate, endDate).
		Delete(&models.Bill{})

	return result.RowsAffected, result.Error
}

func (r *billRepository) FindBillByCustomerAndMonth(customerID string, month int, year int) (*models.Bill, error) {
	var bill models.Bill
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	err := r.db.
		Where("customer_id = ? AND bill_date >= ? AND bill_date < ?",
			customerID, startOfMonth, endOfMonth).
		First(&bill).Error

	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *billRepository) FindBillBySubscriptionAndMonth(subscriptionID uuid.UUID, month int, year int) (*models.Bill, error) {
	var bill models.Bill
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	err := r.db.
		Where("subscription_id = ? AND bill_date >= ? AND bill_date < ?",
			subscriptionID, startOfMonth, endOfMonth).
		First(&bill).Error

	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *billRepository) FindUnpaidBills() ([]models.Bill, error) {
	var bills []models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		Where("status = ?", "unpaid").
		Find(&bills).Error
	return bills, err
}

func (r *billRepository) GetDashboardStats(month, year int, adminID *uuid.UUID) (map[string]int64, error) {
	stats := make(map[string]int64)

	now := time.Now()
	if year <= 0 {
		year = now.Year()
	}
	if month <= 0 || month > 12 {
		month = int(now.Month())
	}

	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	baseBillQuery := func() *gorm.DB {
		q := r.db.Model(&models.Bill{}).
			Joins("JOIN subscriptions ON subscriptions.id = bills.subscription_id").
			Where("subscriptions.deleted_at IS NULL").
			Where("LOWER(subscriptions.status) = ?", "active").
			Where("bills.bill_date >= ? AND bills.bill_date < ?", startOfMonth, endOfMonth)
		if adminID != nil {
			q = q.Joins("JOIN customers ON customers.id = bills.customer_id").Where("customers.admin_id = ?", *adminID)
		}
		return q
	}

	// Helper to sum amount and count
	getStats := func(condition string, args ...interface{}) (int64, int64, error) {
		type Result struct {
			Count  int64
			Amount int64
		}
		var res Result
		err := baseBillQuery().Where(condition, args...).
			Select("COUNT(bills.id) as count, COALESCE(SUM(bills.amount), 0) as amount").
			Scan(&res).Error
		return res.Count, res.Amount, err
	}

	// Paid
	paidCount, paidAmount, err := getStats("LOWER(bills.status) = ?", "paid")
	if err != nil {
		return nil, err
	}
	stats["paid_bills"] = paidCount
	stats["amount_paid"] = paidAmount

	// Unpaid
	// Using end of selected month to check if it's past due? Or time.Now()?
	// If looking at a past month, due_date is compared to time.Now() generally to determine if it's currently overdue.
	unpaidCount, unpaidAmount, err := getStats("LOWER(bills.status) = ? AND bills.due_date >= ?", "unpaid", now)
	if err != nil {
		return nil, err
	}
	stats["unpaid_bills"] = unpaidCount
	stats["amount_unpaid"] = unpaidAmount

	// Overdue
	overdueCount, overdueAmount, err := getStats("(LOWER(bills.status) = ? OR (LOWER(bills.status) = ? AND bills.due_date < ?))", "overdue", "unpaid", now)
	if err != nil {
		return nil, err
	}
	stats["overdue_bills"] = overdueCount
	stats["amount_overdue"] = overdueAmount

	// Total customers
	custQuery := r.db.Table("customers").Where("customers.deleted_at IS NULL")
	if adminID != nil {
		custQuery = custQuery.Where("customers.admin_id = ?", *adminID)
	}
	var customerCount int64
	if err := custQuery.Count(&customerCount).Error; err != nil {
		return nil, err
	}
	stats["total_customers"] = customerCount

	// Total admins
	var adminCount int64
	if err := r.db.Table("users").Where("role = ?", "admin").Count(&adminCount).Error; err != nil {
		return nil, err
	}
	stats["total_admins"] = adminCount

	// Total active subscriptions for the given month
	// Hanya hitung langganan yang start_date-nya jatuh di bulan yang dipilih
	subQuery := r.db.Table("subscriptions").
		Where("subscriptions.deleted_at IS NULL AND LOWER(subscriptions.status) = ?", "active").
		Where("subscriptions.start_date >= ? AND subscriptions.start_date < ?", startOfMonth, endOfMonth)
	if adminID != nil {
		subQuery = subQuery.Joins("JOIN customers ON customers.id = subscriptions.customer_id").Where("customers.admin_id = ?", *adminID)
	}
	var subscriptionCount int64
	if err := subQuery.Count(&subscriptionCount).Error; err != nil {
		return nil, err
	}
	stats["total_subscriptions"] = subscriptionCount

	return stats, nil
}

func (r *billRepository) GetRecentPaidBills(limit int) ([]models.Bill, error) {
	var bills []models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		Where("status = ?", "paid").
		Order("updated_at DESC").
		Limit(limit).
		Find(&bills).Error
	return bills, err
}

func (r *billRepository) GetDashboardChartRows(fromDate time.Time, adminID *uuid.UUID) ([]models.Bill, error) {
	var bills []models.Bill

	query := r.db.Model(&models.Bill{}).
		Select("bills.subscription_id", "bills.bill_date", "bills.due_date", "bills.status", "bills.amount").
		Joins("JOIN subscriptions ON subscriptions.id = bills.subscription_id").
		Where("subscriptions.deleted_at IS NULL").
		Where("LOWER(subscriptions.status) = ?", "active")

	if adminID != nil {
		query = query.Where("bills.admin_id = ?", *adminID)
	}

	err := query.
		Where("bills.bill_date >= ?", fromDate).
		Order("bills.bill_date ASC").
		Find(&bills).Error
	if err != nil {
		return nil, err
	}

	return bills, nil
}
