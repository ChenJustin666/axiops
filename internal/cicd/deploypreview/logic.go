package deploypreview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type yamlResolver interface {
	RenderYAML(context.Context, workflow.WorkflowRun, workflow.YAMLDeployConfig) (workflow.YAMLTarget, error)
}

type helmResolver interface {
	RenderHelm(context.Context, workflow.WorkflowRun, workflow.HelmDeployConfig) (workflow.HelmTarget, error)
}

type helmRenderer interface {
	RenderPreview(context.Context, workflow.HelmRequest) ([]byte, error)
}

type kubernetesPreviewer interface {
	Preview(context.Context, workflow.YAMLProviderRequest) ([]workflow.YAMLResourceChange, error)
	PreviewHelm(context.Context, int64, string, []byte) ([]workflow.YAMLResourceChange, error)
}

type clusterFactory interface {
	workflow.KubernetesClientFactory
	workflow.HelmRESTConfigFactory
}

type Logic struct {
	repo       previewRepository
	authorizer authz.Checker
	yaml       yamlResolver
	helm       helmResolver
	renderer   helmRenderer
	previewer  kubernetesPreviewer
}

func NewLogic(db *sql.DB, authorizer authz.Checker, decryptor workflow.CredentialDecryptor, factory clusterFactory, fieldManager string) *Logic {
	return &Logic{
		repo: NewRepository(db), authorizer: authorizer,
		yaml:      workflow.NewDefaultYAMLResolver(workflow.NewSQLYAMLSourceStore(db)),
		helm:      workflow.NewDefaultHelmResolver(workflow.NewSQLHelmSourceStore(db), decryptor),
		renderer:  workflow.NewOfficialHelmProvider(db, factory),
		previewer: workflow.NewKubernetesYAMLProvider(factory, fieldManager, false),
	}
}

