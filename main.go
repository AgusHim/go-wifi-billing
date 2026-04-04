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

	midtrans_server := os.Getenv("MIDTRANS_SERVER_KEY")
	log.Printf("midtrans_server: %s", midtrans_server)

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

	routerRepo := repositories.NewRouterRepository(gormDB)
	routerImportBatchRepo := repositories.NewRouterImportBatchRepository(gormDB)
	routerImportItemRepo := repositories.NewRouterImportItemRepository(gormDB)
	networkPlanRepo := repositories.NewNetworkPlanRepository(gormDB)
	serviceAccountRepo := repositories.NewServiceAccountRepository(gormDB)
	serviceStatusHistoryRepo := repositories.NewServiceStatusHistoryRepository(gormDB)
	voucherBatchRepo := repositories.NewVoucherBatchRepository(gormDB)
	voucherRepo := repositories.NewVoucherRepository(gormDB)
	renewalHistoryRepo := repositories.NewSubscriptionRenewalHistoryRepository(gormDB)

	odcRepo := repositories.NewOdcRepository(gormDB)
	odcSvc := services.NewOdcService(odcRepo)
	odcCtrl := controllers.NewOdcController(odcSvc)

	odpRepo := repositories.NewOdpRepository(gormDB)
	odpSvc := services.NewOdpService(odpRepo)
	odpCtrl := controllers.NewOdpController(odpSvc)

	subscriptionRepo := repositories.NewSubscriptionRepository(gormDB)
	subscriptionSvc := services.NewSubscriptionService(subscriptionRepo)

	customerRepo := repositories.NewCustomerRepository(gormDB)
	customerSvc := services.NewCustomerService(customerRepo, userSvc, subscriptionSvc)
	customerCtrl := controllers.NewCustomerController(customerSvc)

	// Init WhatsApp service
	whatsappBaseURL := os.Getenv("WHATSAPP_BOT_URL")
	if whatsappBaseURL == "" {
		whatsappBaseURL = "http://localhost:3030/api/public/v1"
	}
	whatsappAPIKey := os.Getenv("WHATSAPP_API_KEY")
	waSvc := services.NewWhatsAppService(whatsappBaseURL, whatsappAPIKey)

	billRepo := repositories.NewBillRepository(gormDB)
	provisioningJobRepo := repositories.NewProvisioningJobRepository(gormDB)
	provisioningLogRepo := repositories.NewProvisioningLogRepository(gormDB)
	routerSvc := services.NewRouterService(
		routerRepo,
		provisioningLogRepo,
		networkPlanRepo,
		serviceAccountRepo,
		routerImportBatchRepo,
		routerImportItemRepo,
	)
	routerCtrl := controllers.NewRouterController(routerSvc)
	routerSvc.StartHealthCheckScheduler()
	provisioningSvc := services.NewProvisioningService(provisioningJobRepo, provisioningLogRepo)
	provisioningCtrl := controllers.NewProvisioningController(provisioningSvc)
	networkPlanSvc := services.NewNetworkPlanService(networkPlanRepo)
	networkPlanCtrl := controllers.NewNetworkPlanController(networkPlanSvc)
	serviceAccountSvc := services.NewServiceAccountService(serviceAccountRepo, provisioningJobRepo, provisioningLogRepo, routerRepo, serviceStatusHistoryRepo)
	serviceAccountCtrl := controllers.NewServiceAccountController(serviceAccountSvc)
	voucherSvc := services.NewVoucherService(voucherBatchRepo, voucherRepo, provisioningJobRepo, provisioningLogRepo, routerRepo)
	voucherCtrl := controllers.NewVoucherController(voucherSvc)
	whatsappCtrl := controllers.NewWhatsAppController(waSvc)
	waTemplateRepo := repositories.NewWhatsAppTemplateRepository(gormDB)
	waTemplateSvc := services.NewWhatsAppTemplateService(waTemplateRepo)
	waTemplateCtrl := controllers.NewWhatsAppTemplateController(waTemplateSvc)
	billingProvisioningSvc := services.NewBillingProvisioningService(serviceAccountRepo, serviceAccountSvc, subscriptionRepo, serviceStatusHistoryRepo)
	renewalSvc := services.NewRenewalService(subscriptionRepo, billRepo, renewalHistoryRepo)
	renewalCtrl := controllers.NewRenewalController(renewalSvc)
	subscriptionCtrl := controllers.NewSubscriptionController(subscriptionSvc, customerSvc, renewalSvc)
	renewalSvc.StartScheduler()
	billSvc := services.NewBillService(billRepo, subscriptionRepo, waSvc, billingProvisioningSvc)
	billCtrl := controllers.NewBillController(billSvc)

	complainRepo := repositories.NewComplainRepository(gormDB)
	complainSvc := services.NewComplainService(complainRepo)
	complainCtrl := controllers.NewComplainController(complainSvc)

	paymentRepo := repositories.NewPaymentRepository(gormDB)
	paymentSvc := services.NewPaymentService(paymentRepo, subscriptionRepo, billRepo, billingProvisioningSvc, renewalSvc)
	paymentCtrl := controllers.NewPaymentController(paymentSvc)

	// Setup Fiber
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000, https://localhost:3000, https://cantika.net, https://www.cantika.net",
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
		routerCtrl,
		provisioningCtrl,
		networkPlanCtrl,
		serviceAccountCtrl,
		renewalCtrl,
		voucherCtrl,
		whatsappCtrl,
		waTemplateCtrl,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
