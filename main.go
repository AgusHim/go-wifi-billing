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
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Init DB (Postgres if POSTGRE_URL set, else SQLite)
	dsn := os.Getenv("POSTGRE_URL")
	sqlitePath := os.Getenv("SQLITE_PATH")

	var gormDB, err = db.InitDB(dsn, sqlitePath)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}

	// Migrate
	if err := db.AutoMigrate(gormDB); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	// Seed data
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

	customerRepo := repositories.NewCustomerRepository(gormDB)
	customerSvc := services.NewCustomerService(customerRepo)
	customerCtrl := controllers.NewCustomerController(customerSvc)

	subscriptionRepo := repositories.NewSubscriptionRepository(gormDB)
	subscriptionSvc := services.NewSubscriptionService(subscriptionRepo)
	subscriptionCtrl := controllers.NewSubscriptionController(subscriptionSvc)

	billRepo := repositories.NewBillRepository(gormDB)
	billSvc := services.NewBillService(billRepo)
	billCtrl := controllers.NewBillController(billSvc)

	// Setup Fiber
	app := fiber.New()

	// Register routes for all controllers
	routes.Setup(app, coverageCtrl, userCtrl, packageCtrl, odcCtrl, odpCtrl, customerCtrl, subscriptionCtrl, billCtrl)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