func (l *Logic) Preview(ctx context.Context, userID, environmentID int64, request Request) (*Result, int) {
	if l == nil || l.repo == nil || l.authorizer == nil || l.previewer == nil || request.ServiceID <= 0 || request.ServiceRevisionID < 0 {
		return nil, errcode.CodeParamInvalid
	}
	target, err := l.repo.ResolveTarget(ctx, environmentID, request.ServiceID, request.ServiceRevisionID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if target == nil {
		return nil, errcode.CodeNotFound
	}
	if target.EnvironmentType != "k8s_namespace" || target.BindingStatus != "validated" || target.EnvironmentStatus != "active" || target.ClusterID <= 0 || strings.TrimSpace(target.Namespace) == "" {
		return nil, errcode.CodeParamInvalid
	}
	if target.DeliveryType != workflow.DeliveryTypeK8sYAML && target.DeliveryType != workflow.DeliveryTypeHelm {
		return nil, errcode.CodeParamInvalid
	}
	if code := l.authorize(ctx, userID, target); code != errcode.CodeSuccess {
		return nil, code
	}

	run := workflow.WorkflowRun{ProjectID: target.ProjectID, EnvironmentID: target.EnvironmentID}
	result := &Result{
		DeliveryType: target.DeliveryType, ProjectID: target.ProjectID, EnvironmentID: target.EnvironmentID,
		ServiceID: target.ServiceID, ServiceRevisionID: target.ServiceRevisionID,
		ClusterID: target.ClusterID, Namespace: target.Namespace,
	}
	switch target.DeliveryType {
	case workflow.DeliveryTypeK8sYAML:
		if l.yaml == nil {
			return nil, errcode.CodeInternalError
		}
		rendered, err := l.yaml.RenderYAML(ctx, run, workflow.YAMLDeployConfig{ServiceID: target.ServiceID, ServiceRevisionID: target.ServiceRevisionID})
		if err != nil {
			return nil, errcode.CodeParamInvalid
		}
		result.Changes, err = l.previewer.Preview(ctx, workflow.YAMLProviderRequest{
			ClusterID: rendered.ClusterID, Namespace: rendered.Namespace, Manifest: rendered.Manifest,
			IdempotencyKey: fmt.Sprintf("preview:%d:%d:%d", target.EnvironmentID, target.ServiceID, target.ServiceRevisionID),
		})
		if err != nil {
			return nil, errcode.CodeClusterConnectFail
		}
	case workflow.DeliveryTypeHelm:
		if l.helm == nil || l.renderer == nil {
			return nil, errcode.CodeInternalError
		}
		releaseName := strings.TrimSpace(request.ReleaseName)
		if releaseName == "" {
			releaseName = target.ServiceCode
		}
		config := workflow.HelmDeployConfig{ServiceID: target.ServiceID, ServiceRevisionID: target.ServiceRevisionID, ReleaseName: releaseName}
		var rendered workflow.HelmTarget
		var err error
		if preview, ok := l.helm.(interface {
			RenderHelmConfig(context.Context, workflow.WorkflowRun, workflow.HelmDeployConfig) (workflow.HelmTarget, error)
		}); ok {
			rendered, err = preview.RenderHelmConfig(ctx, run, config)
		} else {
			rendered, err = l.helm.RenderHelm(ctx, run, config)
		}
		if err != nil {
			return nil, errcode.CodeParamInvalid
		}
		manifest, err := l.renderer.RenderPreview(ctx, workflow.HelmRequest{
			RunID: 1, NodeKey: "preview", ClusterID: rendered.ClusterID, Namespace: rendered.Namespace,
			ReleaseName: rendered.ReleaseName, ChartPath: rendered.ChartPath, Chart: rendered.Chart,
			Values: rendered.Values, TimeoutSeconds: rendered.TimeoutSeconds, IdempotencyKey: "preview",
		})
		if err != nil {
			return nil, errcode.CodeParamInvalid
		}
		result.Changes, err = l.previewer.PreviewHelm(ctx, rendered.ClusterID, rendered.Namespace, manifest)
		if err != nil {
			return nil, errcode.CodeClusterConnectFail
		}
		previous, err := l.repo.LoadLatestHelmInventory(ctx, target.EnvironmentID, target.ServiceID, rendered.ReleaseName)
		if err != nil {
			return nil, errcode.CodeDBError
		}
		result.Changes, err = appendHelmDeletes(result.Changes, previous)
		if err != nil {
			return nil, errcode.CodeInternalError
		}
		result.ReleaseName = rendered.ReleaseName
		result.Values = rendered.SnapshotValues
	}
	sortChanges(result.Changes)
	result.Summary = summarize(result.Changes)
	return result, errcode.CodeSuccess
}

func (l *Logic) authorize(ctx context.Context, userID int64, target *Target) int {
	requests := []authz.Request{
		{Resource: "environment", Action: "view", ProjectID: target.ProjectID, EnvironmentID: target.EnvironmentID, ClusterID: target.ClusterID, Environment: target.EnvironmentCode},
		{Resource: "service", Action: "view", ProjectID: target.ProjectID, ServiceID: target.ServiceID, EnvironmentID: target.EnvironmentID, ClusterID: target.ClusterID, Environment: target.EnvironmentCode},
		{Resource: "cluster", Action: "view", ProjectID: target.ProjectID, ServiceID: target.ServiceID, EnvironmentID: target.EnvironmentID, ClusterID: target.ClusterID, Environment: target.EnvironmentCode},
	}
	for _, request := range requests {
		allowed, err := l.authorizer.Authorize(ctx, userID, request)
		if err != nil {
			return errcode.CodeDBError
		}
		if !allowed {
			return errcode.CodeNoPermission
		}
	}
	return errcode.CodeSuccess
}

func appendHelmDeletes(changes []workflow.YAMLResourceChange, inventory []byte) ([]workflow.YAMLResourceChange, error) {
	if len(inventory) == 0 {
		return changes, nil
	}
	var previous []workflow.YAMLResourceChange
	if err := json.Unmarshal(inventory, &previous); err != nil {
		return nil, fmt.Errorf("decode previous Helm resource inventory: %w", err)
	}
	current := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		current[changeKey(change)] = struct{}{}
	}
	for _, item := range previous {
		if _, exists := current[changeKey(item)]; exists {
			continue
		}
		item.Action = "delete"
		changes = append(changes, item)
	}
	return changes, nil
}

func changeKey(change workflow.YAMLResourceChange) string {
	return change.APIVersion + "/" + change.Kind + "/" + change.Namespace + "/" + change.Name
}

func sortChanges(changes []workflow.YAMLResourceChange) {
	order := map[string]int{"create": 0, "update": 1, "delete": 2, "unchanged": 3}
	sort.Slice(changes, func(i, j int) bool {
		if order[changes[i].Action] != order[changes[j].Action] {
			return order[changes[i].Action] < order[changes[j].Action]
		}
		return changeKey(changes[i]) < changeKey(changes[j])
	})
}

func summarize(changes []workflow.YAMLResourceChange) Summary {
	var summary Summary
	for _, change := range changes {
		switch change.Action {
		case "create":
			summary.Create++
		case "update":
			summary.Update++
		case "delete":
			summary.Delete++
		case "unchanged":
			summary.Unchanged++
		}
	}
	return summary
}
