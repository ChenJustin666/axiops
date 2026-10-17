// Package testschema installs incremental schemas only in disposable test databases.
package testschema

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func EnvironmentYAML(t *testing.T, db *sql.DB) {
	t.Helper()
	var name string
	if err := db.QueryRow(`SELECT DATABASE()`).Scan(&name); err != nil || !strings.HasPrefix(name, "axiops_") || !strings.Contains(name, "_test_") {
		t.Fatal("environment schema migration requires a disposable test database")
	}
	raw, err := os.ReadFile("../../../migrations/046_environment_yaml_revisions.sql")
	if err != nil {
		t.Fatal(err)
	}
	var column int
	if err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='environment_service_override' AND column_name='environment_yaml_revision_id'`).Scan(&column); err != nil {
		t.Fatal(err)
	}
	for _, q := range strings.Split(string(raw), ";") {
		q = strings.TrimSpace(q)
		if q == "" || column > 0 && strings.HasPrefix(q, "ALTER TABLE") {
			continue
		}
		if _, err = db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
}
