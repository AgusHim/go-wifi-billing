package controllers

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Agushim/go_wifi_billing/dto"
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserController struct {
	service              services.UserService
	authorizationService services.AuthorizationService
}

func NewUserController(s services.UserService, authorizationService ...services.AuthorizationService) *UserController {
	controller := &UserController{service: s}
	if len(authorizationService) > 0 {
		controller.authorizationService = authorizationService[0]
	}
	return controller
}

func (ctrl *UserController) RegisterRoutes(router fiber.Router) {
	api := router.Group("/api")

	api.Post("/auth/register", ctrl.Register)
	api.Post("/auth/login", ctrl.Login)
	api.Get("/auth/me", middlewares.UserProtected(), ctrl.GetMe)
	api.Put("/auth/me", middlewares.UserProtected(), ctrl.UpdateMe)

	users := api.Group("/users", middlewares.UserProtected())
	users.Post("/", ctrl.Create)
	users.Get("/", ctrl.GetAll)
	users.Get("/:id", ctrl.GetByID)
	users.Put("/:id", ctrl.Update)
	users.Delete("/:id", ctrl.Delete)
}

func (ctrl *UserController) Register(c *fiber.Ctx) error {
	var input dto.RegisterDTO
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid input"})
	}

	user, err := ctrl.service.Register(input)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": user, "message": "User registered"})
}

func (ctrl *UserController) Create(c *fiber.Ctx) error {
	var input dto.CreateUserDTO
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid input"})
	}

	user, err := ctrl.service.Create(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": user, "message": "User created"})
}

func (ctrl *UserController) Login(c *fiber.Ctx) error {
	var input dto.LoginDTO
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid input"})
	}

	token, user, err := ctrl.service.Login(input)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"token": token, "user": user}, "message": "Login success"})
}

func (ctrl *UserController) GetMe(c *fiber.Ctx) error {
	userClaims := c.Locals("user").(jwt.MapClaims)
	userID := userClaims["user_id"].(string)

	user, err := ctrl.service.GetByID(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User not found"})
	}

	if ctrl.authorizationService == nil {
		return c.JSON(fiber.Map{"success": true, "data": user, "message": "Success get data"})
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
	}
	decision, err := ctrl.authorizationService.Resolve(c.UserContext(), parsedUserID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "forbidden"})
	}
	permissions := make([]string, 0, len(decision.Permissions))
	for permission := range decision.Permissions {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	roleName := decision.RoleKey
	if user.RoleDefinition != nil && user.RoleDefinition.Name != "" {
		roleName = user.RoleDefinition.Name
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"id": user.ID, "name": user.Name, "email": user.Email, "phone": user.Phone,
		"role":        fiber.Map{"id": decision.RoleID, "key": decision.RoleKey, "name": roleName, "is_owner": decision.IsOwner},
		"legacy_role": user.Role, "permissions": permissions, "permission_version": decision.PermissionVersion,
	}, "message": "Success get current authorization profile"})
}

func (ctrl *UserController) UpdateMe(c *fiber.Ctx) error {
	userClaims := c.Locals("user").(jwt.MapClaims)
	userID := userClaims["user_id"].(string)

	var input dto.UpdateProfileDTO
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid input"})
	}

	user, err := ctrl.service.UpdateProfile(userID, input)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": user, "message": "Profile updated"})
}

func (ctrl *UserController) GetAll(c *fiber.Ctx) error {
	search := c.Query("search", "")
	roleParam := c.Query("role", "")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	var roles []string
	if roleParam != "" {
		for _, r := range strings.Split(roleParam, ",") {
			if trimmed := strings.TrimSpace(r); trimmed != "" {
				roles = append(roles, trimmed)
			}
		}
	}

	users, total, err := ctrl.service.GetAll(page, limit, roles, search)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"meta": fiber.Map{
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": int((total + int64(limit) - 1) / int64(limit)),
			},
		},
		"data":    users,
		"message": "Success get data",
	})
}

func (ctrl *UserController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := ctrl.service.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": user, "message": "Success get data"})
}

func (ctrl *UserController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var input dto.UpdateUserDTO
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid input"})
	}

	user, err := ctrl.service.Update(id, input)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": user, "message": "User updated"})
}

func (ctrl *UserController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.service.Delete(id); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "User deleted"})
}
