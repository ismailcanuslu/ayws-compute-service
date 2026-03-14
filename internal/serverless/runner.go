package serverless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
)

// runtimeImages maps Runtime to the Docker image to use as sandbox.
var runtimeImages = map[Runtime]string{
	RuntimePython: "python:3.11-slim",
	RuntimeNode:   "node:20-slim",
	RuntimeGo:     "golang:1.22-alpine",
	RuntimeBash:   "alpine:3.19",
}

// runtimeEntrypoint returns the command that executes the function code.
func runtimeEntrypoint(rt Runtime, code string) []string {
	switch rt {
	case RuntimePython:
		return []string{"python3", "-c", code}
	case RuntimeNode:
		return []string{"node", "-e", code}
	case RuntimeGo:
		// Geçici main wrapper
		wrapped := fmt.Sprintf(`package main
import "fmt"
func main() { %s }`, code)
		return []string{"sh", "-c", fmt.Sprintf(`echo '%s' > /tmp/main.go && go run /tmp/main.go`, wrapped)}
	default: // bash
		return []string{"sh", "-c", code}
	}
}

// Runner executes functions in isolated Docker containers.
type Runner struct {
	docker        *client.Client
	defaultMemory int64 // bytes
	defaultTimeout time.Duration
}

// NewRunner creates a Docker-backed sandbox runner.
func NewRunner(docker *client.Client, memMB int, timeoutSec int) *Runner {
	return &Runner{
		docker:        docker,
		defaultMemory: int64(memMB) * units.MiB,
		defaultTimeout: time.Duration(timeoutSec) * time.Second,
	}
}

// Invoke runs the given function in a fresh container and returns stdout/stderr.
func (r *Runner) Invoke(ctx context.Context, fn *Function, payload map[string]any) (*InvokeResult, error) {
	startedAt := time.Now()

	image := runtimeImages[fn.Runtime]
	if image == "" {
		image = runtimeImages[RuntimeBash]
	}

	// Env vars
	env := make([]string, 0, len(fn.EnvVars)+1)
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			env = append(env, "PAYLOAD="+string(b))
		}
	}
	for k, v := range fn.EnvVars {
		env = append(env, k+"="+v)
	}

	memLimit := r.defaultMemory
	if fn.MemoryMB > 0 {
		memLimit = int64(fn.MemoryMB) * units.MiB
	}

	timeout := r.defaultTimeout
	if fn.TimeoutSec > 0 {
		timeout = time.Duration(fn.TimeoutSec) * time.Second
	}

	// Invoke timeout context
	invokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Container oluştur
	resp, err := r.docker.ContainerCreate(invokeCtx, &container.Config{
		Image:      image,
		Cmd:        runtimeEntrypoint(fn.Runtime, fn.Code),
		Env:        env,
		WorkingDir: "/workspace",
	}, &container.HostConfig{
		Resources: container.Resources{
			Memory:   memLimit,
			CPUQuota: 50000, // %50 CPU
		},
		AutoRemove:  false,
		NetworkMode: "none", // izole ağ
	}, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("container oluşturulamadı: %w", err)
	}

	containerID := resp.ID
	defer r.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}) //nolint:errcheck

	// Başlat
	if err := r.docker.ContainerStart(invokeCtx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("container başlatılamadı: %w", err)
	}

	// Bitmesini bekle
	statusCh, errCh := r.docker.ContainerWait(invokeCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int
	select {
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	case err := <-errCh:
		return &InvokeResult{
			FunctionID:  fn.ID,
			StartedAt:   startedAt,
			FinishedAt:  time.Now(),
			DurationMs:  time.Since(startedAt).Milliseconds(),
			ExitCode:    -1,
			Error:       fmt.Sprintf("container bekleme hatası: %v", err),
			ContainerID: containerID,
		}, nil
	}

	// Logları topla
	logOutput := r.collectLogs(ctx, containerID)
	finishedAt := time.Now()

	return &InvokeResult{
		FunctionID:  fn.ID,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		DurationMs:  finishedAt.Sub(startedAt).Milliseconds(),
		ExitCode:    exitCode,
		Output:      logOutput,
		ContainerID: containerID,
	}, nil
}

func (r *Runner) collectLogs(ctx context.Context, containerID string) string {
	out, err := r.docker.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return ""
	}
	defer out.Close()

	var buf bytes.Buffer
	io.Copy(&buf, out) //nolint:errcheck

	// Docker log stream'i 8-byte header içeriyor — strip et
	raw := buf.String()
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		if len(line) > 8 {
			lines = append(lines, line[8:])
		}
	}
	return strings.Join(lines, "\n")
}
