package projectenv

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
)

func prepareHelmTemplateRevision(ctx context.Context, tx *sql.Tx, uid, sid, latest int64, ref templateversion.Ref) (int64, error) {
	v, err := templateversion.Load(ctx, tx, ref, true)
	if err != nil {
		return 0, err
	}
	if v.Kind != "helm" {
		return 0, fmt.Errorf("请选择 Helm 模板版本")
	}
	if _, err = helmconfig.LoadChart([]byte(v.Definition)); err != nil {
		return 0, fmt.Errorf("模板 Chart 无效")
	}
	var existing int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM service_revision WHERE service_id=? AND source_commit=? AND content_hash=? AND definition_format='helm_chart' ORDER BY revision_number DESC LIMIT 1`, sid, ref.SourceCommit(), v.ContentHash).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	id, _, err := appsvc.CreateRevisionTx(ctx, tx, &appsvc.ServiceRevision{ServiceID: sid, SourceCommit: ref.SourceCommit(), Definition: v.Definition, DefinitionFormat: "helm_chart", ContentHash: v.ContentHash, CreatedBy: uid, ExpectedRevisionID: &latest})
	return id, err
}
