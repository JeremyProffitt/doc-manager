package handlers

import "github.com/gofiber/fiber/v2"

// Home renders the dashboard page with the authenticated user's email.
func Home(c *fiber.Ctx) error {
	return c.Render("home", fiber.Map{
		"UserEmail": c.Locals("userEmail"),
	}, "layouts/base")
}
