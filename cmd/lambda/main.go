package main

import (
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
	"github.com/gofiber/fiber/v2"

	"github.com/JeremyProffitt/doc-manager/internal/config"
)

func main() {
	cfg := config.Load()
	_ = cfg // will be used to initialize stores and services

	app := fiber.New(fiber.Config{
		AppName: "Doc-Manager",
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// TODO: Register routes, middleware, template engine

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		fiberLambda := fiberadapter.New(app)
		lambda.Start(fiberLambda.ProxyWithContext)
	} else {
		log.Fatal(app.Listen(":3000"))
	}
}
