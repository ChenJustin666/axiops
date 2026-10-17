package helmconfig

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
)

const SharedKind = "AxiOpsHelmSharedValues"
const SharedName = "axiops-helm-shared-values"

type Shared struct {
	Values   string `json:"values"`
	Revision int64  `json:"revision"`
}

func LoadShared(ctx context.Context, q Query, eid int64) (Shared, error) {
	s := Shared{Values: "{}\n"}
	err := q.QueryRowContext(ctx, `SELECT manifest,revision FROM environment_resource WHERE environment_id=? AND name=? AND kind=?`, eid, SharedName, SharedKind).Scan(&s.Values, &s.Revision)
	if err == sql.ErrNoRows {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if _, err = ParseValues(s.Values); err != nil {
		return s, fmt.Errorf("环境共享 values 格式无效")
	}
	return s, nil
}

func ValidValuesFilePath(p string) bool {
	return p != "" && path.Clean(p) == p && p != "." && p != ".." && !strings.HasPrefix(p, "../") && !strings.HasPrefix(p, "/") && !strings.ContainsAny(p, "\\:\x00\r\n") && (strings.HasSuffix(strings.ToLower(p), ".yaml") || strings.HasSuffix(strings.ToLower(p), ".yml"))
}
