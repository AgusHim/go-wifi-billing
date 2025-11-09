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

	// health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
