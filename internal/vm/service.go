package vm

import (
	"context"
	"errors"
	"fmt"
	"time"

	proxmox "github.com/luthermonson/go-proxmox"
)

// ErrNotConfigured is returned when Proxmox client is nil (not configured).
var ErrNotConfigured = errors.New("proxmox yapılandırılmamış — config.yaml dosyasını doldurun")

// Service handles VM business logic using the Proxmox API.
type Service struct {
	px *ProxmoxClient
}

// NewService creates a new VM service. px may be nil if Proxmox is not configured.
func NewService(px *ProxmoxClient) *Service {
	return &Service{px: px}
}

func (s *Service) checkReady() error {
	if s.px == nil {
		return ErrNotConfigured
	}
	return nil
}

// List returns all VMs on the configured node.
func (s *Service) List(ctx context.Context) ([]VM, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	node, err := s.px.Client().Node(ctx, s.px.Node())
	if err != nil {
		return nil, fmt.Errorf("node alınamadı: %w", err)
	}

	vms, err := node.VirtualMachines(ctx)
	if err != nil {
		return nil, fmt.Errorf("VM listesi alınamadı: %w", err)
	}

	result := make([]VM, 0, len(vms))
	for _, v := range vms {
		result = append(result, mapVM(v, s.px.Node()))
	}
	return result, nil
}

// Get returns a single VM by its numeric ID.
func (s *Service) Get(ctx context.Context, vmID int) (*VM, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	node, err := s.px.Client().Node(ctx, s.px.Node())
	if err != nil {
		return nil, err
	}

	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("VM bulunamadı (id=%d): %w", vmID, err)
	}

	mapped := mapVM(vm, s.px.Node())
	return &mapped, nil
}

// Create clones a template and optionally starts the new VM.
func (s *Service) Create(ctx context.Context, req CreateVMRequest) (*VM, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	node, err := s.px.Client().Node(ctx, s.px.Node())
	if err != nil {
		return nil, err
	}

	// Template'i al ve clone et
	// Clone signature: (ctx, params) → (newid int, task *Task, err error)
	tmpl, err := node.VirtualMachine(ctx, req.Template)
	if err != nil {
		return nil, fmt.Errorf("template bulunamadı (id=%d): %w", req.Template, err)
	}

	newID, task, err := tmpl.Clone(ctx, &proxmox.VirtualMachineCloneOptions{
		Name: req.Name,
		Full: 1, // full clone (int, 0=linked, 1=full)
	})
	if err != nil {
		return nil, fmt.Errorf("clone başarısız: %w", err)
	}

	// Task bitmesini bekle (max 5 dakika)
	if err := task.WaitFor(ctx, 300); err != nil {
		return nil, fmt.Errorf("clone task zaman aşımı: %w", err)
	}

	newVM, err := node.VirtualMachine(ctx, newID)
	if err != nil {
		return nil, err
	}

	// CPU/RAM güncelle — Config returns (*Task, error)
	cfgTask, err := newVM.Config(ctx,
		proxmox.VirtualMachineOption{Name: "cores", Value: req.CPUs},
		proxmox.VirtualMachineOption{Name: "memory", Value: req.MemoryMB},
	)
	if err != nil {
		return nil, fmt.Errorf("VM config güncellenemedi: %w", err)
	}
	// Config task'ı da bekle
	if cfgTask != nil {
		_ = cfgTask.WaitFor(ctx, 30)
	}

	if req.StartOnCreate {
		startTask, err := newVM.Start(ctx)
		if err != nil {
			return nil, fmt.Errorf("VM başlatılamadı: %w", err)
		}
		_ = startTask.WaitFor(ctx, 30)
	}

	result := mapVM(newVM, s.px.Node())
	return &result, nil
}

// Start powers on a VM.
func (s *Service) Start(ctx context.Context, vmID int) (*ActionResponse, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	node, _ := s.px.Client().Node(ctx, s.px.Node())
	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		return nil, err
	}
	task, err := vm.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("VM başlatılamadı: %w", err)
	}
	return &ActionResponse{VMID: vmID, Action: "start", TaskID: string(task.UPID)}, nil
}

// Stop shuts down a VM (ACPI signal).
func (s *Service) Stop(ctx context.Context, vmID int) (*ActionResponse, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	node, _ := s.px.Client().Node(ctx, s.px.Node())
	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		return nil, err
	}
	task, err := vm.Shutdown(ctx)
	if err != nil {
		return nil, fmt.Errorf("VM durdurulamadı: %w", err)
	}
	return &ActionResponse{VMID: vmID, Action: "stop", TaskID: string(task.UPID)}, nil
}

// Reboot restarts a VM.
func (s *Service) Reboot(ctx context.Context, vmID int) (*ActionResponse, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	node, _ := s.px.Client().Node(ctx, s.px.Node())
	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		return nil, err
	}
	task, err := vm.Reboot(ctx)
	if err != nil {
		return nil, fmt.Errorf("VM yeniden başlatılamadı: %w", err)
	}
	return &ActionResponse{VMID: vmID, Action: "reboot", TaskID: string(task.UPID)}, nil
}

// Delete removes a VM permanently.
func (s *Service) Delete(ctx context.Context, vmID int) error {
	if err := s.checkReady(); err != nil {
		return err
	}
	node, _ := s.px.Client().Node(ctx, s.px.Node())
	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		return err
	}
	// Önce durdur
	if stopTask, stopErr := vm.Stop(ctx); stopErr == nil && stopTask != nil {
		_ = stopTask.WaitFor(ctx, 30)
	}
	task, err := vm.Delete(ctx)
	if err != nil {
		return fmt.Errorf("VM silinemedi: %w", err)
	}
	return task.WaitFor(ctx, 60)
}

// Metrics returns the current resource usage of a VM.
func (s *Service) Metrics(ctx context.Context, vmID int) (*VMMetrics, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	node, _ := s.px.Client().Node(ctx, s.px.Node())
	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		return nil, err
	}

	return &VMMetrics{
		VMID:         vmID,
		Timestamp:    time.Now(),
		CPUUsage:     vm.CPU * 100,
		MemUsedMB:    int64(vm.Mem) / 1024 / 1024,
		MemTotalMB:   int64(vm.MaxMem) / 1024 / 1024,
		DiskReadBps:  int64(vm.DiskRead),
		DiskWriteBps: int64(vm.DiskWrite),
		NetInBps:     int64(vm.NetIn),
		NetOutBps:    int64(vm.Netout), // correct casing from Proxmox struct
	}, nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func mapVM(v *proxmox.VirtualMachine, node string) VM {
	return VM{
		ID:       int(uint64(v.VMID)),
		Name:     v.Name,
		Node:     node,
		Status:   VMStatus(v.Status),
		CPUs:     v.CPUs,
		MemoryMB: int64(v.MaxMem) / 1024 / 1024,
		DiskGB:   float64(v.MaxDisk) / 1024 / 1024 / 1024,
		Uptime:   int64(v.Uptime),
	}
}
