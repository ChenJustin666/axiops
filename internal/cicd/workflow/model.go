package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	DeliveryTypeK8sYAML = "k8s_yaml"
	DeliveryTypeHelm    = "helm"
	DeliveryTypeHost    = "host"

	NodeBuild      = "build"
	NodeYAMLDeploy = "yaml_deploy"
	NodeHelmDeploy = "helm_deploy"
	NodeHostDeploy = "host_deploy"
	NodeTest       = "test"
	NodeApproval   = "approval"
	NodeNotify     = "notify"

	RunPending         = "pending"
	RunPendingApproval = "pending_approval"
	RunRunning         = "running"
	RunSucceeded       = "succeeded"
	RunFailed          = "failed"
	RunCancelling      = "cancelling"
	RunCancelled       = "cancelled"

	DeploymentStrategyRolling   = "rolling"
	DeploymentStrategyCanary    = "canary"
	DeploymentStrategyBlueGreen = "bluegreen"
	TriggerInitialization       = "initialization"
)

type Definition struct {
	DeliveryType string `json:"delivery_type"`
	Nodes        []Node `json:"nodes"`
}

type Workflow struct {
	RequiresEnvironment bool      `json:"requires_environment"`
	TargetEnvironmentID int64     `json:"target_environment_id"`
	ID                  int64     `json:"id"`
	ProjectID           int64     `json:"project_id"`
	Name                string    `json:"name"`
	Code                string    `json:"code"`
	Description         string    `json:"description"`
	DeliveryType        string    `json:"delivery_type"`
	DefinitionVersion   int64     `json:"definition_version"`
	Status              string    `json:"status"`
	Nodes               []Node    `json:"nodes"`
	NodeCount           int       `json:"node_count"`
	CreatedBy           int64     `json:"created_by"`
	UpdatedBy           int64     `json:"updated_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateWorkflowRequest struct {
	TargetEnvironmentID int64      `json:"target_environment_id"`
	Name                string     `json:"name"`
	Code                string     `json:"code"`
	Description         string     `json:"description"`
	Definition          Definition `json:"definition"`
}

type UpdateWorkflowRequest struct {
	TargetEnvironmentID *int64     `json:"target_environment_id,omitempty"`
	ExpectedVersion     int64      `json:"expected_version"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	Definition          Definition `json:"definition"`
}

type RunRequest struct {
	Services              []RunServiceSelection `json:"services,omitempty"`
	SourceSnapshotID      int64                 `json:"source_snapshot_id,omitempty"`
	ExpectedConfigDigest  string                `json:"expected_config_digest,omitempty"`
	ExecutionMode         string                `json:"execution_mode,omitempty"`
	SourceBuildID         int64                 `json:"source_build_id,omitempty"`
	EnvironmentID         int64                 `json:"environment_id"`
	TriggerType           string                `json:"trigger_type"`
	ServiceID             int64                 `json:"service_id,omitempty"`
	ServiceRevisionID     int64                 `json:"service_revision_id,omitempty"`
	SelectWorkflowService bool                  `json:"select_workflow_service,omitempty"`
	UseEnvironmentConfig  bool                  `json:"use_environment_config,omitempty"`
	Branch                string                `json:"branch,omitempty"`
	// DeploymentStrategy is persisted in the run snapshot for audit and provider routing.
	DeploymentStrategy string `json:"deployment_strategy,omitempty"`
}

// Slice order is the submission order for this run only.
type RunServiceSelection struct {
	ServiceID         int64  `json:"service_id"`
	ServiceRevisionID int64  `json:"service_revision_id"`
	Branch            string `json:"branch,omitempty"`
	SourceBuildID     int64  `json:"source_build_id,omitempty"`
}

type WorkflowRun struct {
	RunNumber         int64                `json:"run_number"`
	BuildCommits      []BuildCommitSummary `json:"build_commits,omitempty"`
	ID                int64                `json:"id"`
	WorkflowID        int64                `json:"workflow_id"`
	ProjectID         int64                `json:"project_id"`
	EnvironmentID     int64                `json:"environment_id"`
	DefinitionVersion int64                `json:"definition_version"`
	TriggerType       string               `json:"trigger_type"`
	OperationType     string               `json:"operation_type"`
	SourceSnapshotID  int64                `json:"source_snapshot_id,omitempty"`
	TriggerUserID     int64                `json:"trigger_user_id"`
	TriggerUsername   string               `json:"trigger_username,omitempty"`
	Status            string               `json:"status"`
	ContextSnapshot   json.RawMessage      `json:"context_snapshot"`
	ErrorMessage      string               `json:"error_message"`
	CreatedAt         time.Time            `json:"created_at"`
	StartedAt         *time.Time           `json:"started_at,omitempty"`
	FinishedAt        *time.Time           `json:"finished_at,omitempty"`
	Tasks             []TaskRun            `json:"tasks"`
}

