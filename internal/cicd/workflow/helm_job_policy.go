package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Ordinary Job resources are not Helm hooks. Recreating an older Job during
// atomic rollback can execute business work a second time.
func helmManifestHasJob(manifest []byte, namespace string) (bool, error) {
	if strings.TrimSpace(string(manifest)) == "" {
		return false, nil
	}
	objects, err := parseHelmManifestResources(manifest, namespace)
	if err != nil {
		return false, err
	}
	for _, object := range objects {
		if object.GetKind() == "Job" {
			return true, nil
		}
	}
	return false, nil
}

func helmAtomicPolicy(next, previous []byte, namespace string) (bool, error) {
	for _, manifest := range [][]byte{next, previous} {
		hasJob, err := helmManifestHasJob(manifest, namespace)
		if err != nil {
			return false, err
		}
		if hasJob {
			return false, nil
		}
	}
	return true, nil
}

func rejectHelmJobRollbackInventory(inventory []byte) error {
	var objects []helmResourceIdentity
	if json.Unmarshal(inventory, &objects) != nil {
		return fmt.Errorf("Helm 历史资源清单无效")
	}
	for _, object := range objects {
		if object.Kind == "Job" {
			return fmt.Errorf("该 Release 包含一次性 Job，回滚可能重复执行任务；请使用明确的新任务发布")
		}
	}
	return nil
}
