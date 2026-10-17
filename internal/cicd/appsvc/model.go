package appsvc

import (
	"encoding/json"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
)

const (
	DeliveryTypeUnclassified = "unclassified"
	DeliveryTypeK8sYAML      = "k8s_yaml"
	DeliveryTypeHelm         = "helm"
	DeliveryTypeHost         = "host"
	ServiceStatusActive      = "active"
	ServiceStatusInactive    = "inactive"
	ServiceStatusDeleted     = "deleted"
)

func IsActiveDeliveryType(deliveryType string) bool {
	switch deliveryType {
	case DeliveryTypeK8sYAML, DeliveryTypeHelm, DeliveryTypeHost:
		return true
	default:
		return false
	}
}

func LegacyDeployType(deliveryType string) string {
	switch deliveryType {
	case DeliveryTypeK8sYAML:
		return "k8s"
	case DeliveryTypeHelm:
		return "helm"
	case DeliveryTypeHost:
		return "host"
	default:
		return ""
	}
}

func DefinitionFormat(deliveryType string) string {
	switch deliveryType {
	case DeliveryTypeK8sYAML:
		return "k8s_yaml"
	case DeliveryTypeHelm:
		return "helm_chart"
	case DeliveryTypeHost:
		return "host_spec"
	default:
		return ""
	}
}

func IsValidSourceType(sourceType string) bool {
	switch sourceType {
	case "git", "manual", "template":
		return true
	default:
		return false
	}
}

// Creation sources are narrower than stored sources: legacy custom Helm
// services remain editable, but new ones start from a template or Git.
func IsValidCreationSource(deliveryType, sourceType string) bool {
	return IsValidSourceType(sourceType) && (deliveryType != DeliveryTypeHelm || sourceType != "manual")
}

func IsValidServiceStatus(status string) bool {
	return status == ServiceStatusActive || status == ServiceStatusInactive
}

// AppService 应用服务模型
type AppService struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	Code              string          `json:"code"`
	Description       string          `json:"description"`
	ProjectID         int64           `json:"project_id"`
	Language          string          `json:"language"`
	Framework         string          `json:"framework"`
	RepoURL           string          `json:"repo_url"`
	RepoBranch        string          `json:"repo_branch"`
	CodeSourceID      int64           `json:"code_source_id"`
	RegistryID        int64           `json:"registry_id"`
	ImageName         string          `json:"image_name"`
	DockerfilePath    string          `json:"dockerfile_path"`
	BuildContext      string          `json:"build_context"`
	DeployType        string          `json:"deploy_type"`
	DeliveryType      string          `json:"delivery_type"`
	SourceType        string          `json:"source_type"`
	SourceRef         json.RawMessage `json:"source_ref"`
	CurrentRevisionID int64           `json:"current_revision_id"`
	ComponentCount    int             `json:"component_count"`
	Status            string          `json:"status"`
	OwnerUserID       int64           `json:"owner_user_id"`
	CreatedBy         int64           `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// CreateAppServiceRequest 创建服务
type CreateAppServiceRequest struct {
	Name           string          `json:"name" validate:"required,max=128"`
	Code           string          `json:"code" validate:"required,max=64"`
	Description    string          `json:"description"`
	ProjectID      int64           `json:"project_id" validate:"required"`
	Language       string          `json:"language"`
	Framework      string          `json:"framework"`
	RepoURL        string          `json:"repo_url" validate:"required"`
	RepoBranch     string          `json:"repo_branch"`
	CodeSourceID   int64           `json:"code_source_id"`
	RegistryID     int64           `json:"registry_id"`
	ImageName      string          `json:"image_name"`
	DockerfilePath string          `json:"dockerfile_path"`
	BuildContext   string          `json:"build_context"`
	DeployType     string          `json:"deploy_type"`
	SourceType     string          `json:"source_type"`
	SourceRef      json.RawMessage `json:"source_ref"`
}

type ProjectDeliveryProfile struct {
	DeliveryType   string
	DeliveryStatus string
}

type ServiceRevision struct {
	ID                 int64                  `json:"id"`
	ServiceID          int64                  `json:"service_id"`
	RevisionNumber     int64                  `json:"revision_number"`
	SourceCommit       string                 `json:"source_commit"`
	Definition         string                 `json:"definition"`
	DefinitionFormat   string                 `json:"definition_format"`
	ContentHash        string                 `json:"content_hash"`
	CreatedBy          int64                  `json:"created_by"`
	CreatedAt          time.Time              `json:"created_at"`
	ComponentCount     int                    `json:"-"`
	ExpectedRevisionID *int64                 `json:"-"`
	RepositoryBinding  *HelmRepositoryBinding `json:"-"`
}

type HelmRepositoryBinding struct {
	CodeSourceID int64  `json:"code_source_id"`
	RepoPath     string `json:"repo_path"`
	Ref          string `json:"ref"`
	ChartPath    string `json:"chart_path"`
	CommitID     string `json:"commit_id"`
	RevisionID   int64  `json:"revision_id"`
}

type CreateServiceRevisionRequest struct {
	Template           *templateversion.Ref   `json:"template,omitempty"`
	SourceCommit       string                 `json:"source_commit"`
	Definition         string                 `json:"definition" validate:"required"`
	ComponentCount     int                    `json:"-"`
	ExpectedRevisionID *int64                 `json:"expected_revision_id,omitempty"`
	RepositoryBinding  *HelmRepositoryBinding `json:"-"`
}

// UpdateAppServiceRequest 更新服务
type UpdateAppServiceRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	RepoURL        string `json:"repo_url"`
	RepoBranch     string `json:"repo_branch"`
	CodeSourceID   *int64 `json:"code_source_id"`
	RegistryID     *int64 `json:"registry_id"`
	ImageName      string `json:"image_name"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildContext   string `json:"build_context"`
	DeployType     string `json:"deploy_type"`
	Status         string `json:"status"`
	OwnerUserID    *int64 `json:"owner_user_id"`
}
