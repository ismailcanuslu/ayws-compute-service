package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-compute-service/internal/container"
	"github.com/ismailcanuslu/ayws-compute-service/internal/middleware"
	"github.com/ismailcanuslu/ayws-compute-service/internal/serverless"
	"github.com/ismailcanuslu/ayws-compute-service/internal/vm"
	"github.com/ismailcanuslu/ayws-compute-service/pkg/response"
)

// Setup mounts all routes and returns the configured Fiber app.
func Setup(
	vmH *vm.Handler,
	slH *serverless.Handler,
	ctH *container.Handler,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return response.Internal(c, err.Error())
		},
	})

	// ── Global middleware ──────────────────────────────────────────────────────
	app.Use(middleware.Logger())

	// ── Health ────────────────────────────────────────────────────────────────
	app.Get("/health", func(c *fiber.Ctx) error {
		return response.OK(c, fiber.Map{"status": "ok", "service": "ayws-compute"})
	})

	// ── Compute API ───────────────────────────────────────────────────────────
	api := app.Group("/api/compute")

	vmH.RegisterRoutes(api)
	slH.RegisterRoutes(api)
	ctH.RegisterRoutes(api)

	return app
}
