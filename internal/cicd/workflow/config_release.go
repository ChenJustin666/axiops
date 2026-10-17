package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	"strings"
)

type CurrentDeployment struct {
	SnapshotID int64             `json:"snapshot_id"`
	RunID      int64             `json:"run_id"`
	ClusterID  int64             `json:"cluster_id"`
	Namespace  string            `json:"namespace"`
	Images     map[string]string `json:"images"`
}

func LoadCurrentDeployment(ctx context.Context, q sourceQuery, projectID, eid, sid int64) (*CurrentDeployment, error) {
	var s DeploymentSnapshot
	err := q.QueryRowContext(ctx, `SELECT id,workflow_run_id,cluster_id,namespace,rendered_manifest,manifest_digest,status FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? AND provider=? ORDER BY id DESC LIMIT 1`, projectID, eid, sid, NodeYAMLDeploy).Scan(&s.ID, &s.WorkflowRunID, &s.ClusterID, &s.Namespace, &s.RenderedManifest, &s.ManifestDigest, &s.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// A newer partial/failed apply must never fall back to an older success.
	if s.Status != "applied" || !s.VerifyManifestDigest() {
		return nil, nil
	}
	resources, err := ValidateYAMLManifest(s.RenderedManifest, s.Namespace)
	if err != nil {
		return nil, err
	}
	images := map[string]string{}
	for _, item := range resources {
		spec := workloadPodSpec(item)
		for _, group := range []string{"containers", "initContainers"} {
			containers, _, _ := unstructured.NestedSlice(item.Object, append(spec, group)...)
			for _, raw := range containers {
				c, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := c["name"].(string)
				img, _ := c["image"].(string)
				if name != "" {
					images[item.GetKind()+"/"+item.GetName()+"/"+group+"/"+name] = img
				}
			}
		}
	}
	if len(images) == 0 {
		return nil, nil
	}
	for _, img := range images {
		parts := strings.Split(img, "@")
		if len(parts) != 2 || !promotionDigest.MatchString(parts[1]) {
			return nil, nil
		}
	}
	return &CurrentDeployment{SnapshotID: s.ID, RunID: s.WorkflowRunID, ClusterID: s.ClusterID, Namespace: s.Namespace, Images: images}, nil
}

func workloadPodSpec(item unstructured.Unstructured) []string {
	if item.GetKind() == "Pod" {
		return []string{"spec"}
	}
	if item.GetKind() == "CronJob" {
		return []string{"spec", "jobTemplate", "spec", "template", "spec"}
	}
	return []string{"spec", "template", "spec"}
}

func DeploymentConfigDigest(source *YAMLSource) string {
	if source == nil {
		return ""
	}
	copy := *source
	copy.PreservedImages = nil
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func preserveDeploymentImages(manifest []byte, namespace string, images map[string]string) ([]byte, error) {
	if len(images) == 0 {
		return manifest, nil
	}
	objects, err := ValidateYAMLManifest(manifest, namespace)
	if err != nil {
		return nil, err
	}
	var output []string
	for _, item := range objects {
		spec := workloadPodSpec(item)
		for _, group := range []string{"containers", "initContainers"} {
			containers, found, e := unstructured.NestedSlice(item.Object, append(spec, group)...)
			if e != nil {
				return nil, e
			}
			if !found {
				continue
			}
			for _, raw := range containers {
				c, ok := raw.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid container")
				}
				name, _ := c["name"].(string)
				img, ok := images[item.GetKind()+"/"+item.GetName()+"/"+group+"/"+name]
				if !ok {
					return nil, fmt.Errorf("仅配置发布无法确定新增或改名容器 %s/%s/%s 的原镜像，请使用明确选择镜像的发布", item.GetKind(), item.GetName(), name)
				}
				c["image"] = img
			}
			if err = unstructured.SetNestedSlice(item.Object, containers, append(spec, group)...); err != nil {
				return nil, err
			}
		}
		raw, e := yaml.Marshal(item.Object)
		if e != nil {
			return nil, e
		}
		output = append(output, string(raw))
	}
	return []byte(strings.Join(output, "\n---\n")), nil
}

func (r *Repository) LoadConfigurationDeployment(ctx context.Context, projectID, eid, sid int64) (*CurrentDeployment, error) {
	var active int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_run WHERE project_id=? AND environment_id=? AND status IN ('pending','running','pending_approval')`, projectID, eid).Scan(&active); err != nil {
		return nil, err
	}
	if active > 0 {
		return nil, fmt.Errorf("当前环境已有运行中的发布，请结束后重新预览")
	}
	return LoadCurrentDeployment(ctx, r.db, projectID, eid, sid)
}

func configurationOnlyNodes(nodes []Node, sid int64, current *CurrentDeployment, digest string) ([]Node, error) {
	if current == nil || len(current.Images) == 0 {
		return nil, fmt.Errorf("当前环境没有可追溯的成功部署及全部容器镜像摘要，请先明确选择镜像发布")
	}
	var image string
	for _, value := range current.Images {
		image = value
		break
	}
	parts := strings.SplitN(image, "@", 2)
	if len(parts) != 2 || !promotionDigest.MatchString(parts[1]) {
		return nil, fmt.Errorf("当前镜像摘要无效")
	}
	result, err := promoteYAMLNodes(nodes, sid, &PromotionBuild{ImageURL: parts[0], ImageDigest: parts[1]})
	if err != nil {
		return nil, err
	}
	for i := range result {
		if result[i].Type != NodeYAMLDeploy {
			continue
		}
		var before YAMLDeployConfig
		for _, n := range nodes {
			if n.Key == result[i].Key {
				if err = json.Unmarshal(n.Config, &before); err != nil {
					return nil, err
				}
				break
			}
		}
		if before.FrozenYAML == nil || DeploymentConfigDigest(before.FrozenYAML) != digest {
			return nil, fmt.Errorf("配置已在预览后变化，请重新预览再发布")
		}
		if before.FrozenYAML.ClusterID != current.ClusterID || before.FrozenYAML.Namespace != current.Namespace {
			return nil, fmt.Errorf("当前环境目标与上次部署不同，请明确选择镜像重新发布")
		}
		var c YAMLDeployConfig
		if err = json.Unmarshal(result[i].Config, &c); err != nil {
			return nil, err
		}
		c.FrozenYAML.PreservedImages = current.Images
		result[i].Config, err = json.Marshal(c)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
