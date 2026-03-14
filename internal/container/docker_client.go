package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/rs/zerolog/log"
)

// DockerService manages Docker containers.
type DockerService struct {
	docker *client.Client
}

func NewDockerService(docker *client.Client) *DockerService {
	return &DockerService{docker: docker}
}

// ListContainers returns all containers (running and stopped).
func (s *DockerService) ListContainers(ctx context.Context) ([]Container, error) {
	list, err := s.docker.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("container listesi alınamadı: %w", err)
	}

	result := make([]Container, 0, len(list))
	for _, c := range list {
		result = append(result, mapContainer(c))
	}
	return result, nil
}

// GetContainerDetails returns the full inspect information of a container.
func (s *DockerService) GetContainerDetails(ctx context.Context, idOrName string) (*ContainerDetails, error) {
	info, err := s.docker.ContainerInspect(ctx, idOrName)
	if err != nil {
		return nil, fmt.Errorf("container bulunamadı (%s): %w", idOrName, err)
	}

	// Mounts
	mounts := make([]MountInfo, 0, len(info.Mounts))
	for _, m := range info.Mounts {
		mounts = append(mounts, MountInfo{
			Type:        string(m.Type),
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
			RW:          m.RW,
		})
	}

	// Networks
	nets := make([]NetworkAttach, 0)
	if info.NetworkSettings != nil {
		for netName, ns := range info.NetworkSettings.Networks {
			na := NetworkAttach{Network: netName}
			if ns != nil {
				na.IPAddress = ns.IPAddress
				na.Gateway = ns.Gateway
				na.MacAddress = ns.MacAddress
			}
			nets = append(nets, na)
		}
	}

	// Ports
	ports := make([]PortMapping, 0)
	if info.NetworkSettings != nil {
		for containerPort, bindings := range info.NetworkSettings.Ports {
			for _, b := range bindings {
				parts := strings.Split(string(containerPort), "/")
				proto := "tcp"
				cPort := string(containerPort)
				if len(parts) == 2 {
					cPort = parts[0]
					proto = parts[1]
				}
				ports = append(ports, PortMapping{
					HostIP:        b.HostIP,
					HostPort:      b.HostPort,
					ContainerPort: cPort,
					Protocol:      proto,
				})
			}
		}
	}

	memLimitMB := int64(0)
	if info.HostConfig != nil {
		memLimitMB = info.HostConfig.Memory / 1024 / 1024
	}

	restartPolicy := ""
	cpuQuota := int64(0)
	if info.HostConfig != nil {
		restartPolicy = string(info.HostConfig.RestartPolicy.Name)
		cpuQuota = info.HostConfig.CPUQuota
	}

	cmd := ""
	if info.Config != nil && len(info.Config.Cmd) > 0 {
		cmd = strings.Join(info.Config.Cmd, " ")
	}

	env := []string{}
	entrypoint := []string{}
	if info.Config != nil {
		env = info.Config.Env
		entrypoint = info.Config.Entrypoint
	}

	pid := 0
	startedAt := ""
	finishedAt := ""
	exitCode := 0
	if info.State != nil {
		pid = info.State.Pid
		startedAt = info.State.StartedAt
		finishedAt = info.State.FinishedAt
		exitCode = info.State.ExitCode
	}

	name := strings.TrimPrefix(info.Name, "/")
	created, _ := time.Parse(time.RFC3339Nano, info.Created)

	return &ContainerDetails{
		Container: Container{
			ID:      shortID(info.ID),
			Name:    name,
			Image:   info.Config.Image,
			State:   ContainerState(info.State.Status),
			Status:  info.State.Status,
			Ports:   ports,
			Labels:  info.Config.Labels,
			Created: created,
		},
		Command:       cmd,
		Entrypoint:    entrypoint,
		Env:           env,
		Mounts:        mounts,
		Networks:      nets,
		RestartPolicy: restartPolicy,
		MemoryLimit:   memLimitMB,
		CPUQuota:      cpuQuota,
		PID:           pid,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
		ExitCode:      exitCode,
	}, nil
}

