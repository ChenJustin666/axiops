package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestManagedFiles(t *testing.T) {
	files, err := managedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 7 || files[0] != "004_authorization_scope.sql" {
		t.Fatalf("unexpected managed migrations: %v", files)
	}
	for _, file := range files {
		if file == "fix_admin_password.sql" {
			t.Fatal("manual repair SQL must never be auto-applied")
		}
	}
}

func TestAllManagedMigrationsCanBeSplit(t *testing.T) {
	files, err := managedFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		body, err := fs.ReadFile(migrationFiles, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		statements, err := splitStatements(string(body))
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if len(statements) == 0 {
			t.Fatalf("migration %s is empty", name)
		}
	}
}

func TestSplitStatements(t *testing.T) {
	sqlText := `
-- comment with ;
CREATE TABLE test (value VARCHAR(64));
INSERT INTO test VALUES ('a;b'), ("c;d"), ('it''s fine');
/* block ; comment */
UPDATE test SET value='escaped\\''quote'; # trailing ; comment
`
	statements, err := splitStatements(sqlText)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 3 {
		t.Fatalf("got %d statements: %v", len(statements), statements)
	}
	if !strings.Contains(statements[1], "a;b") {
		t.Fatalf("quoted semicolon was split: %q", statements[1])
	}
}

func TestSplitStatementsRejectsUnterminatedInput(t *testing.T) {
	for _, input := range []string{"SELECT 'broken;", "SELECT 1; /* broken"} {
		if _, err := splitStatements(input); err == nil {
			t.Fatalf("expected parsing failure for %q", input)
		}
	}
}
