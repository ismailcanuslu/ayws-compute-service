package serverless

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-compute-service/pkg/response"
)

// Handler holds the HTTP handlers for the serverless module.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	fns := r.Group("/functions")
	fns.Get("/", h.List)
	fns.Post("/", h.Create)
	fns.Get("/:id", h.Get)
	fns.Post("/:id/invoke", h.Invoke)
	fns.Delete("/:id", h.Delete)
	fns.Get("/:id/logs", h.Logs)
}

func (h *Handler) List(c *fiber.Ctx) error {
	fns, err := h.svc.List(c.Context())
	if err != nil {
		return response.Internal(c, err.Error())
	}
	return response.OK(c, fns)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	fn, err := h.svc.Get(c.Context(), c.Params("id"))
	if err != nil {
		return handleErr(c, err)
	}
	return response.OK(c, fn)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateFunctionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Geçersiz istek gövdesi")
	}
	if req.Name == "" || req.Code == "" {
		return response.BadRequest(c, "name ve code zorunludur")
	}
	if req.Runtime == "" {
		req.Runtime = RuntimePython
	}
	fn, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return response.Internal(c, err.Error())
	}
	return response.Created(c, fn)
}

func (h *Handler) Invoke(c *fiber.Ctx) error {
	var req InvokeRequest
	_ = c.BodyParser(&req) // payload opsiyonel

	result, err := h.svc.Invoke(c.Context(), c.Params("id"), req.Payload)
	if err != nil {
		return handleErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.svc.Delete(c.Context(), c.Params("id")); err != nil {
		return handleErr(c, err)
	}
	return response.NoContent(c)
}

func (h *Handler) Logs(c *fiber.Ctx) error {
	logs, err := h.svc.Logs(c.Context(), c.Params("id"))
	if err != nil {
		return handleErr(c, err)
	}
	return response.OK(c, logs)
}

func handleErr(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrNotConfigured) {
		return response.Unimplemented(c)
	}
	if errors.Is(err, ErrFunctionNotFound) {
		return response.NotFound(c, err.Error())
	}
	return response.Internal(c, err.Error())
}
