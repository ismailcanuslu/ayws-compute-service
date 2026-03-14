// Package serverless provides a Docker-based sandbox runner for user functions.
package serverless

import "time"

// Runtime defines the supported execution environments.
type Runtime string

const (
	RuntimePython     Runtime = "python3.11"
	RuntimeNode       Runtime = "node20"
	RuntimeGo         Runtime = "go1.22"
	RuntimeBash       Runtime = "bash"
)

// Function is a stored serverless function definition.
type Function struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Runtime     Runtime           `json:"runtime"`
	Code        string            `json:"code,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	MemoryMB    int               `json:"memory_mb"`
	TimeoutSec  int               `json:"timeout_seconds"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	InvokeCount int64             `json:"invoke_count"`
}

// InvokeRequest is the body for POST /api/compute/functions/:id/invoke.
type InvokeRequest struct {
	Payload map[string]any `json:"payload"`
}

// InvokeResult is returned after a function execution.
type InvokeResult struct {
	FunctionID  string         `json:"function_id"`
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  time.Time      `json:"finished_at"`
	DurationMs  int64          `json:"duration_ms"`
	ExitCode    int            `json:"exit_code"`
	Output      string         `json:"output"`
	Error       string         `json:"error,omitempty"`
	ContainerID string         `json:"container_id,omitempty"`
}

// CreateFunctionRequest is the payload for POST /api/compute/functions.
type CreateFunctionRequest struct {
	Name       string            `json:"name"        validate:"required,min=1,max=64"`
	Runtime    Runtime           `json:"runtime"     validate:"required"`
	Code       string            `json:"code"        validate:"required"`
	EnvVars    map[string]string `json:"env_vars"`
	MemoryMB   int               `json:"memory_mb"`
	TimeoutSec int               `json:"timeout_seconds"`
}

// FunctionLog is a single execution log entry.
type FunctionLog struct {
	Timestamp  time.Time `json:"timestamp"`
	Level      string    `json:"level"` // "stdout" | "stderr"
	Message    string    `json:"message"`
}
