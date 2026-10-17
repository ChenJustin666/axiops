package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func notifyNodeSnapshot(message string) []byte {
	config, _ := json.Marshal(map[string]string{"message": message})
	snapshot, _ := json.Marshal(Node{Type: NodeNotify, Config: config})
	return snapshot
}

func TestNotifyExecutorCompletesForTerminalOutboxDelivery(t *testing.T) {
	result, err := NewNotifyExecutor().Execute(context.Background(), ExecutionRequest{
		Task: TaskRun{NodeSnapshot: notifyNodeSnapshot("发布结果通知")},
	})
	if err != nil || result.Status != RunSucceeded {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestNotifyExecutorRejectsOversizedMessage(t *testing.T) {
	_, err := NewNotifyExecutor().Execute(context.Background(), ExecutionRequest{
		Task: TaskRun{NodeSnapshot: notifyNodeSnapshot(strings.Repeat("a", 1025))},
	})
	if err == nil {
		t.Fatal("oversized notify message was accepted")
	}
}
