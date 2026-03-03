package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/JeremyProffitt/doc-manager/internal/config"
	"github.com/JeremyProffitt/doc-manager/internal/handlers"
	"github.com/JeremyProffitt/doc-manager/internal/middleware"
	"github.com/JeremyProffitt/doc-manager/internal/services"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

func main() {
	cfg := config.Load()

	// Initialize DynamoDB client
	dynamoClient, err := store.NewDynamoClient(cfg)
	if err != nil {
		log.Fatalf("failed to create DynamoDB client: %v", err)
	}

	// Initialize AWS config for S3
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	// Create S3 client and presign client
	s3Client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(s3Client)

	// Create stores
	userStore := store.NewUserStore(dynamoClient, cfg.UsersTable)
	sessionStore := store.NewSessionStore(dynamoClient, cfg.SessionsTable)
	formStore := store.NewFormStore(dynamoClient, cfg.FormsTable)
	fieldStore := store.NewFieldStore(dynamoClient, cfg.FieldPlacementsTable)
	customerStore := store.NewCustomerStore(dynamoClient, cfg.CustomersTable)
	settingsStore := store.NewSettingsStore(dynamoClient, cfg.SettingsTable)

	// Create services
	s3Service := services.NewS3Service(s3Client, presignClient, cfg.S3Bucket)
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	bedrockService := services.NewBedrockService(bedrockClient, cfg.BedrockModelID)
	analysisService := services.NewAnalysisService(bedrockService, s3Service, formStore, fieldStore, settingsStore)

	// Create handlers
	authHandler := handlers.NewAuthHandler(userStore, sessionStore, cfg.JWTSecret)
	formsHandler := handlers.NewFormsHandler(formStore, fieldStore, s3Service)
	formsHandler.SetAnalysisService(analysisService)
	editorHandler := handlers.NewEditorHandler(formStore, fieldStore, s3Service)
	versionsHandler := handlers.NewVersionsHandler(fieldStore)
	customersHandler := handlers.NewCustomersHandler(customerStore)
	settingsHandler := handlers.NewSettingsHandler(settingsStore)

	// Set up template engine
	engine := html.New("./templates", ".html")
	app := fiber.New(fiber.Config{
		AppName: "Doc-Manager",
		Views:   engine,
	})

	// Serve static files
	app.Static("/static", "./static")

	// Health check (before auth middleware so it's always accessible)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Apply auth middleware to all routes
	app.Use(middleware.AuthRequired(sessionStore, cfg.JWTSecret))

	// Auth routes
	app.Get("/login", authHandler.GetLogin)
	app.Post("/login", authHandler.PostLogin)
	app.Post("/logout", authHandler.PostLogout)

	// Dashboard
	app.Get("/", handlers.Home)

	// Form routes
	app.Get("/forms", formsHandler.ListForms)
	app.Get("/forms/:id/edit", editorHandler.EditForm)
	app.Get("/forms/:id", formsHandler.GetForm)
	app.Post("/api/forms/upload-url", formsHandler.GetUploadURL)
	app.Post("/api/forms/:id/upload-complete", formsHandler.UploadComplete)
	app.Post("/api/forms/:id/analyze", formsHandler.AnalyzeForm)
	app.Delete("/api/forms/:id", formsHandler.DeleteForm)

	// Field placement version routes
	app.Get("/api/forms/:id/fields", versionsHandler.GetCurrentFields)
	app.Get("/api/forms/:id/fields/versions", versionsHandler.ListVersions)
	app.Get("/api/forms/:id/fields/:version", versionsHandler.GetVersion)
	app.Post("/api/forms/:id/fields/revert/:v", versionsHandler.RevertToVersion)
	app.Put("/api/forms/:id/fields", versionsHandler.SaveFields)

	// Customer routes
	app.Get("/customers", customersHandler.ListCustomers)
	app.Get("/customers/new", customersHandler.NewCustomer)
	app.Get("/customers/:id", customersHandler.GetCustomer)
	app.Post("/customers", customersHandler.CreateCustomer)
	app.Put("/api/customers/:id", customersHandler.UpdateCustomer)
	app.Delete("/api/customers/:id", customersHandler.DeleteCustomer)
	app.Get("/api/customers", customersHandler.ListCustomersAPI)

	// Settings routes
	app.Get("/settings/fields", settingsHandler.GetFields)
	app.Put("/api/settings/fields", settingsHandler.UpdateFields)

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		fiberLambda := fiberadapter.New(app)
		lambda.Start(fiberLambda.ProxyWithContext)
	} else {
		log.Fatal(app.Listen(":3000"))
	}
}
