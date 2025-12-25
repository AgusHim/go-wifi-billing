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
