package build

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

func (a *WorkflowAdapter) triggerTest(ctx context.Context, request workflow.BuildTriggerRequest, userID int64) (int64, error) {
	svc, err := a.logic.svcRepo.FindByID(ctx, request.ServiceID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	if svc == nil || svc.Status != "active" {
		return 0, fmt.Errorf("active test service not found")
	}
	allowed, err := a.logic.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "run", ProjectID: svc.ProjectID, ServiceID: svc.ID})
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	if !allowed {
		return 0, fmt.Errorf("test execution permission denied")
	}
	base, err := a.logic.repo.FindConfigByServiceID(ctx, svc.ID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	branch := request.Branch
	repositoryURL, codeSourceID := svc.RepoURL, svc.CodeSourceID
	if branch == "" {
		branch = svc.RepoBranch
	}
	if request.Test.SourceBuildID > 0 {
		source, err := a.logic.repo.FindRunByID(ctx, request.Test.SourceBuildID)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
		}
		if source == nil || source.ProjectID != svc.ProjectID || source.ServiceID != svc.ID || source.Status != StatusSuccess || !validSourceCommit(source.GitCommit) {
			return 0, fmt.Errorf("测试所依赖的构建没有可确认的完整 Commit")
		}
		request.CommitID = source.GitCommit
		branch = source.GitBranch
		var snapshot BuildExecutionSnapshot
		if source.ConfigSnapshot == nil || json.Unmarshal([]byte(*source.ConfigSnapshot), &snapshot) != nil || snapshot.RepositoryURL == "" {
			return 0, fmt.Errorf("测试所依赖的构建缺少代码仓库快照")
		}
		repositoryURL, codeSourceID = snapshot.RepositoryURL, snapshot.CodeSourceID
	}
	if base != nil && base.BranchRules != nil && !branchAllowed(*base.BranchRules, branch) {
		return 0, fmt.Errorf("test branch is excluded by service policy")
	}
	cfg := BuildConfig{Provider: KubeJobProviderName, BuildMode: "test", BuildImage: request.Test.Image, TimeoutSeconds: request.Test.TimeoutSeconds}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 1800
	}
	if base != nil {
		cfg.ResourceCPU = base.ResourceCPU
		cfg.ResourceMemory = base.ResourceMemory
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	// Resolve symlinks too: a repository cannot redirect the working directory.
	cfg.BuildCommand = "set -eu\ncd /workspace\ncd -- " + quote(request.Test.WorkingDir) + "\ncase \"$(pwd -P)\" in /workspace|/workspace/*) ;; *) echo 'test directory escaped workspace'; exit 1;; esac\nexec /bin/sh -euc " + quote(request.Test.Command)
	data, err := json.Marshal(BuildExecutionSnapshot{RepositoryURL: repositoryURL, CodeSourceID: codeSourceID, Config: &cfg})
	if err != nil {
		return 0, err
	}
	snapshot := string(data)
	number, err := a.logic.repo.AllocateBuildNumber(ctx, svc.ID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	id, err := a.logic.repo.CreateRunWithSubmitCommand(ctx, &BuildRun{WorkflowRunID: request.WorkflowRunID, WorkflowNodeKey: request.NodeKey, ServiceID: svc.ID, ProjectID: svc.ProjectID, BuildNumber: number, ConfigSnapshot: &snapshot, TriggerType: "workflow", TriggerUserID: userID, TriggerKey: request.TriggerKey, Status: StatusQueued, Provider: KubeJobProviderName, GitBranch: branch, GitCommit: request.CommitID})
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	return id, nil
}
