package vm

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-compute-service/pkg/response"
)

// Handler holds the HTTP handlers for the VM module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new VM handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts VM routes under the given fiber router.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	vms := r.Group("/vms")
	vms.Get("/", h.List)
	vms.Post("/", h.Create)
	vms.Get("/:id", h.Get)
	vms.Post("/:id/start", h.Start)
	vms.Post("/:id/stop", h.Stop)
	vms.Post("/:id/reboot", h.Reboot)
	vms.Delete("/:id", h.Delete)
	vms.Get("/:id/metrics", h.Metrics)
}

func (h *Handler) List(c *fiber.Ctx) error {
	vms, err := h.svc.List(c.Context())
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.OK(c, vms)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.BadRequest(c, "Geçersiz VM ID")
	}
	vm, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.OK(c, vm)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateVMRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Geçersiz istek gövdesi")
	}
	if req.Name == "" || req.Template == 0 {
		return response.BadRequest(c, "name ve template zorunludur")
	}
	if req.CPUs == 0 {
		req.CPUs = 2
	}
	if req.MemoryMB == 0 {
		req.MemoryMB = 2048
	}
	vm, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.Created(c, vm)
}

func (h *Handler) Start(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.BadRequest(c, "Geçersiz VM ID")
	}
	res, err := h.svc.Start(c.Context(), id)
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.OK(c, res)
}

func (h *Handler) Stop(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.BadRequest(c, "Geçersiz VM ID")
	}
	res, err := h.svc.Stop(c.Context(), id)
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.OK(c, res)
}

func (h *Handler) Reboot(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.BadRequest(c, "Geçersiz VM ID")
	}
	res, err := h.svc.Reboot(c.Context(), id)
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.OK(c, res)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.BadRequest(c, "Geçersiz VM ID")
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return handleSvcErr(c, err)
	}
	return response.NoContent(c)
}

func (h *Handler) Metrics(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.BadRequest(c, "Geçersiz VM ID")
	}
	m, err := h.svc.Metrics(c.Context(), id)
	if err != nil {
		return handleSvcErr(c, err)
	}
	return response.OK(c, m)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseID(c *fiber.Ctx) (int, error) {
	return strconv.Atoi(c.Params("id"))
}

func handleSvcErr(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrNotConfigured) {
		return response.Unimplemented(c)
	}
	return response.Internal(c, err.Error())
}
