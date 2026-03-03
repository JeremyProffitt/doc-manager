package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// AuthHandler holds dependencies for authentication-related HTTP handlers.
type AuthHandler struct {
	userStore    store.UserStore
	sessionStore store.SessionStore
	jwtSecret    string
}

// NewAuthHandler creates a new AuthHandler with the given stores and secret.
func NewAuthHandler(us store.UserStore, ss store.SessionStore, jwtSecret string) *AuthHandler {
	return &AuthHandler{userStore: us, sessionStore: ss, jwtSecret: jwtSecret}
}

// GetLogin renders the login page. If the user is already authenticated
// (userEmail set in context by middleware), it redirects to /.
func (h *AuthHandler) GetLogin(c *fiber.Ctx) error {
	if c.Locals("userEmail") != nil {
		return c.Redirect("/")
	}
	return c.Render("login", fiber.Map{}, "layouts/base")
}

// PostLogin handles login form submission. It validates credentials,
// creates a session, signs a JWT, and sets a session cookie.
func (h *AuthHandler) PostLogin(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	if email == "" || password == "" {
		return c.Status(400).Render("login", fiber.Map{
			"Error": "Email and password are required",
			"Email": email,
		}, "layouts/base")
	}

	user, err := h.userStore.GetUser(email)
	if err != nil {
		return c.Status(500).Render("login", fiber.Map{"Error": "An error occurred"}, "layouts/base")
	}

	if user == nil {
		return c.Status(200).Render("login", fiber.Map{
			"Error": "Invalid email or password",
			"Email": email,
		}, "layouts/base")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return c.Status(200).Render("login", fiber.Map{
			"Error": "Invalid email or password",
			"Email": email,
		}, "layouts/base")
	}

	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.Status(500).Render("login", fiber.Map{"Error": "An error occurred"}, "layouts/base")
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Store session (24-hour TTL)
	expiresAt := time.Now().Add(24 * time.Hour)
	session := &models.Session{
		Token:     sessionToken,
		UserEmail: user.Email,
		ExpiresAt: expiresAt.Unix(),
	}
	if err := h.sessionStore.CreateSession(session); err != nil {
		return c.Status(500).Render("login", fiber.Map{"Error": "An error occurred"}, "layouts/base")
	}

	// Create JWT containing the session token
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_token": sessionToken,
		"exp":           expiresAt.Unix(),
	})
	signed, err := jwtToken.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return c.Status(500).Render("login", fiber.Map{"Error": "An error occurred"}, "layouts/base")
	}

	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    signed,
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.Redirect("/")
}

// PostLogout handles logout by deleting the session from the store,
// clearing the cookie, and redirecting to /login.
func (h *AuthHandler) PostLogout(c *fiber.Ctx) error {
	// Parse JWT to get session token and delete from store
	cookie := c.Cookies("session")
	if cookie != "" {
		token, err := jwt.Parse(cookie, func(t *jwt.Token) (interface{}, error) {
			return []byte(h.jwtSecret), nil
		})
		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if sessionToken, ok := claims["session_token"].(string); ok {
					h.sessionStore.DeleteSession(sessionToken)
				}
			}
		}
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.Redirect("/login")
}
