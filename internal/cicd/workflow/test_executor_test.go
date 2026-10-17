package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeTestRunner struct {
	output  string
	err     error
	command string
}

func (r *fakeTestRunner) Run(_ context.Context, command, _ string) (string, error) {
	r.command = command
	return r.output, r.err
}

func testNodeSnapshot(command string) []byte {
	data, _ := json.Marshal(Node{Type: NodeTest, Config: json.RawMessage(`{"command":"` + command + `"}`)})
	return data
}

func TestTestExecutorRunsConfiguredCommand(t *testing.T) {
	runner := &fakeTestRunner{output: "ok"}
	executor := NewTestExecutor(runner)
	result, err := executor.Execute(context.Background(), ExecutionRequest{Task: TaskRun{NodeSnapshot: testNodeSnapshot("go test ./...")}})
	if err != nil || result.Status != RunSucceeded || result.LogText != "ok" || runner.command != "go test ./..." {
		t.Fatalf("result=%+v err=%v command=%q", result, err, runner.command)
	}
}

func TestTestExecutorSurfacesFailure(t *testing.T) {
	runner := &fakeTestRunner{output: "assertion failed", err: errors.New("exit status 1")}
	_, err := NewTestExecutor(runner).Execute(context.Background(), ExecutionRequest{Task: TaskRun{NodeSnapshot: testNodeSnapshot("go test ./...")}})
	if err == nil || err.Error() != "test failed: assertion failed" {
		t.Fatalf("err=%v", err)
	}
}

func TestTestExecutorRejectsMissingCommand(t *testing.T) {
	runner := &fakeTestRunner{}
	_, err := NewTestExecutor(runner).Execute(context.Background(), ExecutionRequest{Task: TaskRun{NodeSnapshot: testNodeSnapshot("")}})
	if err == nil {
		t.Fatal("missing test command was accepted")
	}
}
