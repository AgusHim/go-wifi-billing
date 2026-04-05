package dto

type CreateExpenseRequest struct {
	Title       string `form:"title" json:"title"`
	Category    string `form:"category" json:"category"`
	Amount      int    `form:"amount" json:"amount"`
	ExpenseDate string `form:"expense_date" json:"expense_date"`
	Description string `form:"description" json:"description"`
	AdminID     string `form:"admin_id" json:"admin_id"`
}
