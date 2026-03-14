package serverless

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrNotConfigured = errors.New("docker yapılandırılmamış ya da servis başlatılamadı")
var ErrFunctionNotFound = errors.New("fonksiyon bulunamadı")

// Store is a simple in-memory function registry.
// Production'da PostgreSQL ile değiştirin.
type store struct {
	mu   sync.RWMutex
	fns  map[string]*Function
	logs map[string][]FunctionLog
}

// Service manages function definitions and invocations.
type Service struct {
	runner *Runner
	db     *store
}

func NewService(runner *Runner) *Service {
	return &Service{
		runner: runner,
		db: &store{
			fns:  make(map[string]*Function),
			logs: make(map[string][]FunctionLog),
		},
	}
}

func (s *Service) checkReady() error {
	if s.runner == nil {
		return ErrNotConfigured
	}
	return nil
}

// List returns all registered functions.
func (s *Service) List(_ context.Context) ([]Function, error) {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	out := make([]Function, 0, len(s.db.fns))
	for _, f := range s.db.fns {
		cp := *f
		cp.Code = "" // kodu liste yanıtında gizle
		out = append(out, cp)
	}
	return out, nil
}

// Get returns a function by ID.
func (s *Service) Get(_ context.Context, id string) (*Function, error) {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	fn, ok := s.db.fns[id]
	if !ok {
		return nil, ErrFunctionNotFound
	}
	cp := *fn
	return &cp, nil
}

// Create registers a new function definition.
func (s *Service) Create(_ context.Context, req CreateFunctionRequest) (*Function, error) {
	if req.MemoryMB == 0 {
		req.MemoryMB = 128
	}
	if req.TimeoutSec == 0 {
		req.TimeoutSec = 30
	}

	fn := &Function{
		ID:         uuid.NewString(),
		Name:       req.Name,
		Runtime:    req.Runtime,
		Code:       req.Code,
		EnvVars:    req.EnvVars,
		MemoryMB:   req.MemoryMB,
		TimeoutSec: req.TimeoutSec,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	s.db.mu.Lock()
	s.db.fns[fn.ID] = fn
	s.db.mu.Unlock()

	return fn, nil
}

// Invoke executes a function and stores its log.
func (s *Service) Invoke(ctx context.Context, id string, payload map[string]any) (*InvokeResult, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	fn, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	result, err := s.runner.Invoke(ctx, fn, payload)
	if err != nil {
		return nil, fmt.Errorf("invoke hatası: %w", err)
	}

	// invoke sayacını artır
	s.db.mu.Lock()
	if stored, ok := s.db.fns[id]; ok {
		stored.InvokeCount++
	}
	// Log tut (son 100)
	logEntry := FunctionLog{
		Timestamp: time.Now(),
		Level:     "stdout",
		Message:   result.Output,
	}
	s.db.logs[id] = append(s.db.logs[id], logEntry)
	if len(s.db.logs[id]) > 100 {
		s.db.logs[id] = s.db.logs[id][len(s.db.logs[id])-100:]
	}
	s.db.mu.Unlock()

	return result, nil
}

// Logs returns recent execution logs for a function.
func (s *Service) Logs(_ context.Context, id string) ([]FunctionLog, error) {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	if _, ok := s.db.fns[id]; !ok {
		return nil, ErrFunctionNotFound
	}
	return s.db.logs[id], nil
}

// Delete removes a function definition.
func (s *Service) Delete(_ context.Context, id string) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	if _, ok := s.db.fns[id]; !ok {
		return ErrFunctionNotFound
	}
	delete(s.db.fns, id)
	delete(s.db.logs, id)
	return nil
}
