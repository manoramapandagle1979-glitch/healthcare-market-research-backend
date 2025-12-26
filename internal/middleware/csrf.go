package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/pkg/response"
)

const (
	CSRFTokenLength = 32
	CSRFHeaderName  = "X-CSRF-Token"
	CSRFCookieName  = "csrf_token"
	CSRFCacheTTL    = 1 * time.Hour
)

// CSRF returns a middleware that implements CSRF protection
// This uses the double-submit cookie pattern:
// 1. Server generates a random token and sends it in both a cookie and response header
// 2. Client must include the token in a custom header on state-changing requests
// 3. Server validates that the header token matches the cookie token
func CSRF() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()

		// Skip CSRF check for safe methods (GET, HEAD, OPTIONS)
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}

		// Get CSRF token from cookie
		cookieToken := c.Cookies(CSRFCookieName)

		// Get CSRF token from header
		headerToken := c.Get(CSRFHeaderName)

		// Validate that both tokens exist and match
		if cookieToken == "" || headerToken == "" {
			return response.Forbidden(c, "CSRF token missing")
		}

		if cookieToken != headerToken {
			return response.Forbidden(c, "CSRF token mismatch")
		}

		// Validate token exists in cache (hasn't expired or been revoked)
		var storedToken string
		cacheKey := fmt.Sprintf("csrf:%s", cookieToken)
		if err := cache.Get(cacheKey, &storedToken); err != nil {
			return response.Forbidden(c, "CSRF token invalid or expired")
		}

		return c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token and sets it in both cookie and header
func GenerateCSRFToken(c *fiber.Ctx) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store token in cache
	cacheKey := fmt.Sprintf("csrf:%s", token)
	if err := cache.Set(cacheKey, token, CSRFCacheTTL); err != nil {
		// Log error but don't fail if Redis is unavailable
		fmt.Printf("Warning: Failed to store CSRF token in cache: %v\n", err)
	}

	// Set token in cookie with Secure, HttpOnly, and SameSite attributes
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(CSRFCacheTTL.Seconds()),
		Secure:   true,  // Only send over HTTPS
		HTTPOnly: true,  // Not accessible via JavaScript (XSS protection)
		SameSite: "Strict", // CSRF protection
	})

	// Also set in response header for client to read
	c.Set(CSRFHeaderName, token)

	return token, nil
}

// CSRFTokenEndpoint returns a handler that generates and returns a CSRF token
func CSRFTokenEndpoint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := GenerateCSRFToken(c)
		if err != nil {
			return response.InternalError(c, "Failed to generate CSRF token")
		}

		return c.JSON(fiber.Map{
			"csrf_token": token,
		})
	}
}
