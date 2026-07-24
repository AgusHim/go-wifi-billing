package middlewares

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestUserProtectedReturns401ForInvalidAuthentication(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	testCases := []struct {
		name          string
		authorization string
	}{
		{name: "missing"},
		{name: "not bearer", authorization: "Basic abc"},
		{name: "empty bearer", authorization: "Bearer"},
		{name: "malformed", authorization: "Bearer invalid"},
		{name: "missing claims", authorization: "Bearer " + signMiddlewareToken(t, jwt.SigningMethodHS256, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()})},
		{name: "wrong algorithm", authorization: "Bearer " + signMiddlewareToken(t, jwt.SigningMethodHS384, validMiddlewareClaims("admin"))},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/private", UserProtected(), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
			request := httptest.NewRequest("GET", "/private", nil)
			if testCase.authorization != "" {
				request.Header.Set("Authorization", testCase.authorization)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", response.StatusCode)
			}
		})
	}
}

func validMiddlewareClaims(role string) jwt.MapClaims {
	return jwt.MapClaims{
		"user_id": uuid.NewString(),
		"role":    role,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
}

func signMiddlewareToken(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte("phase-zero-test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}
