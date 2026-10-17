package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const minimumManagedVersion = 4

//go:embed *.sql
var migrationFiles embed.FS

type Result struct {
	Applied []string
	Current string
}

// Ensure serializes migration checks across services, applies pending migrations
// when enabled, and rejects startup on drift or a missing migration.
func Ensure(ctx context.Context, db *sql.DB, autoApply bool, lockTimeoutSeconds int) (Result, error) {
	if lockTimeoutSeconds <= 0 {
		lockTimeoutSeconds = 30
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Close()

	locked, err := acquireLock(ctx, conn, lockTimeoutSeconds)
	if err != nil {
		return Result{}, err
	}
	if !locked {
		return Result{}, fmt.Errorf("database migration lock was not acquired within %d seconds", lockTimeoutSeconds)
	}
	defer releaseLock(conn)

	if err := verifyBaseline(ctx, conn); err != nil {
		return Result{}, err
	}
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration (
		version VARCHAR(128) NOT NULL,
		checksum CHAR(64) NOT NULL DEFAULT '',
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (version)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps database migration history'`); err != nil {
		return Result{}, fmt.Errorf("ensure migration history table: %w", err)
	}

	files, err := managedFiles()
	if err != nil {
		return Result{}, err
	}
	var result Result
	for _, name := range files {
		body, err := fs.ReadFile(migrationFiles, name)
		if err != nil {
			return result, fmt.Errorf("read migration %s: %w", name, err)
		}
		checksum := sha256.Sum256([]byte(strings.ReplaceAll(string(body), "\r\n", "\n")))
		expected := hex.EncodeToString(checksum[:])
		stored, found, err := appliedChecksum(ctx, conn, name)
		if err != nil {
			return result, err
		}
		if found {
			if stored != "" && stored != expected {
				return result, fmt.Errorf("migration %s checksum mismatch: applied file was modified", name)
			}
			if stored == "" {
				if err := recordApplied(ctx, conn, name, expected); err != nil {
					return result, err
				}
			}
			result.Current = name
			continue
		}
		if !autoApply {
			return result, fmt.Errorf("pending database migration %s; run axiops-migrate before starting the service", name)
		}
		if err := applyFile(ctx, conn, name, string(body)); err != nil {
			return result, err
		}
		if err := recordApplied(ctx, conn, name, expected); err != nil {
			return result, err
		}
		result.Applied = append(result.Applied, name)
		result.Current = name
	}
	return result, nil
}

func verifyBaseline(ctx context.Context, conn *sql.Conn) error {
	for _, table := range []string{"project", "app_service", "role", "role_permission", "menu", "role_menu"} {
		var count int
		if err := conn.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`, table,
		).Scan(&count); err != nil {
			return fmt.Errorf("inspect baseline table %s: %w", table, err)
		}
		if count != 1 {
			return fmt.Errorf("database baseline is incomplete: table %s is missing; initialize migrations 001-003 first", table)
		}
	}
	return nil
}

func managedFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("list embedded migrations: %w", err)
	}
	var names []string
	seenVersions := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			continue
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version < minimumManagedVersion {
			continue
		}
		if existing, ok := seenVersions[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %03d: %s and %s", version, existing, entry.Name())
		}
		seenVersions[version] = entry.Name()
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func acquireLock(ctx context.Context, conn *sql.Conn, timeout int) (bool, error) {
	var locked sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK('axiops_schema_migration', ?)`, timeout).Scan(&locked); err != nil {
		return false, fmt.Errorf("acquire database migration lock: %w", err)
	}
	return locked.Valid && locked.Int64 == 1, nil
}

func releaseLock(conn *sql.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = conn.ExecContext(ctx, `SELECT RELEASE_LOCK('axiops_schema_migration')`)
}

func appliedChecksum(ctx context.Context, conn *sql.Conn, version string) (string, bool, error) {
	var checksum string
	err := conn.QueryRowContext(ctx, `SELECT checksum FROM schema_migration WHERE version=?`, version).Scan(&checksum)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read migration history for %s: %w", version, err)
	}
	return checksum, true, nil
}

func recordApplied(ctx context.Context, conn *sql.Conn, version, checksum string) error {
	_, err := conn.ExecContext(ctx,
		`INSERT INTO schema_migration (version, checksum, applied_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON DUPLICATE KEY UPDATE checksum=VALUES(checksum)`, version, checksum)
	if err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	return nil
}

func applyFile(ctx context.Context, conn *sql.Conn, name, body string) error {
	statements, err := splitStatements(body)
	if err != nil {
		return fmt.Errorf("parse migration %s: %w", name, err)
	}
	for i, statement := range statements {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply migration %s statement %d: %w", name, i+1, err)
		}
	}
	return nil
}

func splitStatements(body string) ([]string, error) {
	var statements []string
	var current strings.Builder
	var quote rune
	inLineComment := false
	inBlockComment := false
	escaped := false
	runes := []rune(body)

	flush := func() {
		statement := strings.TrimSpace(current.String())
		if statement != "" {
			statements = append(statements, statement)
		}
		current.Reset()
	}

	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				current.WriteRune(ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			current.WriteRune(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if ch == quote {
				if next == quote && quote != '`' {
					current.WriteRune(next)
					i++
					continue
				}
				quote = 0
			}
			continue
		}

		switch {
		case ch == '-' && next == '-' && (i+2 >= len(runes) || runes[i+2] == ' ' || runes[i+2] == '\t'):
			inLineComment = true
			i++
		case ch == '#':
			inLineComment = true
		case ch == '/' && next == '*':
			inBlockComment = true
			i++
		case ch == '\'' || ch == '"' || ch == '`':
			quote = ch
			current.WriteRune(ch)
		case ch == ';':
			flush()
		default:
			current.WriteRune(ch)
		}
	}
	if quote != 0 || inBlockComment {
		return nil, fmt.Errorf("unterminated SQL quote or block comment")
	}
	flush()
	return statements, nil
}
