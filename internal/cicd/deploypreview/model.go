package deploypreview

import "github.com/ChenJustin666/axiops/internal/cicd/workflow"

type Request struct {
	ServiceID         int64
	ServiceRevisionID int64
	ReleaseName       string
}

type Target struct {
	ProjectID, EnvironmentID, ServiceID, ServiceRevisionID, ClusterID  int64
	EnvironmentCode, EnvironmentType, BindingStatus, EnvironmentStatus string
	ServiceCode, DeliveryType, Namespace                               string
}

type Summary struct {
	Create    int `json:"create"`
	Update    int `json:"update"`
	Delete    int `json:"delete"`
	Unchanged int `json:"unchanged"`
}

type Result struct {
	DeliveryType      string                        `json:"delivery_type"`
	ProjectID         int64                         `json:"project_id"`
	EnvironmentID     int64                         `json:"environment_id"`
	ServiceID         int64                         `json:"service_id"`
	ServiceRevisionID int64                         `json:"service_revision_id"`
	ClusterID         int64                         `json:"cluster_id"`
	Namespace         string                        `json:"namespace"`
	ReleaseName       string                        `json:"release_name,omitempty"`
	Changes           []workflow.YAMLResourceChange `json:"changes"`
	Summary           Summary                       `json:"summary"`
	Values            map[string]interface{}        `json:"values,omitempty"`
}
