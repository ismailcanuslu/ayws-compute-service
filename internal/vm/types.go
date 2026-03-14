// Package vm provides Proxmox VM lifecycle management.
package vm

import "time"

// VMStatus represents the current state of a virtual machine.
type VMStatus string

const (
	StatusRunning  VMStatus = "running"
	StatusStopped  VMStatus = "stopped"
	StatusPaused   VMStatus = "paused"
	StatusUnknown  VMStatus = "unknown"
)

// VM is the normalized representation of a Proxmox QEMU VM.
type VM struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Node      string   `json:"node"`
	Status    VMStatus `json:"status"`
	CPUs      int      `json:"cpus"`
	MemoryMB  int64    `json:"memory_mb"`
	DiskGB    float64  `json:"disk_gb"`
	IPAddress string   `json:"ip_address,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Uptime    int64    `json:"uptime_seconds,omitempty"`
}

// VMMetrics holds real-time CPU/RAM/Disk statistics.
type VMMetrics struct {
	VMID      int       `json:"vm_id"`
	Timestamp time.Time `json:"timestamp"`
	CPUUsage  float64   `json:"cpu_usage_percent"`
	MemUsedMB int64     `json:"mem_used_mb"`
	MemTotalMB int64    `json:"mem_total_mb"`
	DiskReadBps  int64  `json:"disk_read_bps"`
	DiskWriteBps int64  `json:"disk_write_bps"`
	NetInBps  int64     `json:"net_in_bps"`
	NetOutBps int64     `json:"net_out_bps"`
}

// CreateVMRequest is the payload for POST /api/compute/vms.
type CreateVMRequest struct {
	Name      string   `json:"name"      validate:"required,min=1,max=64"`
	Template  int      `json:"template"  validate:"required"` // Proxmox template VM ID
	CPUs      int      `json:"cpus"      validate:"required,min=1,max=32"`
	MemoryMB  int64    `json:"memory_mb" validate:"required,min=512"`
	DiskGB    float64  `json:"disk_gb"   validate:"required,min=1"`
	Tags      []string `json:"tags"`
	StartOnCreate bool `json:"start_on_create"`
}

// ActionResponse is returned after power operations (start/stop/reboot).
type ActionResponse struct {
	VMID   int    `json:"vm_id"`
	Action string `json:"action"`
	TaskID string `json:"task_id,omitempty"` // Proxmox task UPID
}
