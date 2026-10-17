package workflow

import "encoding/json"

func deployStartOrder(task TaskRun) int {
	if task.NodeType != NodeYAMLDeploy && task.NodeType != NodeHelmDeploy && task.NodeType != NodeHostDeploy {
		return 0
	}
	var node struct {
		Config struct {
			StartOrder int `json:"start_order"`
		} `json:"config"`
	}
	if json.Unmarshal(task.NodeSnapshot, &node) != nil {
		return 0
	}
	return node.Config.StartOrder
}

func hasOrderedDeployments(tasks []TaskRun) bool {
	count := 0
	for _, task := range tasks {
		if deployStartOrder(task) > 0 {
			count++
		}
	}
	return count > 1
}
