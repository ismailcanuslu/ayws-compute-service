package container

import "time"

// ContainerState holds the running state of a container.
type ContainerState string

const (
	StateRunning  ContainerState = "running"
	StateStopped  ContainerState = "stopped"
	StateExited   ContainerState = "exited"
	StatePaused   ContainerState = "paused"
)

// Container is a normalized Docker container view.
type Container struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	State    ContainerState    `json:"state"`
	Status   string            `json:"status"`
	Ports    []PortMapping     `json:"ports,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
	Created  time.Time         `json:"created"`
}

type PortMapping struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ContainerDetails holds the full inspect output of a container.
type ContainerDetails struct {
	Container
	Command     string            `json:"command,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty"`
	Env         []string          `json:"env,omitempty"`
	Mounts      []MountInfo       `json:"mounts,omitempty"`
	Networks    []NetworkAttach   `json:"networks,omitempty"`
	RestartPolicy string          `json:"restart_policy,omitempty"`
	MemoryLimit int64             `json:"memory_limit_mb"`
	CPUQuota    int64             `json:"cpu_quota"`
	PID         int               `json:"pid,omitempty"`
	StartedAt   string            `json:"started_at,omitempty"`
	FinishedAt  string            `json:"finished_at,omitempty"`
	ExitCode    int               `json:"exit_code,omitempty"`
}

type MountInfo struct {
	Type        string `json:"type"`        // bind | volume | tmpfs
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
}

type NetworkAttach struct {
	Network   string `json:"network"`
	IPAddress string `json:"ip_address,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
	MacAddress string `json:"mac_address,omitempty"`
}

// ContainerStats holds real-time performance metrics.
type ContainerStats struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemUsedMB     float64   `json:"mem_used_mb"`
	MemLimitMB    float64   `json:"mem_limit_mb"`
	MemPercent    float64   `json:"mem_percent"`
	NetRxBytes    int64     `json:"net_rx_bytes"`
	NetTxBytes    int64     `json:"net_tx_bytes"`
	BlockReadBytes int64    `json:"block_read_bytes"`
	BlockWriteBytes int64   `json:"block_write_bytes"`
	PIDs          int       `json:"pids"`
}

// NetworkInfo is a Docker network.
type NetworkInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
	IPAM   string `json:"ipam_subnet,omitempty"`
}

// VolumeInfo is a Docker volume.
type VolumeInfo struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  string            `json:"created_at,omitempty"`
}

// CreateContainerRequest is the payload for POST /api/compute/containers.
type CreateContainerRequest struct {
	Name        string            `json:"name"   validate:"required"`
	Image       string            `json:"image"  validate:"required"`
	// Port mappings: [{"host_port":"8080","container_port":"80","protocol":"tcp"}]
	Ports       []PortMapping     `json:"ports"`
	EnvVars     map[string]string `json:"env_vars"`
	// Volume mounts: [{"source":"my-vol","destination":"/data","type":"volume"}]
	Mounts      []MountSpec       `json:"mounts"`
	Networks    []string          `json:"networks"` // network names to connect
	MemoryMB    int64             `json:"memory_mb"`
	CPUPercent  float64           `json:"cpu_percent"` // e.g. 50.0 = 50%
	Restart     string            `json:"restart"`     // "no" | "always" | "on-failure" | "unless-stopped"
	AutoRemove  bool              `json:"auto_remove"`
	Command     []string          `json:"command,omitempty"`   // override CMD
}

type MountSpec struct {
	Type        string `json:"type"`        // "volume" | "bind" | "tmpfs"
	Source      string `json:"source"`      // volume name or host path
	Destination string `json:"destination"` // container path
	ReadOnly    bool   `json:"read_only"`
}

// ── Kubernetes types ──────────────────────────────────────────────────────────

// K8sDeployment is a normalized Kubernetes Deployment view.
type K8sDeployment struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Image     string            `json:"image"`
	Replicas  int32             `json:"replicas"`
	Ready     int32             `json:"ready"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// CreateDeploymentRequest is the payload for POST /api/compute/k8s/deployments.
type CreateDeploymentRequest struct {
	Name      string            `json:"name"      validate:"required"`
	Namespace string            `json:"namespace"`
	Image     string            `json:"image"     validate:"required"`
	Replicas  int32             `json:"replicas"`
	Port      int32             `json:"port"`
	EnvVars   map[string]string `json:"env_vars"`
	Labels    map[string]string `json:"labels"`
}

// ScaleRequest is used to change replica count.
type ScaleRequest struct {
	Replicas int32 `json:"replicas" validate:"min=0,max=100"`
}

// K8sPod is a simplified pod view.
type K8sPod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Node      string `json:"node,omitempty"`
	IP        string `json:"ip,omitempty"`
	Age       string `json:"age"`
}

// K8sNamespace is a Kubernetes namespace.
type K8sNamespace struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
