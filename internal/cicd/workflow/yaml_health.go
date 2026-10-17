package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type YAMLHealthTarget struct {
	YAMLTarget
	Deadline time.Time
}
type YAMLHealthTargetStore interface {
	HealthTarget(context.Context, string) (YAMLHealthTarget, error)
}
type resourceReader interface {
	ReadResource(context.Context, string, unstructured.Unstructured) (*unstructured.Unstructured, error)
}

// workloadDiagnosticReader is intentionally optional. Readiness checks must
// continue to work with lightweight test clients and older providers; the
// real dynamic client adds pod/container/event context when a rollout fails.
type workloadDiagnosticReader interface {
	DiagnoseWorkload(context.Context, string, unstructured.Unstructured) (string, error)
}

func (a *dynamicResourceApplier) ReadResource(ctx context.Context, ns string, obj unstructured.Unstructured) (*unstructured.Unstructured, error) {
	client, err := a.resourceClient(obj.GroupVersionKind(), ns)
	if err != nil {
		return nil, err
	}
	return client.Get(ctx, obj.GetName(), metav1.GetOptions{})
}

// Reload the immutable deployment target on every observation, including after
// worker restarts. No kubeconfig or application YAML goes into the external ID.
func (s *SQLDeploymentSnapshotStore) HealthTarget(ctx context.Context, key string) (YAMLHealthTarget, error) {
	parts := strings.Split(key, ":")
	if len(parts) != 3 && len(parts) != 5 {
		return YAMLHealthTarget{}, fmt.Errorf("invalid YAML execution key")
	}
	runID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || runID <= 0 || parts[0] != "workflow" {
		return YAMLHealthTarget{}, fmt.Errorf("invalid YAML run id")
	}
	var target YAMLHealthTarget
	var started time.Time
	var raw []byte
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE(started_at,created_at),node_snapshot FROM workflow_task_run WHERE workflow_run_id=? AND node_key=?`, runID, parts[2]).Scan(&started, &raw)
	if err != nil {
		return target, err
	}
	var node Node
	if err = json.Unmarshal(raw, &node); err != nil {
		return target, err
	}
	timeout := node.TimeoutSeconds
	if timeout <= 0 {
		timeout = 600
	}
	target.Deadline = started.Add(time.Duration(timeout) * time.Second)
	if len(parts) == 5 {
		if parts[3] != "rollback" {
			return target, fmt.Errorf("invalid rollback key")
		}
		id, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			return target, err
		}
		snapshot, err := s.FindApplied(ctx, id)
		if err != nil {
			return target, err
		}
		if snapshot == nil {
			return target, fmt.Errorf("rollback snapshot missing")
		}
		target.YAMLTarget = YAMLTarget{ClusterID: snapshot.ClusterID, Namespace: snapshot.Namespace, Manifest: snapshot.RenderedManifest}
	} else {
		err = s.db.QueryRowContext(ctx, `SELECT cluster_id,namespace,rendered_manifest FROM deployment_snapshot WHERE workflow_run_id=? AND node_key=? AND provider=?`, runID, parts[2], NodeYAMLDeploy).Scan(&target.ClusterID, &target.Namespace, &target.Manifest)
	}
	return target, err
}

func (p *KubernetesYAMLProvider) checkTarget(ctx context.Context, target YAMLHealthTarget) (YAMLProviderState, error) {
	deadlineExceeded := !target.Deadline.IsZero() && time.Now().After(target.Deadline)
	resources, err := ValidateYAMLManifest(target.Manifest, target.Namespace)
	if err != nil {
		return YAMLProviderState{}, err
	}
	client, err := p.factory.Client(ctx, target.ClusterID)
	if err != nil {
		if deadlineExceeded {
			message := "deployment readiness timeout; cluster unavailable: " + safeDiagnostic(err.Error())
			return YAMLProviderState{Status: RunFailed, ErrorMessage: message, LogText: "[failed] " + message}, nil
		}
		return YAMLProviderState{}, err
	}
	reader, ok := client.(resourceReader)
	if !ok {
		return YAMLProviderState{}, fmt.Errorf("target cluster does not support readiness checks")
	}
	state := YAMLProviderState{Status: RunSucceeded}
	lines := []string{fmt.Sprintf("[check] cluster=%d namespace=%s", target.ClusterID, target.Namespace)}
	checked := 0
	for _, obj := range resources {
		switch obj.GetKind() {
		case "Deployment", "StatefulSet", "DaemonSet", "Job":
		default:
			lines = append(lines, fmt.Sprintf("[resource] %s/%s submitted; no rollout check for this kind", obj.GetKind(), obj.GetName()))
			continue
		}
		checked++
		live, err := reader.ReadResource(ctx, target.Namespace, obj)
		if err != nil {
			message := fmt.Sprintf("read %s/%s status: %s", obj.GetKind(), obj.GetName(), safeDiagnostic(err.Error()))
			lines = append(lines, "[check] "+message)
			if state.Status != RunFailed {
				state.Status = RunRunning
			}
			if deadlineExceeded {
				state.Status = RunFailed
				state.ErrorMessage = "deployment readiness timeout: " + message
			}
			continue
		}
		ready, failed, reason := workloadReady(live)
		lines = append(lines, workloadCheckSummary(live, ready))
		if replicas, found, _ := unstructured.NestedInt64(live.Object, "spec", "replicas"); found && replicas == 0 && (live.GetKind() == "Deployment" || live.GetKind() == "StatefulSet") {
			lines = append(lines, "[rollout] desired replicas=0: scaled down by configuration; rollout completion does not mean the application is serving traffic")
		}
		conditions, _, _ := unstructured.NestedSlice(live.Object, "status", "conditions")
		for _, raw := range conditions {
			if c, ok := raw.(map[string]interface{}); ok {
				lines = append(lines, safeDiagnostic(conditionDiagnostic("[condition] "+live.GetKind()+"/"+live.GetName(), c)))
			}
		}
		diagnostic := appendWorkloadDiagnostic(ctx, client, target.Namespace, *live, "")
		if diagnostic != "" {
			lines = append(lines, diagnostic)
		}
		if failed || (!ready && deadlineExceeded) {
			state.Status = RunFailed
			if !failed {
				reason = "deployment readiness timeout: " + reason
			}
			state.ErrorMessage = truncateDiagnostic(safeDiagnostic(reason)+"\n"+diagnostic, 8192)
		} else if !ready && state.Status != RunFailed {
			state.Status = RunRunning
		}
	}
	if state.Status == RunSucceeded {
		if checked == 0 {
			lines = append(lines, "[succeeded] resources submitted; manifest has no supported rollout target; application health not verified")
		} else {
			lines = append(lines, "[succeeded] Kubernetes rollout checks passed. Application health follows configured readiness probes; external traffic and business checks are not included.")
		}
	} else if state.Status == RunFailed {
		lines = append(lines, "[failed] "+state.ErrorMessage)
	} else {
		lines = append(lines, "[waiting] waiting for rollout; transient pod errors will be retried until the deployment deadline")
	}
	state.LogText = strings.Join(lines, "\n")
	return state, nil
}

func appendWorkloadDiagnostic(ctx context.Context, client KubernetesResourceApplier, namespace string, workload unstructured.Unstructured, reason string) string {
	diagnostics, ok := client.(workloadDiagnosticReader)
	if !ok {
		return reason
	}
	detail, err := diagnostics.DiagnoseWorkload(ctx, namespace, workload)
	if err != nil {
		return strings.TrimSpace(reason + "\n[diagnostics] unavailable: " + safeDiagnostic(err.Error()))
	}
	if strings.TrimSpace(detail) == "" {
		return reason
	}
	return strings.TrimSpace(reason + "\n" + detail)
}

func workloadCheckSummary(obj *unstructured.Unstructured, ready bool) string {
	n := func(fields ...string) int64 { v, _, _ := unstructured.NestedInt64(obj.Object, fields...); return v }
	desired, found, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if !found {
		desired = 1
	}
	if obj.GetKind() == "DaemonSet" {
		desired = n("status", "desiredNumberScheduled")
	}
	return fmt.Sprintf("[rollout] %s/%s ready=%t generation=%d observed=%d desired=%d replicas=%d updated=%d readyReplicas=%d available=%d scheduled=%d updatedScheduled=%d availableScheduled=%d succeeded=%d failed=%d", obj.GetKind(), obj.GetName(), ready, obj.GetGeneration(), n("status", "observedGeneration"), desired, n("status", "replicas"), n("status", "updatedReplicas"), n("status", "readyReplicas"), n("status", "availableReplicas"), n("status", "currentNumberScheduled"), n("status", "updatedNumberScheduled"), n("status", "numberAvailable"), n("status", "succeeded"), n("status", "failed"))
}

func workloadReady(obj *unstructured.Unstructured) (bool, bool, string) {
	n := func(fields ...string) int64 { v, _, _ := unstructured.NestedInt64(obj.Object, fields...); return v }
	reason := obj.GetKind() + "/" + obj.GetName() + " waiting for rollout"
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, value := range conditions {
		c, _ := value.(map[string]interface{})
		if (obj.GetKind() == "Job" && c["type"] == "Failed" && c["status"] == "True") || (c["type"] == "Progressing" && c["reason"] == "ProgressDeadlineExceeded" && c["status"] == "False") {
			return false, true, fmt.Sprintf("%s: %v", reason, c["message"])
		}
		if obj.GetKind() == "Job" && c["type"] == "Complete" && c["status"] == "True" {
			return true, false, ""
		}
	}
	if obj.GetKind() == "Job" {
		return false, false, reason
	}
	if n("status", "observedGeneration") < obj.GetGeneration() {
		return false, false, reason
	}
	desired, found, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if !found {
		desired = 1
	}
	ready := false
	switch obj.GetKind() {
	case "Deployment":
		ready = n("status", "updatedReplicas") == desired && n("status", "availableReplicas") == desired && n("status", "replicas") == desired
	case "StatefulSet":
		strategy, _, _ := unstructured.NestedString(obj.Object, "spec", "updateStrategy", "type")
		partition := n("spec", "updateStrategy", "rollingUpdate", "partition")
		if partition > desired {
			partition = desired
		}
		ready = n("status", "readyReplicas") == desired && (strategy == "OnDelete" || n("status", "updatedReplicas") >= desired-partition)
	case "DaemonSet":
		desired = n("status", "desiredNumberScheduled")
		ready = desired > 0 && n("status", "updatedNumberScheduled") == desired && n("status", "numberAvailable") == desired
	}
	return ready, false, reason
}
