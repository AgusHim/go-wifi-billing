package repositories

import (
	"fmt"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FinanceSummaryResult struct {
	TotalIncome  int64 `json:"total_income"`
	TotalExpense int64 `json:"total_expense"`
	NetProfit    int64 `json:"net_profit"`
}

type FinanceMonthlyRow struct {
	Month   string `json:"month"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
	Net     int64  `json:"net"`
}

type IncomeByMethodRow struct {
	Method string `json:"method" gorm:"column:method"`
	Count  int64  `json:"count"  gorm:"column:count"`
	Total  int64  `json:"total"  gorm:"column:total"`
}

type IncomeByPackageRow struct {
	PackageName string `json:"package_name" gorm:"column:package_name"`
	Count       int64  `json:"count"        gorm:"column:count"`
	Total       int64  `json:"total"        gorm:"column:total"`
}

type FinanceRepository interface {
	GetSummary(startDate *time.Time, endDate *time.Time, adminID *uuid.UUID) (FinanceSummaryResult, error)
	GetMonthly(months int) ([]FinanceMonthlyRow, error)
	GetIncomeByMethod(startDate *time.Time, endDate *time.Time, adminID *uuid.UUID) ([]IncomeByMethodRow, error)
	GetIncomeByPackage(startDate *time.Time, endDate *time.Time) ([]IncomeByPackageRow, error)
}

type financeRepository struct {
	db *gorm.DB
}

func NewFinanceRepository(db *gorm.DB) FinanceRepository {
	return &financeRepository{db}
}

func (r *financeRepository) GetSummary(startDate *time.Time, endDate *time.Time, adminID *uuid.UUID) (FinanceSummaryResult, error) {
	var totalIncome int64
	var totalExpense int64

	incomeQuery := r.db.Model(&models.Payment{}).Where("status = ?", "confirmed")
	if adminID != nil {
		incomeQuery = incomeQuery.Where("admin_id = ?", *adminID)
	}
	if startDate != nil {
		incomeQuery = incomeQuery.Where("payment_date >= ?", *startDate)
	}
	if endDate != nil {
		incomeQuery = incomeQuery.Where("payment_date < ?", *endDate)
	}
	if err := incomeQuery.Select("COALESCE(SUM(amount), 0)").Scan(&totalIncome).Error; err != nil {
		return FinanceSummaryResult{}, err
	}

	expenseQuery := r.db.Model(&models.Expense{})
	if startDate != nil {
		expenseQuery = expenseQuery.Where("expense_date >= ?", *startDate)
	}
	if endDate != nil {
		expenseQuery = expenseQuery.Where("expense_date < ?", *endDate)
	}
	if err := expenseQuery.Select("COALESCE(SUM(amount), 0)").Scan(&totalExpense).Error; err != nil {
		return FinanceSummaryResult{}, err
	}

	return FinanceSummaryResult{
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		NetProfit:    totalIncome - totalExpense,
	}, nil
}

func (r *financeRepository) GetMonthly(months int) ([]FinanceMonthlyRow, error) {
	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month()-time.Month(months-1), 1, 0, 0, 0, 0, time.UTC)

	type monthAmount struct {
		Month  string `gorm:"column:month"`
		Amount int64  `gorm:"column:amount"`
	}

	var incomeRows []monthAmount
	r.db.Model(&models.Payment{}).
		Select("TO_CHAR(DATE_TRUNC('month', payment_date), 'YYYY-MM') as month, COALESCE(SUM(amount), 0) as amount").
		Where("status = ? AND payment_date >= ?", "confirmed", startDate).
		Group("DATE_TRUNC('month', payment_date)").
		Scan(&incomeRows)

	var expenseRows []monthAmount
	r.db.Model(&models.Expense{}).
		Select("TO_CHAR(DATE_TRUNC('month', expense_date), 'YYYY-MM') as month, COALESCE(SUM(amount), 0) as amount").
		Where("expense_date >= ?", startDate).
		Group("DATE_TRUNC('month', expense_date)").
		Scan(&expenseRows)

	incomeMap := make(map[string]int64)
	for _, row := range incomeRows {
		incomeMap[row.Month] = row.Amount
	}
	expenseMap := make(map[string]int64)
	for _, row := range expenseRows {
		expenseMap[row.Month] = row.Amount
	}

	result := make([]FinanceMonthlyRow, 0, months)
	for i := months - 1; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		monthKey := fmt.Sprintf("%d-%02d", t.Year(), int(t.Month()))
		income := incomeMap[monthKey]
		expense := expenseMap[monthKey]
		result = append(result, FinanceMonthlyRow{
			Month:   monthKey,
			Income:  income,
			Expense: expense,
			Net:     income - expense,
		})
	}
	return result, nil
}

func (r *financeRepository) GetIncomeByMethod(startDate *time.Time, endDate *time.Time, adminID *uuid.UUID) ([]IncomeByMethodRow, error) {
	query := r.db.Model(&models.Payment{}).
		Select("method, COUNT(id) as count, COALESCE(SUM(amount), 0) as total").
		Where("status = ?", "confirmed")

	if adminID != nil {
		query = query.Where("admin_id = ?", *adminID)
	}
	if startDate != nil {
		query = query.Where("payment_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("payment_date < ?", *endDate)
	}

	var rows []IncomeByMethodRow
	err := query.Group("method").Order("total DESC").Scan(&rows).Error
	return rows, err
}

func (r *financeRepository) GetIncomeByPackage(startDate *time.Time, endDate *time.Time) ([]IncomeByPackageRow, error) {
	query := r.db.Raw(`
		SELECT
			pkg.name AS package_name,
			COUNT(p.id) AS count,
			COALESCE(SUM(p.amount), 0) AS total
		FROM payments p
		JOIN bills b   ON p.bill_id         = b.id  AND b.deleted_at  IS NULL
		JOIN subscriptions s ON b.subscription_id = s.id  AND s.deleted_at IS NULL
		JOIN packages pkg    ON s.package_id      = pkg.id AND pkg.deleted_at IS NULL
		WHERE p.status = 'confirmed'
		  AND p.deleted_at IS NULL
		  `+buildDateFilter("p.payment_date", startDate, endDate)+`
		GROUP BY pkg.id, pkg.name
		ORDER BY total DESC
	`, buildDateArgs(startDate, endDate)...)

	var rows []IncomeByPackageRow
	err := query.Scan(&rows).Error
	return rows, err
}

// buildDateFilter returns extra WHERE clauses for a date column, using positional $N placeholders.
func buildDateFilter(col string, startDate *time.Time, endDate *time.Time) string {
	clause := ""
	if startDate != nil {
		clause += " AND " + col + " >= ?"
	}
	if endDate != nil {
		clause += " AND " + col + " < ?"
	}
	return clause
}

// buildDateArgs returns the date values in the same order as buildDateFilter.
func buildDateArgs(startDate *time.Time, endDate *time.Time) []interface{} {
	var args []interface{}
	if startDate != nil {
		args = append(args, *startDate)
	}
	if endDate != nil {
		args = append(args, *endDate)
	}
	return args
}
