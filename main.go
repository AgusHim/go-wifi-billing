package main

import (
	"log"
	"os"

	"github.com/Agushim/go_wifi_billing/controllers"
	"github.com/Agushim/go_wifi_billing/db"
	"github.com/Agushim/go_wifi_billing/db/seed"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/routes"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Init DB (Postgres if POSTGRE_URL set, else SQLite)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dsn := os.Getenv("POSTGRES_URL")
	log.Printf("dsn: %s", dsn)
	gormDB, err := db.InitDB(dsn)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}

	// Migrate
	if err := db.AutoMigrate(gormDB); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	seed.Seed(gormDB)

	// Init repository, service, controller
	coverageRepo := repositories.NewCoverageRepository(gormDB)
	coverageSvc := services.NewCoverageService(coverageRepo)
	coverageCtrl := controllers.NewCoverageController(coverageSvc)

	userRepo := repositories.NewUserRepository(gormDB)
	userSvc := services.NewUserService(userRepo)
	userCtrl := controllers.NewUserController(userSvc)

	packageRepo := repositories.NewPackageRepository(gormDB)
	packageSvc := services.NewPackageService(packageRepo)
	packageCtrl := controllers.NewPackageController(packageSvc)

	odcRepo := repositories.NewOdcRepository(gormDB)
	odcSvc := services.NewOdcService(odcRepo)
	odcCtrl := controllers.NewOdcController(odcSvc)

	odpRepo := repositories.NewOdpRepository(gormDB)
	odpSvc := services.NewOdpService(odpRepo)
	odpCtrl := controllers.NewOdpController(odpSvc)

	subscriptionRepo := repositories.NewSubscriptionRepository(gormDB)
	subscriptionSvc := services.NewSubscriptionService(subscriptionRepo)
	subscriptionCtrl := controllers.NewSubscriptionController(subscriptionSvc)

	customerRepo := repositories.NewCustomerRepository(gormDB)
	customerSvc := services.NewCustomerService(customerRepo, userSvc, subscriptionSvc)
	customerCtrl := controllers.NewCustomerController(customerSvc)

	billRepo := repositories.NewBillRepository(gormDB)
	billSvc := services.NewBillService(billRepo, subscriptionRepo)
	billCtrl := controllers.NewBillController(billSvc)

	complainRepo := repositories.NewComplainRepository(gormDB)
	complainSvc := services.NewComplainService(complainRepo)
	complainCtrl := controllers.NewComplainController(complainSvc)

	paymentRepo := repositories.NewPaymentRepository(gormDB)
	paymentSvc := services.NewPaymentService(paymentRepo, subscriptionRepo, billRepo)
	paymentCtrl := controllers.NewPaymentController(paymentSvc)

	// Setup Fiber
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000, https://localhost:3000, http://103.103.22.212, https://103.103.22.212, https://103.103.22.212:3000",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: true,
	}))

	// Register routes for all controllers
	routes.Setup(
		app,
		coverageCtrl,
		userCtrl,
		packageCtrl,
		odcCtrl,
		odpCtrl,
		customerCtrl,
		subscriptionCtrl,
		billCtrl,
		complainCtrl,
		paymentCtrl,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
