package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
)

func lockNodeServices(ctx context.Context, tx *sql.Tx, projectID int64, nodes []Node) error {
	configs := make([]json.RawMessage, 0, len(nodes))
	for _, node := range nodes {
		configs = append(configs, node.Config)
	}
	ids, err := servicestate.IDs(configs...)
	if err != nil {
		return err
	}
	return servicestate.LockActive(ctx, tx, projectID, ids...)
}
func lockTaskServices(ctx context.Context, tx *sql.Tx, projectID int64, tasks []TaskRun) error {
	nodes := make([]Node, 0, len(tasks))
	for _, task := range tasks {
		var node Node
		if err := json.Unmarshal(task.NodeSnapshot, &node); err != nil {
			return err
		}
		nodes = append(nodes, node)
	}
	return lockNodeServices(ctx, tx, projectID, nodes)
}