// GetContainerStats returns real-time resource usage (one snapshot).
func (s *DockerService) GetContainerStats(ctx context.Context, idOrName string) (*ContainerStats, error) {
	resp, err := s.docker.ContainerStats(ctx, idOrName, false) // false = one-shot
	if err != nil {
		return nil, fmt.Errorf("istatistikler alınamadı: %w", err)
	}
	defer resp.Body.Close()

	var raw types.StatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("istatistik ayrıştırılamadı: %w", err)
	}

	cpuPercent := calculateCPU(&raw)
	memUsed := float64(raw.MemoryStats.Usage) / 1024 / 1024
	memLimit := float64(raw.MemoryStats.Limit) / 1024 / 1024
	memPct := 0.0
	if memLimit > 0 {
		memPct = (memUsed / memLimit) * 100
	}

	var netRx, netTx int64
	for _, n := range raw.Networks {
		netRx += int64(n.RxBytes)
		netTx += int64(n.TxBytes)
	}

	var blkR, blkW int64
	for _, b := range raw.BlkioStats.IoServicedRecursive {
		switch b.Op {
		case "Read":
			blkR += int64(b.Value)
		case "Write":
			blkW += int64(b.Value)
		}
	}

	return &ContainerStats{
		ID:              shortID(raw.ID),
		Timestamp:       time.Now(),
		CPUPercent:      cpuPercent,
		MemUsedMB:       memUsed,
		MemLimitMB:      memLimit,
		MemPercent:      memPct,
		NetRxBytes:      netRx,
		NetTxBytes:      netTx,
		BlockReadBytes:  blkR,
		BlockWriteBytes: blkW,
		PIDs:            int(raw.PidsStats.Current),
	}, nil
}

func calculateCPU(v *types.StatsJSON) float64 {
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	numCPUs := float64(v.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * numCPUs * 100.0
	}
	return 0
}

// StartContainer starts a stopped/exited container.
func (s *DockerService) StartContainer(ctx context.Context, idOrName string) error {
	if err := s.docker.ContainerStart(ctx, idOrName, container.StartOptions{}); err != nil {
		return fmt.Errorf("container başlatılamadı: %w", err)
	}
	return nil
}

// StopContainer sends SIGTERM and waits for graceful stop (10s timeout).
func (s *DockerService) StopContainer(ctx context.Context, idOrName string) error {
	timeout := 10
	if err := s.docker.ContainerStop(ctx, idOrName, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("container durdurulamadı: %w", err)
	}
	return nil
}

// RemoveContainer removes a container.
func (s *DockerService) RemoveContainer(ctx context.Context, idOrName string, force bool) error {
	if err := s.docker.ContainerRemove(ctx, idOrName, container.RemoveOptions{Force: force}); err != nil {
		return fmt.Errorf("container silinemedi: %w", err)
	}
	return nil
}

// Logs returns the last N lines of a container's stdout+stderr.
func (s *DockerService) Logs(ctx context.Context, idOrName string, tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}
	reader, err := s.docker.ContainerLogs(ctx, idOrName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		return "", fmt.Errorf("loglar alınamadı: %w", err)
	}
	defer reader.Close()

	var sb strings.Builder
	io.Copy(&sb, reader) //nolint:errcheck
	return sb.String(), nil
}

// CreateContainer creates and starts a new container with full options.
func (s *DockerService) CreateContainer(ctx context.Context, req CreateContainerRequest) (*Container, error) {
	// Port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for _, p := range req.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		port := nat.Port(fmt.Sprintf("%s/%s", p.ContainerPort, proto))
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{{
			HostIP:   p.HostIP,
			HostPort: p.HostPort,
		}}
	}

	// Env
	env := make([]string, 0, len(req.EnvVars))
	for k, v := range req.EnvVars {
		env = append(env, k+"="+v)
	}

	// Mounts
	mounts := make([]mount.Mount, 0, len(req.Mounts))
	for _, m := range req.Mounts {
		mt := mount.TypeVolume
		switch m.Type {
		case "bind":
			mt = mount.TypeBind
		case "tmpfs":
			mt = mount.TypeTmpfs
		}
		mounts = append(mounts, mount.Mount{
			Type:     mt,
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: m.ReadOnly,
		})
	}

	// Resource limits
	var memLimit int64
	if req.MemoryMB > 0 {
		memLimit = req.MemoryMB * units.MiB
	}
	var cpuQuota int64
	if req.CPUPercent > 0 {
		cpuQuota = int64(req.CPUPercent * 1000)
	}

	restart := req.Restart
	if restart == "" {
		restart = "no"
	}

	resp, err := s.docker.ContainerCreate(ctx,
		&container.Config{
			Image:        req.Image,
			Env:          env,
			ExposedPorts: exposedPorts,
			Cmd:          req.Command,
		},
		&container.HostConfig{
			PortBindings:  portBindings,
			Mounts:        mounts,
			RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(restart)},
			AutoRemove:    req.AutoRemove,
			Resources: container.Resources{
				Memory:   memLimit,
				CPUQuota: cpuQuota,
			},
		},
		&network.NetworkingConfig{},
		nil,
		req.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("container oluşturulamadı: %w", err)
	}

	// Ek ağlara bağlan
	for _, netName := range req.Networks {
		if err := s.docker.NetworkConnect(ctx, netName, resp.ID, nil); err != nil {
			log.Warn().Err(err).Str("network", netName).Msg("ağa bağlanılamadı")
		}
	}

	if err := s.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("container başlatılamadı: %w", err)
	}

	det, err := s.GetContainerDetails(ctx, resp.ID)
	if err != nil {
		return nil, err
	}
	return &det.Container, nil
}

