package envworkload

import "k8s.io/apimachinery/pkg/types"

type WorkloadStatusReason struct {
	Reason string `json:"reason"`
	Pods   int    `json:"pods"`
}

// WorkloadItem is one workload row shown in the environment view.
type WorkloadItem struct {
	RestartAllowed   bool                   `json:"restart_allowed"`
	UID              types.UID              `json:"-"`
	ControllerStatus string                 `json:"controller_status,omitempty"`
	StatusReasons    []WorkloadStatusReason `json:"status_reasons,omitempty"`
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace"`
	Kind             string                 `json:"kind"`
	Ready            string                 `json:"ready"`
	DesiredReplicas  int32                  `json:"desired_replicas"`
	ReadyReplicas    int32                  `json:"ready_replicas"`
	Images           []string               `json:"images"`
	Schedule         string                 `json:"schedule,omitempty"`
	Suspend          bool                   `json:"suspend,omitempty"`
	Status           string                 `json:"status,omitempty"`
	Completions      string                 `json:"completions,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`
	CreatedAt        int64                  `json:"created_at"`
}

// PodItem is one pod row.
type PodItem struct {
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	Phase      string   `json:"phase"`
	Status     string   `json:"status"`
	Ready      string   `json:"ready"`
	Restarts   int32    `json:"restarts"`
	NodeName   string   `json:"node_name"`
	PodIP      string   `json:"pod_ip"`
	Images     []string `json:"images"`
	Containers []string `json:"containers"`
	CreatedAt  int64    `json:"created_at"`
}

// ServiceItem is one service row.
type ServiceItem struct {
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	Type       string   `json:"type"`
	ClusterIP  string   `json:"cluster_ip"`
	ExternalIP []string `json:"external_ip,omitempty"`
	Ports      []string `json:"ports"`
	CreatedAt  int64    `json:"created_at"`
}

// IngressItem is one ingress row.
type IngressItem struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Class     string   `json:"class,omitempty"`
	Hosts     []string `json:"hosts"`
	Address   []string `json:"address,omitempty"`
	CreatedAt int64    `json:"created_at"`
}

// PVCItem is one persistent volume claim row.
type PVCItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"access_modes"`
	StorageClass string `json:"storage_class"`
	VolumeName   string `json:"volume_name"`
	CreatedAt    int64  `json:"created_at"`
}

// ConfigItem is one ConfigMap/Secret row.
type ConfigItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Type      string `json:"type,omitempty"`
	Keys      int    `json:"keys"`
	CreatedAt int64  `json:"created_at"`
}
