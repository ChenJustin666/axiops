package workflow

import "fmt"

type SchedulerPlan struct {
	Runnable  []string `json:"runnable"`
	Skippable []string `json:"skippable"`
	Waiting   []string `json:"waiting"`
	Complete  bool     `json:"complete"`
}

// Plan computes the next safe scheduler decision without mutating the run.
// Providers are invoked only for Runnable nodes; all other states remain
// durable and can be recomputed after a worker restart.
func Plan(run WorkflowRun) (SchedulerPlan, error) {
	plan := SchedulerPlan{}
	if run.Status == RunPendingApproval || run.Status == RunCancelling || run.Status == RunCancelled || run.Status == RunFailed || run.Status == RunSucceeded {
		return plan, nil
	}
	if run.Status != RunPending && run.Status != RunRunning {
		return plan, fmt.Errorf("unsupported workflow run status %q", run.Status)
	}
	states := make(map[string]string, len(run.Tasks))
	ordered := hasOrderedDeployments(run.Tasks)
	for _, task := range run.Tasks {
		if task.NodeKey == "" {
			return plan, fmt.Errorf("workflow task has empty node key")
		}
		if _, exists := states[task.NodeKey]; exists {
			return plan, fmt.Errorf("duplicate workflow task %q", task.NodeKey)
		}
		states[task.NodeKey] = task.Status
		if ordered && task.Status == "skipped" {
			states[task.NodeKey] = RunFailed
		}
	}
	for _, task := range run.Tasks {
		switch task.Status {
		case RunSucceeded, "skipped":
			continue
		case RunFailed, RunCancelling, RunCancelled:
			continue
		case RunRunning:
			continue
		case RunPending:
			ready, blocked, invalid := dependenciesReady(task, states)
			if invalid != "" {
				return plan, fmt.Errorf("task %q depends on unknown task %q", task.NodeKey, invalid)
			}
			phaseReady, phaseBlocked, phaseFailed := orderedBuildPhase(task, run.Tasks)
			if phaseFailed {
				plan.Skippable = append(plan.Skippable, task.NodeKey)
				continue
			} else if !phaseReady {
				ready = false
				blocked = phaseBlocked
			}
			if ready && !startOrderReady(task, run.Tasks) {
				blocked = true
				ready = false
			}
			if ready {
				plan.Runnable = append(plan.Runnable, task.NodeKey)
			} else if blocked {
				plan.Waiting = append(plan.Waiting, task.NodeKey)
			} else {
				plan.Skippable = append(plan.Skippable, task.NodeKey)
			}
		default:
			plan.Waiting = append(plan.Waiting, task.NodeKey)
		}
	}
	plan.Complete = len(plan.Runnable) == 0 && len(plan.Waiting) == 0 && len(plan.Skippable) == 0
	return plan, nil
}

// orderedBuildPhase keeps a multi-service ordered release reproducible: all
// build/test nodes finish before any ordered deployment is submitted. A
// selected-service run contains only one service and therefore has no phase
// barrier.
func orderedBuildPhase(task TaskRun, tasks []TaskRun) (ready, blocked, failed bool) {
	if deployStartOrder(task) <= 0 || !hasOrderedDeployments(tasks) {
		return true, false, false
	}
	for _, prior := range tasks {
		if prior.NodeType != NodeBuild && prior.NodeType != NodeTest {
			continue
		}
		switch prior.Status {
		case RunSucceeded:
			continue
		case RunFailed, RunCancelled, RunCancelling, "skipped":
			failed = true
		default:
			blocked = true
		}
	}
	return !blocked && !failed, blocked, failed
}

func startOrderReady(task TaskRun, tasks []TaskRun) bool {
	order := deployStartOrder(task)
	if order <= 1 {
		return true
	}
	for _, prior := range tasks {
		priorOrder := deployStartOrder(prior)
		if priorOrder <= 0 || priorOrder >= order {
			continue
		}
		if prior.ProviderRunID == "" && prior.Status != RunSucceeded && prior.Status != RunFailed && prior.Status != RunCancelled && prior.Status != "skipped" {
			return false
		}
	}
	return true
}

func dependenciesReady(task TaskRun, states map[string]string) (ready, blocked bool, invalid string) {
	for _, dependency := range task.DependsOn {
		state, exists := states[dependency]
		if !exists {
			return false, false, dependency
		}
		switch state {
		case RunSucceeded, "skipped":
			continue
		case RunFailed, RunCancelling, RunCancelled:
			return false, false, ""
		default:
			blocked = true
		}
	}
	return !blocked, blocked, ""
}
