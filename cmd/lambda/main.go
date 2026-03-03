package main

import (
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/JeremyProffitt/doc-manager/internal/config"
	"github.com/JeremyProffitt/doc-manager/internal/handlers"
	"github.com/JeremyProffitt/doc-manager/internal/middleware"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

func main() {
	cfg := config.Load()

	// Initialize DynamoDB client
	dynamoClient, err := store.NewDynamoClient(cfg)
	if err != nil {
		log.Fatalf("failed to create DynamoDB client: %v", err)
	}

	// Create stores
	userStore := store.NewUserStore(dynamoClient, cfg.UsersTable)
	sessionStore := store.NewSessionStore(dynamoClient, cfg.SessionsTable)

	// Create handlers
	authHandler := handlers.NewAuthHandler(userStore, sessionStore, cfg.JWTSecret)

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

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		fiberLambda := fiberadapter.New(app)
		lambda.Start(fiberLambda.ProxyWithContext)
	} else {
		log.Fatal(app.Listen(":3000"))
	}
}
