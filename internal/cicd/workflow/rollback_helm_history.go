package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

func validateHelmRollbackHistory(request HelmRollbackRequest, manifest []byte) error {
	inventory, err := helmManifestInventory(manifest, request.Namespace)
	if err != nil {
		return fmt.Errorf("Helm 历史资源清单校验失败")
	}
	digest := sha256.Sum256(inventory)
	if request.ExpectedInventoryDigest != "" && hex.EncodeToString(digest[:]) != request.ExpectedInventoryDigest {
		return fmt.Errorf("Helm 历史资源与部署快照不一致，已停止回滚")
	}
	objects, err := parseHelmManifestResources(manifest, request.Namespace)
	if err != nil {
		return err
	}
	count := 0
	for _, o := range objects {
		if o.GetKind()=="Job"{return fmt.Errorf("历史 Release 包含一次性 Job，禁止通过回滚重复执行任务")}
		for _, group := range []string{"containers", "initContainers"} {
			items, _, _ := unstructured.NestedSlice(o.Object, append(workloadPodSpec(o), group)...)
			for _, raw := range items {
				c, _ := raw.(map[string]interface{})
				image, _ := c["image"].(string)
				parts := strings.Split(image, "@")
				if len(parts) != 2 || !promotionDigest.MatchString(parts[1]) {
					return fmt.Errorf("Helm 历史版本包含可变镜像标签，禁止作为确定版本回滚")
				}
				count++
			}
		}
	}
	if count == 0 {
		return fmt.Errorf("Helm 历史版本没有可校验的容器镜像")
	}
	return nil
}
