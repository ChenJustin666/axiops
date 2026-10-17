package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type PromotionBuild struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	ServiceID   int64  `json:"service_id"`
	BuildNumber int64  `json:"build_number"`
	Status      string `json:"status"`
	ImageURL    string `json:"image_url"`
	ImageDigest string `json:"image_digest"`
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
}

func (r *Repository) LoadPromotionBuild(ctx context.Context, id int64) (*PromotionBuild, error) {
	var b PromotionBuild
	err := r.db.QueryRowContext(ctx, `SELECT id,project_id,service_id,build_number,status,COALESCE(image_url,''),COALESCE(image_digest,''),COALESCE(git_commit,''),COALESCE(git_branch,'') FROM build_run WHERE id=?`, id).
		Scan(&b.ID, &b.ProjectID, &b.ServiceID, &b.BuildNumber, &b.Status, &b.ImageURL, &b.ImageDigest, &b.Commit, &b.Branch)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

var promotionDigest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var promotionImage = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*(?::[0-9]+)?/[a-z0-9_./-]+(?::[A-Za-z0-9_.-]+)?$`)

func validPromotionImage(b *PromotionBuild) bool {
	return b != nil && promotionDigest.MatchString(b.ImageDigest) && promotionImage.MatchString(b.ImageURL) && !strings.Contains(b.ImageURL, "..")
}

// Keep target-environment YAML/variables/assets. Only the immutable image comes
// from the selected successful build. Never copy source-environment settings.
func promoteYAMLNodes(nodes []Node, serviceID int64, b *PromotionBuild) ([]Node, error) {
	removed := map[string]Node{}
	byKey := map[string]Node{}
	for _, n := range nodes {
		byKey[n.Key] = n
		if n.Type == NodeBuild {
			removed[n.Key] = n
		}
		if n.Type == NodeTest || n.Type == NodeHostDeploy {
			return nil, fmt.Errorf("此编排不支持镜像复用，请使用构建 + YAML/Helm 部署工作流；不会跳过测试或其他类型的部署")
		}
	}
	var expand func(string, map[string]bool) ([]string, error)
	expand = func(key string, visiting map[string]bool) ([]string, error) {
		if visiting[key] {
			return nil, fmt.Errorf("工作流依赖存在循环")
		}
		if _, ok := byKey[key]; !ok {
			return nil, fmt.Errorf("工作流依赖不存在")
		}
		n, skip := removed[key]
		if !skip {
			return []string{key}, nil
		}
		visiting[key] = true
		defer delete(visiting, key)
		var out []string
		for _, dep := range n.DependsOn {
			v, e := expand(dep, visiting)
			if e != nil {
				return nil, e
			}
			out = append(out, v...)
		}
		return out, nil
	}
	result := make([]Node, 0, len(nodes))
	deploys := 0
	for _, n := range nodes {
		if n.Type == NodeBuild {
			continue
		}
		if n.Type == NodeYAMLDeploy {
			var c YAMLDeployConfig
			if json.Unmarshal(n.Config, &c) != nil || c.ServiceID != serviceID || c.FrozenYAML == nil {
				return nil, fmt.Errorf("复用镜像前必须冻结目标环境的服务配置")
			}
			source := *c.FrozenYAML
			source.ImageRef = b.ImageURL + "@" + b.ImageDigest
			c.FrozenYAML = &source
			n.Config, _ = json.Marshal(c)
			deploys++
		}
		if n.Type == NodeHelmDeploy {
			var c HelmDeployConfig
			if json.Unmarshal(n.Config, &c) != nil || c.ServiceID != serviceID || c.FrozenHelm == nil {
				return nil, fmt.Errorf("复用镜像前必须冻结目标环境的 Helm 配置")
			}
			c.ImageRef = b.ImageURL + "@" + b.ImageDigest
			n.Config, _ = json.Marshal(c)
			deploys++
		}
		deps := []string{}
		seen := map[string]bool{}
		for _, key := range n.DependsOn {
			expanded, e := expand(key, map[string]bool{})
			if e != nil {
				return nil, e
			}
			for _, d := range expanded {
				if !seen[d] {
					deps = append(deps, d)
					seen[d] = true
				}
			}
		}
		n.DependsOn = deps
		result = append(result, n)
	}
	if deploys != 1 {
		return nil, fmt.Errorf("复用镜像需要恰好一个目标服务的 YAML 或 Helm 部署节点")
	}
	return result, nil
}