type BuildCommitSummary struct {
	BuildID   int64  `json:"build_id"`
	ServiceID int64  `json:"service_id"`
	Branch    string `json:"branch"`
	Commit    string `json:"commit"`
	Subject   string `json:"subject"`
	Author    string `json:"author"`
	URL       string `json:"url"`
}

type TaskRun struct {
	RetryCount    int             `json:"retry_count"`
	NextRetryAt   *time.Time      `json:"next_retry_at,omitempty"`
	RetryHistory  []TaskRetry     `json:"retry_history,omitempty"`
	ID            int64           `json:"id"`
	WorkflowRunID int64           `json:"workflow_run_id"`
	NodeKey       string          `json:"node_key"`
	NodeType      string          `json:"node_type"`
	DependsOn     []string        `json:"depends_on,omitempty"`
	NodeSnapshot  json.RawMessage `json:"node_snapshot"`
	Status        string          `json:"status"`
	Attempt       int             `json:"attempt"`
	ProviderRunID string          `json:"provider_run_id"`
	LogRef        string          `json:"log_ref"`
	LogText       string          `json:"log_text"`
	ErrorMessage  string          `json:"error_message"`
	LockedBy      string          `json:"locked_by,omitempty"`
	LeaseUntil    *time.Time      `json:"lease_until,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
}

type TaskRetry struct {
	ProviderRunID string    `json:"provider_run_id"`
	ErrorMessage  string    `json:"error_message"`
	FinishedAt    time.Time `json:"finished_at"`
}

type DeliveryProfile struct {
	DeliveryType   string
	DeliveryStatus string
}

var workflowCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func ValidateWorkflowIdentity(name, code string) bool {
	return strings.TrimSpace(name) != "" && len(strings.TrimSpace(name)) <= 128 && workflowCodePattern.MatchString(strings.TrimSpace(code))
}

type Node struct {
	Key            string          `json:"key"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Config         json.RawMessage `json:"config,omitempty"`
	DependsOn      []string        `json:"depends_on,omitempty"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	RetryLimit     int             `json:"retry_limit"`
}

type ValidationIssue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func ValidateDefinition(def Definition) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if !validDeliveryType(def.DeliveryType) {
		issues = append(issues, issue("delivery_type", "invalid_delivery_type", "delivery_type must be k8s_yaml, helm, or host"))
	}
	if len(def.Nodes) == 0 {
		return append(issues, issue("nodes", "empty_nodes", "workflow must contain at least one node"))
	}
	if len(def.Nodes) > 100 {
		issues = append(issues, issue("nodes", "too_many_nodes", "workflow cannot contain more than 100 nodes"))
	}
	nodes := make(map[string]Node, len(def.Nodes))
	for i, node := range def.Nodes {
		field := fmt.Sprintf("nodes[%d]", i)
		key := strings.TrimSpace(node.Key)
		if key == "" || len(key) > 64 {
			issues = append(issues, issue(field+".key", "invalid_key", "node key is required and must be at most 64 characters"))
			continue
		}
		if _, exists := nodes[key]; exists {
			issues = append(issues, issue(field+".key", "duplicate_key", "node key must be unique"))
			continue
		}
		if strings.TrimSpace(node.Name) == "" || len(strings.TrimSpace(node.Name)) > 128 {
			issues = append(issues, issue(field+".name", "invalid_name", "node name is required and must be at most 128 characters"))
		}
		if !validNodeType(node.Type) {
			issues = append(issues, issue(field+".type", "invalid_node_type", "unsupported workflow node type"))
		}
		if !compatible(def.DeliveryType, node.Type) {
			issues = append(issues, issue(field+".type", "incompatible_node_type", "deploy node is incompatible with project delivery type"))
		}
		if node.TimeoutSeconds < 0 || node.TimeoutSeconds > 86400 {
			issues = append(issues, issue(field+".timeout_seconds", "invalid_timeout", "timeout must be between 0 and 86400 seconds"))
		}
		if node.RetryLimit < 0 || node.RetryLimit > 10 {
			issues = append(issues, issue(field+".retry_limit", "invalid_retry", "retry limit must be between 0 and 10"))
		}
		if node.RetryLimit > 0 && node.Type != NodeBuild {
			issues = append(issues, issue(field+".retry_limit", "unsupported_retry", "自动重试目前仅支持构建节点；部署失败请检查状态后重新执行"))
		}
		if node.Type == NodeTest {
			var cfg struct {
				ServiceID int64  `json:"service_id"`
				Image     string `json:"image"`
				Command   string `json:"command"`
			}
			if json.Unmarshal(node.Config, &cfg) != nil || cfg.ServiceID < 0 || strings.TrimSpace(cfg.Image) == "" || strings.TrimSpace(cfg.Command) == "" || len(cfg.Command) > 16384 {
				issues = append(issues, issue(field+".config", "invalid_test_config", "测试节点需要执行镜像和测试命令，工作目录为仓库内相对路径"))
			}
		}
		nodes[key] = node
	}
	if len(nodes) == 0 {
		return issues
	}
	graph := make(map[string][]string, len(nodes))
	for key, node := range nodes {
		seenDependencies := make(map[string]struct{}, len(node.DependsOn))
		for _, dependency := range node.DependsOn {
			dependency = strings.TrimSpace(dependency)
			if dependency == "" {
				issues = append(issues, issue("nodes."+key+".depends_on", "empty_dependency", "dependency key cannot be empty"))
				continue
			}
			if _, exists := nodes[dependency]; !exists {
				issues = append(issues, issue("nodes."+key+".depends_on", "missing_dependency", "dependency does not exist"))
				continue
			}
			if dependency == key {
				issues = append(issues, issue("nodes."+key+".depends_on", "self_dependency", "node cannot depend on itself"))
			}
			if _, exists := seenDependencies[dependency]; exists {
				issues = append(issues, issue("nodes."+key+".depends_on", "duplicate_dependency", "dependency key must be unique"))
			}
			seenDependencies[dependency] = struct{}{}
			graph[key] = append(graph[key], dependency)
		}
		if len(node.Config) > 0 && !json.Valid(node.Config) {
			issues = append(issues, issue("nodes."+key+".config", "invalid_config", "node config must be valid JSON"))
		}
	}
	if cycle(graph, nodes) {
		issues = append(issues, issue("nodes", "cycle_detected", "workflow dependencies must form an acyclic graph"))
	}
	deployCount := 0
	orderedCount := 0
	orders := make(map[int]bool)
	serviceIDs := make(map[string]int64, len(nodes))
	for key, node := range nodes {
		var config struct {
			ServiceID  int64 `json:"service_id"`
			StartOrder int   `json:"start_order"`
		}
		_ = json.Unmarshal(node.Config, &config)
		serviceIDs[key] = config.ServiceID
		if node.Type != NodeYAMLDeploy && node.Type != NodeHelmDeploy && node.Type != NodeHostDeploy {
			if config.StartOrder != 0 {
				issues = append(issues, issue("nodes."+key+".config.start_order", "invalid_start_order", "only deployment nodes can have a start order"))
			}
			continue
		}
		deployCount++
		if config.StartOrder > 0 {
			orderedCount++
			orders[config.StartOrder] = true
		} else if config.StartOrder < 0 {
			issues = append(issues, issue("nodes."+key+".config.start_order", "invalid_start_order", "start order must be positive"))
		}
	}
	if orderedCount > 0 {
		for key, node := range nodes {
			if node.Type != NodeBuild && node.Type != NodeTest && node.Type != NodeYAMLDeploy && node.Type != NodeHelmDeploy && node.Type != NodeHostDeploy {
				issues = append(issues, issue("nodes."+key+".type", "invalid_start_order", "ordered deployment supports service nodes only"))
			}
		}
		if deployCount < 2 || orderedCount != deployCount || len(orders) != deployCount {
			issues = append(issues, issue("nodes", "invalid_start_order", "every deployment needs a unique start order"))
		}
		for order := 1; order <= deployCount; order++ {
			if !orders[order] {
				issues = append(issues, issue("nodes", "invalid_start_order", "deployment start order must be consecutive from 1"))
				break
			}
		}
		for key, node := range nodes {
			for _, dependency := range node.DependsOn {
				if serviceIDs[key] > 0 && serviceIDs[dependency] > 0 && serviceIDs[key] != serviceIDs[dependency] {
					issues = append(issues, issue("nodes."+key+".depends_on", "ordered_cross_service_dependency", "deployment start order cannot include cross-service success dependencies"))
				}
			}
		}
	}
	return issues
}

func issue(field, code, message string) ValidationIssue {
	return ValidationIssue{Field: field, Code: code, Message: message}
}

func validDeliveryType(value string) bool {
	return value == DeliveryTypeK8sYAML || value == DeliveryTypeHelm || value == DeliveryTypeHost
}

func validNodeType(value string) bool {
	switch value {
	case NodeBuild, NodeYAMLDeploy, NodeHelmDeploy, NodeHostDeploy, NodeTest, NodeApproval, NodeNotify:
		return true
	default:
		return false
	}
}

func compatible(deliveryType, nodeType string) bool {
	switch nodeType {
	case NodeYAMLDeploy:
		return deliveryType == DeliveryTypeK8sYAML
	case NodeHelmDeploy:
		return deliveryType == DeliveryTypeHelm
	case NodeHostDeploy:
		return deliveryType == DeliveryTypeHost
	default:
		return true
	}
}

func cycle(graph map[string][]string, nodes map[string]Node) bool {
	state := make(map[string]uint8, len(nodes))
	var visit func(string) bool
	visit = func(key string) bool {
		if state[key] == 1 {
			return true
		}
		if state[key] == 2 {
			return false
		}
		state[key] = 1
		for _, dependency := range graph[key] {
			if visit(dependency) {
				return true
			}
		}
		state[key] = 2
		return false
	}
	for key := range nodes {
		if visit(key) {
			return true
		}
	}
	return false
}
