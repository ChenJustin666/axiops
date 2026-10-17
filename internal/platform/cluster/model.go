package cluster

import "time"

// Cluster 集群模型
type Cluster struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Code          string     `json:"code"`
	Description   string     `json:"description"`
	Provider      string     `json:"provider"`
	Region        string     `json:"region"`
	Version       string     `json:"version"`
	APIServer     string     `json:"api_server"`
	Kubeconfig    string     `json:"-"` // 不在列表中返回
	Status        string     `json:"status"`
	NodeCount     int        `json:"node_count"`
	CPUTotal      string     `json:"cpu_total"`
	MemoryTotal   string     `json:"memory_total"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
	CreatedBy     int64      `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CreateClusterRequest 创建集群
type CreateClusterRequest struct {
	Name        string `json:"name" validate:"required,max=64"`
	Code        string `json:"code" validate:"required,max=64"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
	Region      string `json:"region"`
	APIServer   string `json:"api_server" validate:"required"`
	Kubeconfig  string `json:"kubeconfig" validate:"required"`
}

// UpdateClusterRequest 更新集群
type UpdateClusterRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	APIServer   string `json:"api_server"`
	Kubeconfig  string `json:"kubeconfig"`
}
