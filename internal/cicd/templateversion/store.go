// Package templateversion stores immutable template definitions without importing
// service or workflow packages. Both creation and environment upgrades resolve
// the exact same server-side snapshot.
package templateversion

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Ref struct {
	ID       int64 `json:"id"`
	Revision int   `json:"revision"`
}

func (r Ref) Valid() bool          { return r.ID > 0 && r.Revision > 0 }
func (r Ref) SourceCommit() string { return fmt.Sprintf("template:%d@%d", r.ID, r.Revision) }

var referencePattern = regexp.MustCompile(`^template:([1-9][0-9]*)@([1-9][0-9]*)$`)

func ParseSource(source string) (Ref, bool) {
	m := referencePattern.FindStringSubmatch(source)
	if m == nil {
		return Ref{}, false
	}
	id, a := strconv.ParseInt(m[1], 10, 64)
	rev, b := strconv.Atoi(m[2])
	r := Ref{ID: id, Revision: rev}
	return r, a == nil && b == nil && r.Valid()
}

type Query interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
type Version struct {
	Ref
	Snapshot    json.RawMessage `json:"snapshot"`
	Definition  string          `json:"definition"`
	ContentHash string          `json:"content_hash"`
	CreatedAt   time.Time       `json:"created_at"`
	Kind        string          `json:"kind"`
	Status      string          `json:"status"`
}

func Load(ctx context.Context, q Query, ref Ref, active bool) (*Version, error) {
	if !ref.Valid() {
		return nil, fmt.Errorf("模板版本无效")
	}
	v := &Version{Ref: ref}
	err := q.QueryRowContext(ctx, `SELECT v.snapshot,v.definition,v.content_hash,v.created_at,t.kind,t.status FROM delivery_template_version v JOIN delivery_template t ON t.id=v.template_id WHERE v.template_id=? AND v.revision=?`, ref.ID, ref.Revision).Scan(&v.Snapshot, &v.Definition, &v.ContentHash, &v.CreatedAt, &v.Kind, &v.Status)
	if err != nil {
		return nil, fmt.Errorf("模板历史版本不可用，请重新选择已保存的版本")
	}
	if active && v.Status != "active" {
		return nil, fmt.Errorf("模板已归档，不能新增引用；已有环境配置保持不变")
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(v.Definition)))
	if hex.EncodeToString(digest[:]) != v.ContentHash {
		return nil, fmt.Errorf("模板版本完整性校验失败")
	}
	return v, nil
}

func Hash(definition string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(definition)))
	return hex.EncodeToString(sum[:])
}
