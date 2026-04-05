package routes

import (
	"github.com/Agushim/go_wifi_billing/controllers"
	"github.com/gofiber/fiber/v2"
)

func Setup(
	app *fiber.App,
	coverageCtrl *controllers.CoverageController,
	userCtrl *controllers.UserController,
	packageCtrl *controllers.PackageController,
	odcCtrl *controllers.OdcController,
	odpCtrl *controllers.OdpController,
	customerCtrl *controllers.CustomerController,
	subscriptionCtrl *controllers.SubscriptionController,
	billCtrl *controllers.BillController,
	complainCtrl *controllers.ComplainController,
	paymentCtrl *controllers.PaymentController,
	routerCtrl *controllers.RouterController,
	provisioningCtrl *controllers.ProvisioningController,
	networkPlanCtrl *controllers.NetworkPlanController,
	serviceAccountCtrl *controllers.ServiceAccountController,
	renewalCtrl *controllers.RenewalController,
	voucherCtrl *controllers.VoucherController,
	whatsappCtrl *controllers.WhatsAppController,
	whatsappTemplateCtrl *controllers.WhatsAppTemplateController,
	expenseCtrl *controllers.ExpenseController,
	financeCtrl *controllers.FinanceController,
) {

	coverageCtrl.RegisterRoutes(app)
	userCtrl.RegisterRoutes(app)
	packageCtrl.RegisterRoutes(app)
	odcCtrl.RegisterRoutes(app)
	odpCtrl.RegisterRoutes(app)
	customerCtrl.RegisterRoutes(app)
	subscriptionCtrl.RegisterRoutes(app)
	billCtrl.RegisterRoutes(app)
	complainCtrl.RegisterRoutes(app)
	paymentCtrl.RegisterRoutes(app)
	routerCtrl.RegisterRoutes(app)
	provisioningCtrl.RegisterRoutes(app)
	networkPlanCtrl.RegisterRoutes(app)
	serviceAccountCtrl.RegisterRoutes(app)
	renewalCtrl.RegisterRoutes(app)
	voucherCtrl.RegisterRoutes(app)
	whatsappCtrl.RegisterRoutes(app)
	whatsappTemplateCtrl.RegisterRoutes(app)
	expenseCtrl.RegisterRoutes(app)
	financeCtrl.RegisterRoutes(app)

	// health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
