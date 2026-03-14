package container

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-compute-service/pkg/response"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	docker *DockerService
	k8s    *K8sService
}

func NewHandler(docker *DockerService, k8s *K8sService) *Handler {
	return &Handler{docker: docker, k8s: k8s}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	// ── Docker containers ──────────────────────────────────────────────────────
	c := r.Group("/containers")
	c.Get("/", h.ListContainers)
	c.Post("/", h.CreateContainer)
	c.Post("/prune", h.Prune)
	c.Get("/:id", h.GetContainer)
	c.Post("/:id/start", h.StartContainer)
	c.Post("/:id/stop", h.StopContainer)
	c.Delete("/:id", h.RemoveContainer)
	c.Get("/:id/logs", h.ContainerLogs)
	c.Get("/:id/stats", h.ContainerStats)

	// ── Networks ───────────────────────────────────────────────────────────────
	n := r.Group("/networks")
	n.Get("/", h.ListNetworks)
	n.Post("/", h.CreateNetwork)

	// ── Volumes ────────────────────────────────────────────────────────────────
	v := r.Group("/volumes")
	v.Get("/", h.ListVolumes)
	v.Post("/", h.CreateVolume)

	// ── Kubernetes ────────────────────────────────────────────────────────────
	k := r.Group("/k8s")
	k.Get("/namespaces", h.ListNamespaces)
	k.Get("/namespaces/:ns/pods", h.ListPods)
	k.Get("/deployments", h.ListDeployments)
	k.Post("/deployments", h.CreateDeployment)
	k.Put("/deployments/:name", h.ScaleDeployment)
	k.Delete("/deployments/:name", h.DeleteDeployment)
}

// ── helpers ────────────────────────────────────────────────────────────────────

func dockerNotConfigured(c *fiber.Ctx) error {
	return response.Fail(c, fiber.StatusServiceUnavailable, "Docker yapılandırılmamış veya bağlantı kurulamadı")
}

func k8sNotConfigured(c *fiber.Ctx) error {
	return response.Fail(c, fiber.StatusServiceUnavailable, "Kubernetes yapılandırılmamış veya bağlantı kurulamadı")
}

func internalErr(c *fiber.Ctx, err error) error {
	return response.Internal(c, err.Error())
}

// ── Container handlers ─────────────────────────────────────────────────────────

func (h *Handler) ListContainers(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	list, err := h.docker.ListContainers(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("container listesi alınamadı")
		return internalErr(c, err)
	}
	return response.OK(c, list)
}

func (h *Handler) GetContainer(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	details, err := h.docker.GetContainerDetails(c.Context(), c.Params("id"))
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, details)
}

func (h *Handler) CreateContainer(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	var req CreateContainerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Geçersiz istek gövdesi: "+err.Error())
	}
	if req.Image == "" {
		return response.BadRequest(c, "image alanı zorunludur")
	}
	cont, err := h.docker.CreateContainer(c.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("container oluşturulamadı")
		return internalErr(c, err)
	}
	return response.Created(c, cont)
}

func (h *Handler) StartContainer(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	if err := h.docker.StartContainer(c.Context(), c.Params("id")); err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, fiber.Map{"status": "started"})
}

func (h *Handler) StopContainer(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	if err := h.docker.StopContainer(c.Context(), c.Params("id")); err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, fiber.Map{"status": "stopped"})
}

func (h *Handler) RemoveContainer(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	force := c.QueryBool("force", false)
	if err := h.docker.RemoveContainer(c.Context(), c.Params("id"), force); err != nil {
		return internalErr(c, err)
	}
	return response.NoContent(c)
}

func (h *Handler) ContainerLogs(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	tail := c.Query("tail", "100")
	logs, err := h.docker.Logs(c.Context(), c.Params("id"), tail)
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, fiber.Map{"logs": logs})
}

func (h *Handler) ContainerStats(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	stats, err := h.docker.GetContainerStats(c.Context(), c.Params("id"))
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, stats)
}

func (h *Handler) Prune(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	if err := h.docker.PruneContainers(c.Context()); err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, fiber.Map{"status": "pruned"})
}

// ── Network handlers ───────────────────────────────────────────────────────────

func (h *Handler) ListNetworks(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	nets, err := h.docker.ListNetworks(c.Context())
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, nets)
}

func (h *Handler) CreateNetwork(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	var body struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name zorunludur")
	}
	net, err := h.docker.CreateNetwork(c.Context(), body.Name, body.Driver)
	if err != nil {
		return internalErr(c, err)
	}
	return response.Created(c, net)
}

// ── Volume handlers ────────────────────────────────────────────────────────────

func (h *Handler) ListVolumes(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	vols, err := h.docker.ListVolumes(c.Context())
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, vols)
}

func (h *Handler) CreateVolume(c *fiber.Ctx) error {
	if h.docker == nil {
		return dockerNotConfigured(c)
	}
	var body struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name zorunludur")
	}
	vol, err := h.docker.CreateVolume(c.Context(), body.Name, body.Driver)
	if err != nil {
		return internalErr(c, err)
	}
	return response.Created(c, vol)
}

// ── Kubernetes handlers ────────────────────────────────────────────────────────

func (h *Handler) ListNamespaces(c *fiber.Ctx) error {
	if h.k8s == nil {
		return k8sNotConfigured(c)
	}
	nss, err := h.k8s.ListNamespaces(c.Context())
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, nss)
}

func (h *Handler) ListPods(c *fiber.Ctx) error {
	if h.k8s == nil {
		return k8sNotConfigured(c)
	}
	pods, err := h.k8s.ListPods(c.Context(), c.Params("ns"))
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, pods)
}

func (h *Handler) ListDeployments(c *fiber.Ctx) error {
	if h.k8s == nil {
		return k8sNotConfigured(c)
	}
	deps, err := h.k8s.ListDeployments(c.Context(), c.Query("namespace", "default"))
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, deps)
}

func (h *Handler) CreateDeployment(c *fiber.Ctx) error {
	if h.k8s == nil {
		return k8sNotConfigured(c)
	}
	var req CreateDeploymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}
	dep, err := h.k8s.CreateDeployment(c.Context(), req)
	if err != nil {
		return internalErr(c, err)
	}
	return response.Created(c, dep)
}

func (h *Handler) ScaleDeployment(c *fiber.Ctx) error {
	if h.k8s == nil {
		return k8sNotConfigured(c)
	}
	var req ScaleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}
	dep, err := h.k8s.ScaleDeployment(c.Context(), c.Params("name"), c.Query("namespace", "default"), req.Replicas)
	if err != nil {
		return internalErr(c, err)
	}
	return response.OK(c, dep)
}

func (h *Handler) DeleteDeployment(c *fiber.Ctx) error {
	if h.k8s == nil {
		return k8sNotConfigured(c)
	}
	if err := h.k8s.DeleteDeployment(c.Context(), c.Params("name"), c.Query("namespace", "default")); err != nil {
		return internalErr(c, err)
	}
	return response.NoContent(c)
}
