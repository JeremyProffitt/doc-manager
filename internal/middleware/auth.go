package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// AuthRequired returns a Fiber middleware that validates session cookies
// via JWT and the session store. Unauthenticated requests are redirected
// to /login. Certain paths are exempt from authentication.
func AuthRequired(sessionStore store.SessionStore, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip auth for exempt paths
		if path == "/login" || strings.HasPrefix(path, "/static/") || path == "/health" {
			return c.Next()
		}

		// Read session cookie
		cookie := c.Cookies("session")
		if cookie == "" {
			return c.Redirect("/login")
		}

		// Parse and validate JWT
		token, err := jwt.Parse(cookie, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Redirect("/login")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Redirect("/login")
		}

		sessionToken, ok := claims["session_token"].(string)
		if !ok {
			return c.Redirect("/login")
		}

		// Validate session exists in store
		session, err := sessionStore.GetSession(sessionToken)
		if err != nil || session == nil {
			return c.Redirect("/login")
		}

		// Set user info in context for downstream handlers
		c.Locals("userEmail", session.UserEmail)
		return c.Next()
	}
}
