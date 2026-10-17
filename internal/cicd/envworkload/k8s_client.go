package envworkload

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RESTConfigFactory resolves a read-only rest.Config for a CICD cluster id.
// It is satisfied by the existing workflow.SQLKubernetesClientFactory so the
// environment view queries the exact same cluster the deployment targets.
type RESTConfigFactory interface {
	RESTConfig(ctx context.Context, clusterID int64) (*rest.Config, error)
}

// k8sClient wraps a typed clientset for read-only namespace-scoped queries.
type k8sClient struct {
	cs kubernetes.Interface
}

func newK8sClient(ctx context.Context, factory RESTConfigFactory, clusterID int64) (*k8sClient, error) {
	if factory == nil {
		return nil, fmt.Errorf("kubernetes client factory is not configured")
	}
	cfg, err := factory.RESTConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	cfg = rest.CopyConfig(cfg)
	cfg.Timeout = 10 * time.Second
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create cluster %d clientset: %w", clusterID, err)
	}
	return &k8sClient{cs: cs}, nil
}

func (c *k8sClient) listDeployments(ctx context.Context, ns string) ([]appsv1.Deployment, error) {
	l, err := c.cs.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listStatefulSets(ctx context.Context, ns string) ([]appsv1.StatefulSet, error) {
	l, err := c.cs.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listDaemonSets(ctx context.Context, ns string) ([]appsv1.DaemonSet, error) {
	l, err := c.cs.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listCronJobs(ctx context.Context, ns string) ([]batchv1.CronJob, error) {
	l, err := c.cs.BatchV1().CronJobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listJobs(ctx context.Context, ns string) ([]batchv1.Job, error) {
	l, err := c.cs.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listPods(ctx context.Context, ns string) ([]corev1.Pod, error) {
	l, err := c.cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listServices(ctx context.Context, ns string) ([]corev1.Service, error) {
	l, err := c.cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listIngresses(ctx context.Context, ns string) ([]networkingv1.Ingress, error) {
	l, err := c.cs.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listPVCs(ctx context.Context, ns string) ([]corev1.PersistentVolumeClaim, error) {
	l, err := c.cs.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listConfigMaps(ctx context.Context, ns string) ([]corev1.ConfigMap, error) {
	l, err := c.cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *k8sClient) listSecrets(ctx context.Context, ns string) ([]corev1.Secret, error) {
	l, err := c.cs.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}
