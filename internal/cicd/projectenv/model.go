package projectenv

import (
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"regexp"
	"strings"
	"time"
)

const (
	DeliveryTypeK8sYAML = "k8s_yaml"
	DeliveryTypeHelm    = "helm"
	DeliveryTypeHost    = "host"

	EnvironmentTypeK8sNamespace = "k8s_namespace"
	EnvironmentTypeHostGroup    = "host_group"

	BindingStatusValidated         = "validated"
	BindingStatusPendingValidation = "pending_validation"
)

var (
	codePattern      = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	namespacePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	variablePattern  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,127}$`)
)

type ProjectEnvironment struct {
	ID                  int64     `json:"id"`
	ProjectID           int64     `json:"project_id"`
	Name                string    `json:"name"`
	Code                string    `json:"code"`
	EnvironmentType     string    `json:"environment_type"`
	ClusterID           int64     `json:"cluster_id"`
	Namespace           string    `json:"namespace"`
	HostGroupRef        string    `json:"host_group_ref"`
	BindingStatus       string    `json:"binding_status"`
	IsProduction        bool      `json:"is_production"`
	DeployApproval      bool      `json:"deploy_approval"`
	Status              string    `json:"status"`
	CreatedBy           int64     `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	initialHelmServices []InitialHelmService
	preview             bool
	initialConfigs      []*ServiceConfig
}

type InitialHelmService struct {
	ServiceID  int64               `json:"service_id"`
	RevisionID int64               `json:"revision_id"`
	Helm       helmconfig.Settings `json:"helm"`
}

type CreateEnvironmentRequest struct {
	HelmServices   []InitialHelmService `json:"helm_services,omitempty"`
	preview        bool
	configs        []*ServiceConfig
	failure        error
	Name           string `json:"name"`
	Code           string `json:"code"`
	ClusterID      int64  `json:"cluster_id"`
	Namespace      string `json:"namespace"`
	HostGroupRef   string `json:"host_group_ref"`
	IsProduction   bool   `json:"is_production"`
	DeployApproval bool   `json:"deploy_approval"`
}

// These standard codes are reserved for production. Custom environment names,
// namespaces and code substrings never determine the production classification.
func IsProductionCode(code string) bool {
	return code == "prod" || code == "production"
}

// CloneEnvironmentRequest describes the target runtime context. The source
// environment is taken from the URL and production environments are never
// eligible as either source or target.
type CloneEnvironmentRequest struct {
	Name           string `json:"name"`
	Code           string `json:"code"`
	ClusterID      int64  `json:"cluster_id"`
	Namespace      string `json:"namespace"`
	HostGroupRef   string `json:"host_group_ref"`
	IsProduction   bool   `json:"is_production"`
	DeployApproval bool   `json:"deploy_approval"`
}

type ProjectDeliveryProfile struct {
	DeliveryType   string
	DeliveryStatus string
}

type ClusterTarget struct {
	ID     int64
	Status string
}

type VariableSet struct {
	ID            int64          `json:"id"`
	ProjectID     int64          `json:"project_id"`
	EnvironmentID int64          `json:"environment_id"`
	ServiceID     int64          `json:"service_id"`
	Name          string         `json:"name"`
	Scope         string         `json:"scope"`
	Revision      int64          `json:"revision"`
	Items         []VariableItem `json:"items"`
	CreatedBy     int64          `json:"created_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type VariableItem struct {
	ID           int64  `json:"id"`
	VariableKey  string `json:"key"`
	Value        string `json:"value,omitempty"`
	CredentialID int64  `json:"credential_id,omitempty"`
	IsSecret     bool   `json:"is_secret"`
}

type VariableItemInput struct {
	VariableKey  string `json:"key"`
	Value        string `json:"value"`
	CredentialID int64  `json:"credential_id"`
	IsSecret     bool   `json:"is_secret"`
}

type UpdateVariableSetRequest struct {
	Revision int64               `json:"revision"`
	Items    []VariableItemInput `json:"items"`
}

type ServiceTarget struct {
	ID           int64
	ProjectID    int64
	DeliveryType string
}

type EnvironmentServiceOverride struct {
	ID                   int64     `json:"id"`
	ProjectEnvironmentID int64     `json:"project_environment_id"`
	ServiceID            int64     `json:"service_id"`
	ServiceRevisionID    int64     `json:"service_revision_id"`
	ImageRef             string    `json:"image_ref"`
	Replicas             *int      `json:"replicas,omitempty"`
	Revision             int64     `json:"revision"`
	UpdatedBy            int64     `json:"updated_by"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type SaveServiceOverrideRequest struct {
	ExpectedRevision  int64  `json:"expected_revision"`
	ServiceRevisionID int64  `json:"service_revision_id"`
	ImageRef          string `json:"image_ref"`
	Replicas          *int   `json:"replicas"`
}

type EnvironmentResource struct {
	ID            int64     `json:"id"`
	EnvironmentID int64     `json:"environment_id"`
	Name          string    `json:"name"`
	Kind          string    `json:"kind"`
	Manifest      string    `json:"manifest"`
	Revision      int64     `json:"revision"`
	CreatedBy     int64     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SaveEnvironmentResourceRequest struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Manifest         string `json:"manifest"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func ResolveEnvironmentTarget(deliveryType string, req CreateEnvironmentRequest) (environmentType, bindingStatus string, valid bool) {
	switch deliveryType {
	case DeliveryTypeK8sYAML, DeliveryTypeHelm:
		valid = req.ClusterID > 0 && namespacePattern.MatchString(strings.TrimSpace(req.Namespace)) && strings.TrimSpace(req.HostGroupRef) == ""
		return EnvironmentTypeK8sNamespace, BindingStatusValidated, valid
	case DeliveryTypeHost:
		valid = req.ClusterID == 0 && strings.TrimSpace(req.Namespace) == "" && strings.TrimSpace(req.HostGroupRef) != ""
		return EnvironmentTypeHostGroup, BindingStatusPendingValidation, valid
	default:
		return "", "", false
	}
}

func ValidateEnvironmentIdentity(name, code string) bool {
	return strings.TrimSpace(name) != "" && len(strings.TrimSpace(name)) <= 64 && codePattern.MatchString(strings.TrimSpace(code))
}

func ValidateVariableItems(items []VariableItemInput) bool {
	if len(items) > 200 {
		return false
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.VariableKey)
		if !variablePattern.MatchString(key) {
			return false
		}
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
		if item.IsSecret {
			if item.CredentialID <= 0 || item.Value != "" {
				return false
			}
		} else if item.CredentialID != 0 || len(item.Value) > 65535 {
			return false
		}
	}
	return true
}

func IsEnvironmentCompatible(environmentType, bindingStatus, deliveryType string) bool {
	if bindingStatus != BindingStatusValidated {
		return false
	}
	switch deliveryType {
	case DeliveryTypeK8sYAML, DeliveryTypeHelm:
		return environmentType == EnvironmentTypeK8sNamespace
	case DeliveryTypeHost:
		return environmentType == EnvironmentTypeHostGroup
	default:
		return false
	}
}

func ValidateServiceOverrideRequest(req SaveServiceOverrideRequest) bool {
	if req.ExpectedRevision < 0 || req.ServiceRevisionID <= 0 || len(strings.TrimSpace(req.ImageRef)) > 512 {
		return false
	}
	return req.Replicas == nil || (*req.Replicas >= 0 && *req.Replicas <= 10000)
}
