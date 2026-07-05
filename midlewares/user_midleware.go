package middlewares

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func UserProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")

		// Log request info
		log.Printf("[AUTH] %s %s", c.Method(), c.OriginalURL())

		if tokenString == "" {
			log.Println("[AUTH] Missing token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "missing token",
			})
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		jwtSecret := []byte(os.Getenv("JWT_SECRET"))
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			log.Printf("[AUTH] Token parse error: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "invalid token",
			})
		}

		if !token.Valid {
			log.Println("[AUTH] Invalid token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "invalid token",
			})
		}

		log.Println("[AUTH] Token validated successfully")

		c.Locals("user", token.Claims)
		return c.Next()
	}
}

func RequireRoles(allowed ...string) fiber.Handler {
	roles := make(map[string]bool, len(allowed))
	for _, role := range allowed {
		roles[strings.ToLower(strings.TrimSpace(role))] = true
	}

	return func(c *fiber.Ctx) error {
		userClaims, ok := c.Locals("user").(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "unauthorized",
			})
		}

		role, _ := userClaims["role"].(string)
		if !roles[strings.ToLower(strings.TrimSpace(role))] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "forbidden",
			})
		}

		return c.Next()
	}
}
