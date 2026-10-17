package appsvc

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// Logic 应用服务业务逻辑
type Logic struct {
	repo       *Repository
	authorizer authz.Checker
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer}
}

func (l *Logic) List(ctx context.Context, userID, projectID int64, keyword, status string, page, pageSize int) ([]AppService, int64, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 500 {
		pageSize = 500
	}
	scope, err := l.authorizer.Scopes(ctx, userID, "service", "view")
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	services, total, err := l.repo.List(ctx, scope, projectID, keyword, status, page, pageSize)
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	return services, total, errcode.CodeSuccess
}

func (l *Logic) GetByID(ctx context.Context, userID, id int64) (*AppService, int) {
	s, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if s == nil {
		return nil, errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "view", ProjectID: s.ProjectID, ServiceID: s.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	return s, errcode.CodeSuccess
}

func (l *Logic) Create(ctx context.Context, req *CreateAppServiceRequest, createdBy int64) (int64, int) {
	if req == nil || req.ProjectID <= 0 || createdBy <= 0 || !validServiceIdentity(req.Name, req.Code) {
		return 0, errcode.CodeParamInvalid
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.RepoURL = strings.TrimSpace(req.RepoURL)
	req.RepoBranch = strings.TrimSpace(req.RepoBranch)
	req.SourceType = strings.TrimSpace(req.SourceType)
	allowed, err := l.authorizer.Authorize(ctx, createdBy, authz.Request{Resource: "service", Action: "create", ProjectID: req.ProjectID})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if !allowed {
		return 0, errcode.CodeNoPermission
	}
	profile, err := l.repo.FindProjectDeliveryProfile(ctx, req.ProjectID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if profile == nil {
		return 0, errcode.CodeProjectNotFound
	}
	if profile.DeliveryStatus != "active" || !IsActiveDeliveryType(profile.DeliveryType) {
		return 0, errcode.CodeParamInvalid
	}
	legacyDeployType := LegacyDeployType(profile.DeliveryType)
	if req.DeployType != "" && req.DeployType != legacyDeployType {
		return 0, errcode.CodeParamInvalid
	}
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "git"
	}
	if !IsValidCreationSource(profile.DeliveryType, sourceType) || (sourceType == "git" && strings.TrimSpace(req.RepoURL) == "") {
		return 0, errcode.CodeParamInvalid
	}
	if req.CodeSourceID > 0 {
		allowed, err := l.codeSourceAllowed(ctx, createdBy, req.CodeSourceID, req.ProjectID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}
	if len(req.SourceRef) > 0 && !json.Valid(req.SourceRef) {
		return 0, errcode.CodeParamInvalid
	}
	if req.RegistryID != 0 {
		if code := l.validateRegistryBinding(ctx, req.RegistryID, req.ProjectID); code != errcode.CodeSuccess {
			return 0, code
		}
	}
	// 检查 project+code 唯一
	existing, err := l.repo.FindByProjectAndCode(ctx, req.ProjectID, req.Code)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if existing != nil {
		return 0, errcode.CodeServiceCodeExists
	}

	s := &AppService{
		Name:           req.Name,
		Code:           req.Code,
		Description:    req.Description,
		ProjectID:      req.ProjectID,
		Language:       req.Language,
		Framework:      req.Framework,
		RepoURL:        req.RepoURL,
		RepoBranch:     req.RepoBranch,
		CodeSourceID:   req.CodeSourceID,
		RegistryID:     req.RegistryID,
		ImageName:      req.ImageName,
		DockerfilePath: req.DockerfilePath,
		BuildContext:   req.BuildContext,
		DeployType:     legacyDeployType,
		DeliveryType:   profile.DeliveryType,
		SourceType:     sourceType,
		SourceRef:      req.SourceRef,
		Status:         ServiceStatusActive,
		OwnerUserID:    createdBy,
		CreatedBy:      createdBy,
	}
	if s.RepoBranch == "" {
		s.RepoBranch = "main"
	}
	if s.DockerfilePath == "" {
		s.DockerfilePath = "Dockerfile"
	}
	if s.BuildContext == "" {
		s.BuildContext = "."
	}
	if len(s.SourceRef) == 0 {
		s.SourceRef = json.RawMessage("{}")
	}

	id, err := l.repo.Create(ctx, s)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

func (l *Logic) Update(ctx context.Context, userID, id int64, req *UpdateAppServiceRequest) int {
	if req == nil || id <= 0 || userID <= 0 {
		return errcode.CodeParamInvalid
	}
	s, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if s == nil {
		return errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "update", ProjectID: s.ProjectID, ServiceID: s.ID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}

	if s.Status == ServiceStatusDeleted {
		return errcode.CodeParamInvalid
	}
	if req.Name != "" {
		name := strings.TrimSpace(req.Name)
		if name != s.Name && !validServiceName(name) {
			return errcode.CodeParamInvalid
		}
		s.Name = name
	}
	if req.Description != "" {
		s.Description = req.Description
	}
	if req.Language != "" {
		s.Language = req.Language
	}
	if req.Framework != "" {
		s.Framework = req.Framework
	}
	if req.RepoURL != "" {
		s.RepoURL = strings.TrimSpace(req.RepoURL)
	}
	if req.RepoBranch != "" {
		s.RepoBranch = strings.TrimSpace(req.RepoBranch)
	}
	if req.CodeSourceID != nil {
		if *req.CodeSourceID > 0 {
			allowed, err := l.codeSourceAllowed(ctx, userID, *req.CodeSourceID, s.ProjectID)
			if err != nil {
				return errcode.CodeDBError
			}
			if !allowed {
				return errcode.CodeNoPermission
			}
		}
		s.CodeSourceID = *req.CodeSourceID
	}
	if req.RegistryID != nil {
		if code := l.validateRegistryBinding(ctx, *req.RegistryID, s.ProjectID); code != errcode.CodeSuccess {
			return code
		}
		s.RegistryID = *req.RegistryID
	}
	if req.ImageName != "" {
		s.ImageName = req.ImageName
	}
	if req.DockerfilePath != "" {
		s.DockerfilePath = req.DockerfilePath
	}
	if req.BuildContext != "" {
		s.BuildContext = req.BuildContext
	}
	if req.DeployType != "" && req.DeployType != s.DeployType {
		return errcode.CodeParamInvalid
	}
	if req.Status != "" {
		if !IsValidServiceStatus(req.Status) {
			return errcode.CodeParamInvalid
		}
		s.Status = req.Status
	}
	if existing, err := l.repo.FindByProjectAndCode(ctx, s.ProjectID, s.Code); err != nil {
		return errcode.CodeDBError
	} else if existing != nil && existing.ID != s.ID {
		return errcode.CodeParamInvalid
	}
	if req.OwnerUserID != nil {
		s.OwnerUserID = *req.OwnerUserID
	}

	if err := l.repo.Update(ctx, s); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) validateRegistryBinding(ctx context.Context, registryID, projectID int64) int {
	if registryID < 0 {
		return errcode.CodeParamInvalid
	}
	if registryID == 0 {
		return errcode.CodeSuccess
	}
	allowed, err := l.repo.RegistryBelongsToProject(ctx, registryID, projectID)
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func (l *Logic) codeSourceAllowed(ctx context.Context, userID, sourceID, projectID int64) (bool, error) {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "code_source", Action: "view", ProjectID: projectID})
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}
	belongs, err := l.repo.CodeSourceBelongsToProject(ctx, sourceID, projectID)
	return belongs, err
}

func (l *Logic) CreateRevision(ctx context.Context, userID, serviceID int64, req *CreateServiceRevisionRequest) (*ServiceRevision, int) {
	if req == nil || userID <= 0 || serviceID <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	if req.ExpectedRevisionID != nil && *req.ExpectedRevisionID < 0 {
		return nil, errcode.CodeParamInvalid
	}
	s, err := l.repo.FindByID(ctx, serviceID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if s == nil {
		return nil, errcode.CodeServiceNotFound
	}
	if s.Status != ServiceStatusActive {
		return nil, errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "update", ProjectID: s.ProjectID, ServiceID: s.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	definition := strings.TrimSpace(req.Definition)
	format := DefinitionFormat(s.DeliveryType)
	if req.Template != nil {
		if format != "helm_chart" || !req.Template.Valid() {
			return nil, errcode.CodeParamInvalid
		}
		scope, e := l.authorizer.Scopes(ctx, userID, "build_template", "view")
		if e != nil {
			return nil, errcode.CodeDBError
		}
		if !scope.HasAnyAllow() {
			return nil, errcode.CodeNoPermission
		}
		v, e := templateversion.Load(ctx, l.repo.db, *req.Template, true)
		if e != nil || v.Kind != "helm" {
			return nil, errcode.CodeServiceDefinitionInvalid
		}
		if definition != "" && !helmconfig.SameFiles(definition, v.Definition) {
			return nil, errcode.CodeServiceDefinitionInvalid
		}
		definition = strings.TrimSpace(v.Definition)
		req.SourceCommit = req.Template.SourceCommit()
	} else if format == "helm_chart" && strings.HasPrefix(req.SourceCommit, "template:") {
		// Template provenance is assigned only after resolving a stored version.
		return nil, errcode.CodeParamInvalid
	}
	if definition == "" || format == "" {
		return nil, errcode.CodeParamInvalid
	}
	if format == "k8s_yaml" {
		if err := validateYAMLDefinition(definition); err != nil {
			return nil, errcode.CodeServiceDefinitionInvalid
		}
	}
	if format == "helm_chart" {
		chart, err := helmconfig.LoadChart([]byte(definition))
		if err == nil {
			err = helmconfig.ValidateChartSyntax(chart)
		}
		if err != nil {
			return nil, errcode.CodeServiceDefinitionInvalid
		}
	}
	digest := sha256.Sum256([]byte(definition))
	revision := &ServiceRevision{
		ServiceID:          serviceID,
		SourceCommit:       strings.TrimSpace(req.SourceCommit),
		Definition:         definition,
		DefinitionFormat:   format,
		ContentHash:        hex.EncodeToString(digest[:]),
		CreatedBy:          userID,
		ComponentCount:     req.ComponentCount,
		ExpectedRevisionID: req.ExpectedRevisionID,
		RepositoryBinding:  req.RepositoryBinding,
	}
	if req.ComponentCount < 0 {
		return nil, errcode.CodeParamInvalid
	}
	id, revisionNumber, err := l.repo.CreateRevision(ctx, revision)
	if errors.Is(err, errRevisionChanged) {
		return nil, errcode.CodeServiceRevisionChanged
	}
	if err != nil {
		return nil, errcode.CodeDBError
	}
	revision.ID = id
	revision.RevisionNumber = revisionNumber
	return revision, errcode.CodeSuccess
}

func (l *Logic) ListRevisions(ctx context.Context, userID, serviceID int64, latest ...bool) ([]ServiceRevision, int) {
	s, err := l.repo.FindByID(ctx, serviceID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if s == nil {
		return nil, errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "view", ProjectID: s.ProjectID, ServiceID: s.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	revisions, err := l.repo.ListRevisions(ctx, serviceID, latest...)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return revisions, errcode.CodeSuccess
}

func (l *Logic) Delete(ctx context.Context, userID, id int64) int {
	_, code := l.DeleteChecked(ctx, userID, id)
	return code
}

var serviceCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var serviceNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*$`)

const serviceNameHint = "服务名称须为 1–128 位小写字母、数字及合法的 .、_、- 分隔符，以字母或数字开头和结尾，例如 risk_server"

func validServiceName(name string) bool {
	name = strings.TrimSpace(name)
	return len(name) <= 128 && serviceNamePattern.MatchString(name)
}

func ValidServiceIdentity(name, code string) bool { return validServiceIdentity(name, code) }

func validServiceIdentity(name, code string) bool {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	return validServiceName(name) && serviceCodePattern.MatchString(code)
}
