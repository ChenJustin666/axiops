package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// TestCommandRunner is kept behind an interface so workflow tests can use a
// deterministic fake and production deployments can replace shell execution
// with a sandboxed runner or a build provider later.
type TestCommandRunner interface {
	Run(context.Context, string, string) (string, error)
}

// ShellTestRunner is only for explicit local tooling. The production Worker
// registers IsolatedTestExecutor and must never run repository code locally.
type ShellTestRunner struct{}

type testOutputTail struct {
	mu   sync.Mutex
	data []byte
}

func (b *testOutputTail) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	const limit = 1 << 20
	if len(p) >= limit {
		b.data = append(b.data[:0], p[len(p)-limit:]...)
	} else {
		if len(b.data)+len(p) > limit {
			b.data = append([]byte(nil), b.data[len(b.data)+len(p)-limit:]...)
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}

func NewShellTestRunner() *ShellTestRunner { return &ShellTestRunner{} }

func (r *ShellTestRunner) Run(ctx context.Context, command, workingDir string) (string, error) {
	if r == nil || strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("test command is required")
	}
	var process *exec.Cmd
	if runtime.GOOS == "windows" {
		process = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		process = exec.CommandContext(ctx, "sh", "-euc", command)
	}
	if strings.TrimSpace(workingDir) != "" {
		process.Dir = workingDir
	}
	output := &testOutputTail{}
	process.Stdout, process.Stderr = output, output
	err := process.Run()
	return string(output.data), err
}

type testNodeConfig struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir,omitempty"`
}

type TestExecutor struct{ runner TestCommandRunner }

func NewTestExecutor(runner TestCommandRunner) *TestExecutor { return &TestExecutor{runner: runner} }

func (e *TestExecutor) NodeType() string { return NodeTest }

func (e *TestExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e == nil || e.runner == nil {
		return ExecutionResult{}, fmt.Errorf("test runner is not configured")
	}
	var config testNodeConfig
	if len(request.Task.NodeSnapshot) == 0 {
		return ExecutionResult{}, fmt.Errorf("invalid test node snapshot")
	}
	var node struct {
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(request.Task.NodeSnapshot, &node); err != nil || json.Unmarshal(node.Config, &config) != nil {
		return ExecutionResult{}, fmt.Errorf("invalid test node configuration")
	}
	if strings.TrimSpace(config.Command) == "" || len(config.Command) > 16<<10 {
		return ExecutionResult{}, fmt.Errorf("test command must be between 1 and 16384 characters")
	}
	output, err := e.runner.Run(ctx, config.Command, config.WorkingDir)
	if err != nil {
		message := strings.TrimSpace(output)
		if message == "" {
			message = err.Error()
		} else if len(message) > 4096 {
			message = message[len(message)-4096:]
		}
		return ExecutionResult{LogText: output}, fmt.Errorf("test failed: %s", message)
	}
	return ExecutionResult{Status: RunSucceeded, LogText: output}, nil
}
