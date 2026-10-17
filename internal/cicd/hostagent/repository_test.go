package hostagent

import "testing"

func rolloutCommands(agentIDs ...int64) []DeploymentCommand {
	commands := make([]DeploymentCommand, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		commands = append(commands, DeploymentCommand{
			WorkflowRunID: 10, TaskKey: "deploy", ProjectID: 20,
			ServiceID: 30, ServiceRevisionID: 40, AgentID: agentID,
		})
	}
	return commands
}

func TestPrepareRolloutOnlyReleasesFirstBatch(t *testing.T) {
	prepared, targetHash, batches, err := prepareRollout(rolloutCommands(9, 2, 5, 7, 3), RolloutPolicy{BatchSize: 2, IntervalSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(targetHash) != 64 || batches != 3 {
		t.Fatalf("hash=%q batches=%d", targetHash, batches)
	}
	for index, item := range prepared {
		if index > 0 && item.Command.AgentID <= prepared[index-1].Command.AgentID {
			t.Fatalf("targets are not deterministic: %+v", prepared)
		}
		expectedBatch := index/2 + 1
		expectedStatus := StatusBlocked
		if expectedBatch == 1 {
			expectedStatus = StatusPending
		}
		if item.BatchNumber != expectedBatch || item.Status != expectedStatus {
			t.Fatalf("unexpected item %d: %+v", index, item)
		}
	}
}

func TestPrepareRolloutRejectsChangedIdentityAndDuplicateAgent(t *testing.T) {
	commands := rolloutCommands(1, 2)
	commands[1].ProjectID = 99
	if _, _, _, err := prepareRollout(commands, RolloutPolicy{BatchSize: 1}); err == nil {
		t.Fatal("mixed project rollout was accepted")
	}
	if _, _, _, err := prepareRollout(rolloutCommands(1, 1), RolloutPolicy{BatchSize: 1}); err == nil {
		t.Fatal("duplicate Agent rollout was accepted")
	}
}

func TestRolloutTransitionSuccessFailureAndCompletion(t *testing.T) {
	tests := []struct {
		name                                         string
		status                                       string
		commandBatch, current, remaining, next, want int
	}{
		{name: "wait for current batch", status: StatusSuccess, commandBatch: 1, current: 1, remaining: 1, next: 2, want: rolloutWait},
		{name: "release next batch", status: StatusSuccess, commandBatch: 1, current: 1, next: 2, want: rolloutRelease},
		{name: "complete final batch", status: StatusSuccess, commandBatch: 2, current: 2, want: rolloutComplete},
		{name: "failure stops rollout", status: StatusFailed, commandBatch: 1, current: 1, next: 2, want: rolloutFail},
		{name: "rollback stops rollout", status: StatusRolledBack, commandBatch: 1, current: 1, next: 2, want: rolloutFail},
		{name: "stale batch cannot release", status: StatusSuccess, commandBatch: 1, current: 2, next: 3, want: rolloutWait},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := decideRolloutTransition(test.status, test.commandBatch, test.current, test.remaining, test.next); got != test.want {
				t.Fatalf("transition=%d want=%d", got, test.want)
			}
		})
	}
}
