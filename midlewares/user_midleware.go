package middlewares

import (
	"errors"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidAuthentication = errors.New("invalid authentication")

func UserProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := c.Locals("user").(jwt.MapClaims); ok {
			return c.Next()
		}
		if err := AuthenticateRequest(c); err != nil {
			return unauthorizedResponse(c)
		}
		return c.Next()
	}
}

// AuthenticateRequest validates the JWT and stores trusted identity claims
// without advancing the Fiber handler chain. It is shared by the centralized
// route-permission registry and UserProtected.
func AuthenticateRequest(c *fiber.Ctx) error {
	authorization := strings.TrimSpace(c.Get("Authorization"))
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return ErrInvalidAuthentication
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		return ErrInvalidAuthentication
	}
	token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return ErrInvalidAuthentication
	}

	if !token.Valid {
		return ErrInvalidAuthentication
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrInvalidAuthentication
	}
	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)
	if _, err := uuid.Parse(strings.TrimSpace(userID)); err != nil || strings.TrimSpace(role) == "" {
		return ErrInvalidAuthentication
	}

	c.Locals("user", claims)
	return nil
}

func unauthorizedResponse(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
}
