package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

func (l *Logic) selectedRunNodes(ctx context.Context, userID int64, workflow *Workflow, selections []RunServiceSelection, mode string) ([]Node, int, error) {
	runner := l.repo.(runRepository)
	seen := map[int64]bool{}
	var result []Node
	for index, selection := range selections {
		if selection.ServiceID <= 0 || selection.ServiceRevisionID <= 0 || seen[selection.ServiceID] {
			return nil, errcode.CodeParamInvalid, fmt.Errorf("批量服务重复或缺少有效版本")
		}
		seen[selection.ServiceID] = true
		if mode == "build" && (strings.TrimSpace(selection.Branch) == "" || selection.SourceBuildID != 0) || mode == "existing_image" && (selection.SourceBuildID <= 0 || selection.Branch != "" || (workflow.DeliveryType != DeliveryTypeK8sYAML && workflow.DeliveryType != DeliveryTypeHelm)) {
			return nil, errcode.CodeParamInvalid, fmt.Errorf("服务 %d 的分支或镜像来源无效", selection.ServiceID)
		}
		selected, err := selectWorkflowServiceNodes(workflow.Nodes, selection.ServiceID)
		if err != nil {
			return nil, errcode.CodeParamInvalid, err
		}
		allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "view", ProjectID: workflow.ProjectID, ServiceID: selection.ServiceID})
		if err != nil {
			return nil, errcode.CodeDBError, err
		}
		if !allowed {
			return nil, errcode.CodeNoPermission, fmt.Errorf("无权执行所选服务")
		}
		target, err := runner.FindReleaseTarget(ctx, selection.ServiceID, selection.ServiceRevisionID)
		if err != nil {
			return nil, errcode.CodeDBError, err
		}
		if target == nil || target.ProjectID != workflow.ProjectID || target.DeliveryType != workflow.DeliveryType {
			return nil, errcode.CodeParamInvalid, fmt.Errorf("所选服务版本不属于当前项目或交付类型不匹配")
		}
		selected, err = applyReleaseOverride(selected, selection.ServiceID, selection.ServiceRevisionID)
		if err == nil && mode == "build" {
			selected, err = applyBranchOverride(selected, strings.TrimSpace(selection.Branch))
		}
		if err != nil {
			return nil, errcode.CodeParamInvalid, err
		}
		for i := range selected {
			if selected[i].Type == NodeYAMLDeploy || selected[i].Type == NodeHelmDeploy || selected[i].Type == NodeHostDeploy {
				var config map[string]json.RawMessage
				if err := json.Unmarshal(selected[i].Config, &config); err != nil {
					return nil, errcode.CodeParamInvalid, err
				}
				config["start_order"], _ = json.Marshal(index + 1)
				selected[i].Config, _ = json.Marshal(config)
			}
		}
		result = append(result, selected...)
	}
	return result, errcode.CodeSuccess, nil
}

func (l *Logic) promoteSelectedRun(ctx context.Context, userID int64, workflow *Workflow, nodes []Node, selections []RunServiceSelection) ([]Node, error) {
	loader, ok := l.repo.(interface {
		LoadPromotionBuild(context.Context, int64) (*PromotionBuild, error)
	})
	if !ok {
		return nil, fmt.Errorf("镜像来源读取不可用")
	}
	var result []Node
	for _, selection := range selections {
		build, err := loader.LoadPromotionBuild(ctx, selection.SourceBuildID)
		if err != nil {
			return nil, err
		}
		if build == nil || build.ProjectID != workflow.ProjectID || build.ServiceID != selection.ServiceID || build.Status != "success" || !validPromotionImage(build) {
			return nil, fmt.Errorf("服务 %d 的构建没有成功生成可追溯镜像", selection.ServiceID)
		}
		allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "view", ProjectID: workflow.ProjectID, ServiceID: selection.ServiceID, BuildID: build.ID})
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, fmt.Errorf("无权读取服务 %d 的所选构建", selection.ServiceID)
		}
		selected, err := selectWorkflowServiceNodes(nodes, selection.ServiceID)
		if err != nil {
			return nil, err
		}
		selected, err = promoteYAMLNodes(selected, selection.ServiceID, build)
		if err != nil {
			return nil, err
		}
		result = append(result, selected...)
	}
	return result, nil
}
