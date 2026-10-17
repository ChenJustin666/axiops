package workflow

// Traverse dependencies rather than accepting an unrelated parallel build.
func hasServiceBuildDependency(tasks []TaskRun, deploy TaskRun, serviceID int64) bool {
	byKey := make(map[string]TaskRun, len(tasks))
	for _, task := range tasks {
		byKey[task.NodeKey] = task
	}
	queue := append([]string(nil), deploy.DependsOn...)
	seen := map[string]bool{}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if seen[key] {
			continue
		}
		seen[key] = true
		task, ok := byKey[key]
		if !ok {
			continue
		}
		if task.NodeType == NodeBuild {
			cfg, err := buildConfig(task.NodeSnapshot)
			if err == nil && cfg.ServiceID == serviceID {
				return true
			}
		}
		queue = append(queue, task.DependsOn...)
	}
	return false
}
