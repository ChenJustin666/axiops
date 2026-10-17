package testschema

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TemplateVersions(t *testing.T, db *sql.DB) {
	t.Helper()
	var name string
	if err := db.QueryRow(`SELECT DATABASE()`).Scan(&name); err != nil || !strings.HasPrefix(name, "axiops_") || !strings.Contains(name, "_test_") {
		t.Fatal("template migrations require a disposable test database")
	}
	for _, file := range []string{"039_delivery_templates.sql", "052_delivery_template_versions.sql"} {
		raw, err := os.ReadFile("../../../migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		for _, query := range strings.Split(string(raw), ";") {
			query = strings.TrimSpace(query)
			if query != "" {
				if _, err = db.Exec(query); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
