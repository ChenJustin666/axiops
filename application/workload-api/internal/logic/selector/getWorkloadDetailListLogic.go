package selector

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChenJustin666/axiops/application/workload-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/workload-api/internal/types"
	k8stypes "github.com/ChenJustin666/axiops/common/k8smanager/types"

	"github.com/zeromicro/go-zero/core/logx"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type GetWorkloadDetailListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get workload detail list (environment view)
func NewGetWorkloadDetailListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWorkloadDetailListLogic {
	return &GetWorkloadDetailListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWorkloadDetailListLogic) GetWorkloadDetailList(req *types.WorkloadDetailListRequest) (resp *types.WorkloadDetailListResponse, err error) {
	client, err := l.svcCtx.K8sManager.GetCluster(l.ctx, req.ClusterUuid)
	if err != nil {
		l.Errorf("get cluster client failed: %v", err)
		return nil, fmt.Errorf("get cluster client failed")
	}

	var items []types.WorkloadDetailItem

	switch req.Type {
	case "deployment":
		list, e := client.Deployment().ListAll(req.Namespace)
		if e != nil {
			return nil, fmt.Errorf("list Deployment failed: %v", e)
		}
		for i := range list {
			items = append(items, deploymentToItem(&list[i]))
		}
	case "statefulset":
		list, e := client.StatefulSet().ListAll(req.Namespace)
		if e != nil {
			return nil, fmt.Errorf("list StatefulSet failed: %v", e)
		}
		for i := range list {
			items = append(items, statefulSetToItem(&list[i]))
		}
	case "daemonset":
		list, e := client.DaemonSet().ListAll(req.Namespace)
		if e != nil {
			return nil, fmt.Errorf("list DaemonSet failed: %v", e)
		}
		for i := range list {
			items = append(items, daemonSetToItem(&list[i]))
		}
	case "cronjob":
		list, e := client.CronJob().ListAll(req.Namespace)
		if e != nil {
			return nil, fmt.Errorf("list CronJob failed: %v", e)
		}
		for i := range list {
			items = append(items, cronJobToItem(&list[i]))
		}
	case "job":
		jobResp, e := client.Job().List(req.Namespace, k8stypes.ListRequest{Page: 1, PageSize: 500})
		if e != nil {
			return nil, fmt.Errorf("list Job failed: %v", e)
		}
		for i := range jobResp.Items {
			items = append(items, jobInfoToItem(&jobResp.Items[i]))
		}
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", req.Type)
	}

	// optional name search filter
	if keyword := strings.TrimSpace(req.Search); keyword != "" {
		kw := strings.ToLower(keyword)
		filtered := items[:0]
		for _, it := range items {
			if strings.Contains(strings.ToLower(it.Name), kw) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	if items == nil {
		items = []types.WorkloadDetailItem{}
	}

	return &types.WorkloadDetailListResponse{
		Total: len(items),
		Items: items,
	}, nil
}

func containerImages(containers []corev1.Container) []string {
	images := make([]string, 0, len(containers))
	for _, c := range containers {
		images = append(images, c.Image)
	}
	return images
}

func deploymentToItem(d *appsv1.Deployment) types.WorkloadDetailItem {
	var desired int32
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	ready := d.Status.ReadyReplicas
	return types.WorkloadDetailItem{
		Name:            d.Name,
		Namespace:       d.Namespace,
		Kind:            "Deployment",
		Ready:           fmt.Sprintf("%d/%d", ready, desired),
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
		Images:          containerImages(d.Spec.Template.Spec.Containers),
		Labels:          d.Labels,
		CreatedAt:       d.CreationTimestamp.UnixMilli(),
	}
}

func statefulSetToItem(s *appsv1.StatefulSet) types.WorkloadDetailItem {
	var desired int32
	if s.Spec.Replicas != nil {
		desired = *s.Spec.Replicas
	}
	ready := s.Status.ReadyReplicas
	return types.WorkloadDetailItem{
		Name:            s.Name,
		Namespace:       s.Namespace,
		Kind:            "StatefulSet",
		Ready:           fmt.Sprintf("%d/%d", ready, desired),
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
		Images:          containerImages(s.Spec.Template.Spec.Containers),
		Labels:          s.Labels,
		CreatedAt:       s.CreationTimestamp.UnixMilli(),
	}
}

func daemonSetToItem(d *appsv1.DaemonSet) types.WorkloadDetailItem {
	desired := d.Status.DesiredNumberScheduled
	ready := d.Status.NumberReady
	return types.WorkloadDetailItem{
		Name:            d.Name,
		Namespace:       d.Namespace,
		Kind:            "DaemonSet",
		Ready:           fmt.Sprintf("%d/%d", ready, desired),
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
		Images:          containerImages(d.Spec.Template.Spec.Containers),
		Labels:          d.Labels,
		CreatedAt:       d.CreationTimestamp.UnixMilli(),
	}
}

func cronJobToItem(c *batchv1.CronJob) types.WorkloadDetailItem {
	suspend := false
	if c.Spec.Suspend != nil {
		suspend = *c.Spec.Suspend
	}
	status := "Active"
	if suspend {
		status = "Suspended"
	}
	return types.WorkloadDetailItem{
		Name:      c.Name,
		Namespace: c.Namespace,
		Kind:      "CronJob",
		Schedule:  c.Spec.Schedule,
		Suspend:   suspend,
		Status:    status,
		Images:    containerImages(c.Spec.JobTemplate.Spec.Template.Spec.Containers),
		Labels:    c.Labels,
		CreatedAt: c.CreationTimestamp.UnixMilli(),
	}
}

func jobInfoToItem(j *k8stypes.JobInfo) types.WorkloadDetailItem {
	return types.WorkloadDetailItem{
		Name:        j.Name,
		Namespace:   j.Namespace,
		Kind:        "Job",
		Status:      j.Status,
		Completions: fmt.Sprintf("%d/%d", j.Succeeded, j.Completions),
		CreatedAt:   j.CreationTimestamp.UnixMilli(),
	}
}
