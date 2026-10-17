package projectenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

func (h *Handler) validateDraftAssets(ctx context.Context, tx *sql.Tx, repo *Repository, uid, projectID, eid int64, old, next serviceassets.Settings) error {
	if err := serviceassets.Validate(next); err != nil {
		return err
	}
	var environmentCode string
	if err := tx.QueryRowContext(ctx, `SELECT code FROM project_environment WHERE id=? AND project_id=?`, eid, projectID).Scan(&environmentCode); err != nil {
		return err
	}
	for _, secret := range next.Secrets {
		if secret.CredentialID == 0 {
			continue
		}
		var owner int64
		var kind, data string
		if tx.QueryRowContext(ctx, `SELECT project_id,type,data FROM credential WHERE id=?`, secret.CredentialID).Scan(&owner, &kind, &data) != nil || kind != "secret" || owner != 0 && owner != projectID {
			return fmt.Errorf("凭据不可用或不属于当前项目")
		}
		allowed, err := h.logic.authorizer.Authorize(ctx, uid, authz.Request{Resource: "credential", Action: "view", ProjectID: owner, EnvironmentID: eid, Environment: environmentCode})
		if err != nil || !allowed {
			return fmt.Errorf("没有查看目标凭据的权限")
		}
		if repo.decryptor == nil {
			return fmt.Errorf("凭据解密服务不可用")
		}
		plain, err := repo.decryptor.Decrypt(data)
		var fields map[string]interface{}
		if err != nil || json.Unmarshal([]byte(plain), &fields) != nil {
			return fmt.Errorf("凭据无法读取")
		}
		if _, ok := fields[secret.CredentialKey].(string); !ok {
			return fmt.Errorf("凭据 #%d 缺少所选文本字段", secret.CredentialID)
		}
	}
	for _, before := range old.Volumes {
		if before.ExistingClaim != "" {
			continue
		}
		for _, after := range next.Volumes {
			if before.Name == after.Name && (before.Capacity != after.Capacity || before.StorageClass != after.StorageClass || before.AccessMode != after.AccessMode || before.ExistingClaim != after.ExistingClaim) {
				return fmt.Errorf("已有新建 PVC 不可在此改写容量、StorageClass 或访问模式；请使用独立扩容流程")
			}
		}
	}
	return nil
}