// ── Networks ──────────────────────────────────────────────────────────────────

// ListNetworks returns all Docker networks.
func (s *DockerService) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	nets, err := s.docker.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ağ listesi alınamadı: %w", err)
	}
	result := make([]NetworkInfo, 0, len(nets))
	for _, n := range nets {
		subnet := ""
		if len(n.IPAM.Config) > 0 {
			subnet = n.IPAM.Config[0].Subnet
		}
		result = append(result, NetworkInfo{
			ID:     n.ID[:12],
			Name:   n.Name,
			Driver: n.Driver,
			Scope:  n.Scope,
			IPAM:   subnet,
		})
	}
	return result, nil
}

// CreateNetwork creates a new Docker network.
func (s *DockerService) CreateNetwork(ctx context.Context, name, driver string) (*NetworkInfo, error) {
	if driver == "" {
		driver = "bridge"
	}
	resp, err := s.docker.NetworkCreate(ctx, name, types.NetworkCreate{Driver: driver})
	if err != nil {
		return nil, fmt.Errorf("ağ oluşturulamadı: %w", err)
	}
	return &NetworkInfo{ID: resp.ID[:12], Name: name, Driver: driver}, nil
}

// ── Volumes ───────────────────────────────────────────────────────────────────

// ListVolumes returns all Docker volumes.
func (s *DockerService) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	resp, err := s.docker.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("volume listesi alınamadı: %w", err)
	}
	result := make([]VolumeInfo, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		result = append(result, VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Labels:     v.Labels,
			CreatedAt:  v.CreatedAt,
		})
	}
	return result, nil
}

// CreateVolume creates a new Docker volume.
func (s *DockerService) CreateVolume(ctx context.Context, name, driver string) (*VolumeInfo, error) {
	if driver == "" {
		driver = "local"
	}
	v, err := s.docker.VolumeCreate(ctx, volume.CreateOptions{Name: name, Driver: driver})
	if err != nil {
		return nil, fmt.Errorf("volume oluşturulamadı: %w", err)
	}
	return &VolumeInfo{Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint}, nil
}

// PullImage ensures a Docker image is present locally.
func (s *DockerService) PullImage(ctx context.Context, img string) error {
	reader, err := s.docker.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	io.Copy(io.Discard, reader) //nolint:errcheck
	return nil
}

// PruneContainers removes all stopped containers.
func (s *DockerService) PruneContainers(ctx context.Context) error {
	_, err := s.docker.ContainersPrune(ctx, filters.Args{})
	return err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mapContainer(c types.Container) Container {
	name := ""
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	ports := make([]PortMapping, 0, len(c.Ports))
	for _, p := range c.Ports {
		ports = append(ports, PortMapping{
			HostIP:        p.IP,
			HostPort:      fmt.Sprintf("%d", p.PublicPort),
			ContainerPort: fmt.Sprintf("%d", p.PrivatePort),
			Protocol:      p.Type,
		})
	}
	return Container{
		ID:      shortID(c.ID),
		Name:    name,
		Image:   c.Image,
		State:   ContainerState(c.State),
		Status:  c.Status,
		Ports:   ports,
		Labels:  c.Labels,
		Created: time.Unix(c.Created, 0),
	}
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
