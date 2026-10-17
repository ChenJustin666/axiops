package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const restartAnnotation = "axiops.io/restart-id"

var getRestartOptions = metav1.GetOptions{}

func restartResourceClient(ctx context.Context, f HelmRESTConfigFactory, cid int64, ns, kind string) (dynamic.ResourceInterface, error) {
	if f == nil || cid <= 0 || ns == "" || restartKind(kind) == "" {
		return nil, fmt.Errorf("invalid restart target")
	}
	config, err := f.RESTConfig(ctx, cid)
	if err != nil {
		return nil, err
	}
	config = rest.CopyConfig(config)
	config.Timeout = 15 * time.Second
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	resource := map[string]string{"Deployment": "deployments", "StatefulSet": "statefulsets", "DaemonSet": "daemonsets"}[restartKind(kind)]
	return client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: resource}).Namespace(ns), nil
}

type WorkloadRestartExecutor struct {
	db      *sql.DB
	factory HelmRESTConfigFactory
	logic   *Logic
}

func NewWorkloadRestartExecutor(db *sql.DB, factory HelmRESTConfigFactory, checker authz.Checker) *WorkloadRestartExecutor {
	return &WorkloadRestartExecutor{db: db, factory: factory, logic: NewLogic(db, checker)}
}
func (e *WorkloadRestartExecutor) NodeType() string { return NodeWorkloadRestart }

func restartFrozen(request ExecutionRequest) (restartContext, error) {
	var f restartContext
	var n Node
	var g RestartGuard
	if json.Unmarshal(request.Run.ContextSnapshot, &f) != nil || json.Unmarshal(request.Task.NodeSnapshot, &n) != nil || json.Unmarshal(n.Config, &g) != nil || g != f.Guard || f.Target == nil || f.Actor == nil || f.Actor.UserID != request.Run.TriggerUserID || f.Actor.Username == "" || f.Guard.UID == "" || f.Guard.Generation <= 0 || len(f.Guard.SpecHash) != 64 || f.Target.ID != request.Run.EnvironmentID || f.Target.ProjectID != request.Run.ProjectID || f.Guard.WorkflowID != request.Run.WorkflowID || f.Target.ClusterID <= 0 || f.Target.Namespace == "" || restartKind(g.Kind) != g.Kind || g.Name == "" {
		return f, fmt.Errorf("重启任务缺少有效的目标和执行人快照")
	}
	return f, nil
}

func restartMarker(r ExecutionRequest) string {
	return fmt.Sprintf("workflow-%d-%s", r.Run.ID, r.Task.NodeKey)
}

func restartMatches(o *unstructured.Unstructured, g RestartGuard) bool {
	return string(o.GetUID()) == g.UID && restartSpecHash(o) == g.SpecHash
}
func restartOwns(o *unstructured.Unstructured, r ExecutionRequest) bool {
	marker, _, _ := unstructured.NestedString(o.Object, "spec", "template", "metadata", "annotations", restartAnnotation)
	return marker == restartMarker(r)
}

// A resourceVersion test makes competing updates (including another restart)
// fail instead of overwriting them. No container configuration is replaced.
func restartPatch(o *unstructured.Unstructured, marker string) ([]byte, error) {
	annotations, found, err := unstructured.NestedStringMap(o.Object, "spec", "template", "metadata", "annotations")
	if err != nil {
		return nil, err
	}
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[restartAnnotation] = marker
	ops := []map[string]interface{}{{"op": "test", "path": "/metadata/uid", "value": string(o.GetUID())}, {"op": "test", "path": "/metadata/resourceVersion", "value": o.GetResourceVersion()}}
	op := "add"
	if found {
		op = "replace"
	}
	ops = append(ops, map[string]interface{}{"op": op, "path": "/spec/template/metadata/annotations", "value": annotations})
	return json.Marshal(ops)
}

func (e *WorkloadRestartExecutor) Execute(ctx context.Context, r ExecutionRequest) (ExecutionResult, error) {
	if r.Task.StartedAt == nil {
		if err := e.db.QueryRowContext(ctx, "SELECT started_at FROM workflow_task_run WHERE workflow_run_id=? AND node_key=?", r.Run.ID, r.Task.NodeKey).Scan(&r.Task.StartedAt); err != nil {
			return ExecutionResult{}, err
		}
	}
	f, err := restartFrozen(r)
	if err != nil {
		return ExecutionResult{}, err
	}
	client, err := restartResourceClient(ctx, e.factory, f.Target.ClusterID, f.Target.Namespace, f.Guard.Kind)
	if err != nil {
		return ExecutionResult{}, err
	}
	o, err := client.Get(ctx, f.Guard.Name, getRestartOptions)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("读取重启目标失败: %w", err)
	}
	if !restartMatches(o, f.Guard) {
		return ExecutionResult{}, fmt.Errorf("工作负载被替换或配置已变化，已停止重启")
	}
	// Recovery recognizes this exact request before permission/version checks.
	// A revoked actor must never patch again, but an existing rollout is observed.
	if restartOwns(o, r) {
		return ExecutionResult{Status: RunRunning, ProviderRunID: restartMarker(r), LogText: "[recover] 已确认原重启请求，继续检查就绪状态"}, nil
	}
	if o.GetGeneration() != f.Guard.Generation {
		return ExecutionResult{}, fmt.Errorf("工作负载已更新，请重新预览")
	}
	if err := validateRestartWorkload(o); err != nil {
		return ExecutionResult{}, err
	}
	if code := e.logic.authorizeRestart(ctx, r.Run.TriggerUserID, f.Target, f.Guard); code != 0 {
		return ExecutionResult{}, fmt.Errorf("执行人当前已无此服务的重启权限")
	}
	if err := validateRestartDatabase(ctx, e.db, r.Run, f); err != nil {
		return ExecutionResult{}, err
	}
	if restartExpired(r, time.Now()) {
		return ExecutionResult{}, fmt.Errorf("重启任务已超时，请检查当前资源状态")
	}
	patch, err := restartPatch(o, restartMarker(r))
	if err != nil {
		return ExecutionResult{}, err
	}
	if r.ReportProgress != nil {
		if err := r.ReportProgress(fmt.Sprintf("[target] 集群 #%d · %s · %s/%s；保留原镜像与配置", f.Target.ClusterID, f.Target.Namespace, f.Guard.Kind, f.Guard.Name)); err != nil {
			return ExecutionResult{}, err
		}
	}
	_, err = client.Patch(ctx, f.Guard.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	if err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) || apierrors.IsForbidden(err) || apierrors.IsNotFound(err) || apierrors.IsUnauthorized(err) {
			return ExecutionResult{}, fmt.Errorf("重启提交被拒绝，请刷新资源后重试: %w", err)
		}
		return ExecutionResult{}, fmt.Errorf("%w: 重启提交结果待确认", ErrSubmissionUncertain)
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: restartMarker(r), LogText: "[restart] 已提交滚动重启，等待新容器就绪"}, nil
}

