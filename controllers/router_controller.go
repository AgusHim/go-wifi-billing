package controllers

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RouterController struct {
	service services.RouterService
}

func NewRouterController(service services.RouterService) *RouterController {
	return &RouterController{service: service}
}

func (c *RouterController) RegisterRoutes(router fiber.Router) {
	r := router.Group("/admin_api/routers")
	r.Get("/", c.GetAll)
	r.Get("/:id", c.GetByID)
	r.Get("/:id/resources", c.GetResources)
	r.Get("/:id/import-preview", c.PreviewImport)
	r.Get("/:id/import-batches", c.ListImportBatches)
	r.Post("/", c.Create)
	r.Post("/health-check", c.RunHealthCheckAll)
	r.Post("/:id/import-staging", c.StageImport)
	r.Post("/:id/test-connection", c.TestConnection)
	r.Put("/:id", c.Update)
	r.Delete("/:id", c.Delete)
	router.Get("/admin_api/router-import-batches/:id", c.GetImportBatch)
}

func (c *RouterController) Create(ctx *fiber.Ctx) error {
	var input models.Router
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	created, err := c.service.Create(&input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": created, "message": "Router created"})
}

func (c *RouterController) GetAll(ctx *fiber.Ctx) error {
	list, err := c.service.GetAll()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": list, "message": "Success get data"})
}

func (c *RouterController) GetByID(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	item, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": item, "message": "Success get data"})
}

func (c *RouterController) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	var input models.Router
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid payload"})
	}
	updated, err := c.service.Update(id, &input)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": updated, "message": "Router updated"})
}

func (c *RouterController) TestConnection(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	result, err := c.service.TestConnection(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router connection tested"})
}

func (c *RouterController) RunHealthCheckAll(ctx *fiber.Ctx) error {
	result, err := c.service.RunHealthCheckAll()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router health check executed"})
}

func (c *RouterController) GetResources(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	kind := ctx.Query("kind", "")
	result, err := c.service.FetchResources(id, kind)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router resources fetched"})
}

func (c *RouterController) PreviewImport(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	mode := ctx.Query("mode", "all")
	result, err := c.service.PreviewImport(id, mode)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router import preview generated"})
}

func (c *RouterController) StageImport(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	mode := ctx.Query("mode", "all")
	result, err := c.service.StageImport(id, mode)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": result, "message": "Router import staged"})
}

func (c *RouterController) ListImportBatches(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	result, err := c.service.ListImportBatches(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router import batches fetched"})
}

func (c *RouterController) GetImportBatch(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	result, err := c.service.GetImportBatch(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": result, "message": "Router import batch fetched"})
}

func (c *RouterController) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid ID"})
	}
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Router deleted"})
}
