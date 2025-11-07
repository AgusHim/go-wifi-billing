package controllers

import (
	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type UserController struct {
	service services.UserService
}

func NewUserController(s services.UserService) *UserController {
	return &UserController{service: s}
}

func (ctrl *UserController) RegisterRoutes(router fiber.Router) {
	api := router.Group("/api")

	api.Post("/auth/register", ctrl.Register)
	api.Post("/auth/login", ctrl.Login)

	users := api.Group("/users")
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

func (ctrl *UserController) GetAll(c *fiber.Ctx) error {
	role := c.Query("role", "")
	users, err := ctrl.service.GetAll(role)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": users, "message": "Success get data"})
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
	var input models.User
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid input"})
	}

	user, err := ctrl.service.Update(id, &input)
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