func restartExpired(r ExecutionRequest, now time.Time) bool {
	// StartedAt is persisted on first claim and survives worker replacement.
	return r.Task.StartedAt != nil && now.Sub(*r.Task.StartedAt) >= 600*time.Second
}

func (e *WorkloadRestartExecutor) Reconcile(ctx context.Context, r ExecutionRequest) (ExecutionResult, error) {
	f, err := restartFrozen(r)
	if err != nil {
		return restartFailure(err.Error()), nil
	}
	if restartExpired(r, time.Now()) {
		return restartFailure("滚动重启等待超过 10 分钟，请查看环境资源的 Pod 状态与事件；集群仍会继续按自身策略处理"), nil
	}
	client, err := restartResourceClient(ctx, e.factory, f.Target.ClusterID, f.Target.Namespace, f.Guard.Kind)
	if err != nil {
		return ExecutionResult{}, err
	}
	o, err := client.Get(ctx, f.Guard.Name, getRestartOptions)
	if apierrors.IsNotFound(err) {
		return restartFailure("原工作负载已删除"), nil
	}
	if err != nil {
		return ExecutionResult{}, err
	}
	return observeRestart(o, r, f), nil
}

func restartFailure(message string) ExecutionResult {
	return ExecutionResult{Status: RunFailed, ErrorMessage: message, LogText: "[failed] " + message}
}

func observeRestart(o *unstructured.Unstructured, r ExecutionRequest, f restartContext) ExecutionResult {
	if !restartMatches(o, f.Guard) || !restartOwns(o, r) || o.GetGeneration() != f.Guard.Generation+1 {
		return restartFailure("重启期间工作负载被替换或有其他配置更新，请核对实际资源")
	}
	observed, _, _ := unstructured.NestedInt64(o.Object, "status", "observedGeneration")
	if observed < o.GetGeneration() {
		return ExecutionResult{Status: RunRunning, LogText: "[waiting] 等待控制器确认本次重启"}
	}
	ready, failed, reason := workloadReady(o)
	if o.GetKind() == "StatefulSet" {
		current, _, _ := unstructured.NestedString(o.Object, "status", "currentRevision")
		updated, _, _ := unstructured.NestedString(o.Object, "status", "updateRevision")
		ready = ready && current != "" && current == updated
	}
	if failed {
		return restartFailure("工作负载重启失败：" + safeDiagnostic(reason))
	}
	if ready {
		return ExecutionResult{Status: RunSucceeded, LogText: "[ready] 滚动重启完成，工作负载已就绪"}
	}
	return ExecutionResult{Status: RunRunning, LogText: "[waiting] 滚动重启中，等待控制器完成更新并就绪"}
}

func (e *WorkloadRestartExecutor) Cancel(ctx context.Context, r ExecutionRequest) (ExecutionResult, error) {
	f, err := restartFrozen(r)
	if err != nil {
		return restartFailure(err.Error()), nil
	}
	client, err := restartResourceClient(ctx, e.factory, f.Target.ClusterID, f.Target.Namespace, f.Guard.Kind)
	if err != nil {
		return ExecutionResult{}, err
	}
	o, err := client.Get(ctx, f.Guard.Name, getRestartOptions)
	if apierrors.IsNotFound(err) {
		return ExecutionResult{Status: RunCancelled}, nil
	}
	if err != nil {
		return ExecutionResult{}, err
	}
	if !restartOwns(o, r) {
		return ExecutionResult{Status: RunCancelled, LogText: "[cancel] 未发现此请求的重启标记，已停止任务"}, nil
	}
	if restartExpired(r, time.Now()) {
		return restartFailure("已停止等待；已提交的滚动重启无法撤回，请查看环境资源"), nil
	}
	result := observeRestart(o, r, f)
	result.LogText = "[cancel] 已提交的滚动重启继续由集群执行，不删除资源；" + result.LogText
	return result, nil
}

var _ TaskExecutor = (*WorkloadRestartExecutor)(nil)
var _ TaskReconciler = (*WorkloadRestartExecutor)(nil)
var _ TaskCanceller = (*WorkloadRestartExecutor)(nil)
