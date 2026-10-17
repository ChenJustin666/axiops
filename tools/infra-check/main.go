package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/conf"
)

type config struct {
	Mysql struct {
		DataSource string
	}
	Cache struct {
		Host string
		Pass string
	}
}

func main() {
	configPath := flag.String("config", "application/portal-rpc/etc/portal.yaml", "portal RPC config path")
	database := flag.String("database", "", "database name override")
	listTables := flag.Bool("list-tables", false, "list authorization and project-related tables")
	flag.Parse()

	var cfg config
	if err := conf.Load(*configPath, &cfg); err != nil {
		if fallbackErr := loadConfigFallback(*configPath, &cfg); fallbackErr != nil {
			exitf("load config: %v; fallback: %v", err, fallbackErr)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := checkMySQL(ctx, cfg.Mysql.DataSource, *database, *listTables); err != nil {
		exitf("mysql check failed: %v", err)
	}
	if err := checkRedis(ctx, cfg.Cache.Host, cfg.Cache.Pass); err != nil {
		exitf("redis check failed: %v", err)
	}
}

func checkMySQL(ctx context.Context, dataSource, database string, listTables bool) error {
	dsn, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return fmt.Errorf("parse data source: %w", err)
	}
	if database != "" {
		dsn.DBName = database
	}

	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	var currentDatabase string
	var version string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE(), VERSION()").Scan(&currentDatabase, &version); err != nil {
		return err
	}

	var tableCount int
	if err := db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()",
	).Scan(&tableCount); err != nil {
		return err
	}

	fmt.Printf("mysql: ok database=%s version=%s tables=%d\n", currentDatabase, version, tableCount)
	if !listTables {
		return nil
	}

	rows, err := db.QueryContext(ctx, `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND (
    table_name LIKE 'auth\_%'
    OR table_name LIKE 'sys\_%'
    OR table_name LIKE '%project%'
    OR table_name LIKE '%environment%'
  )`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	sort.Strings(tables)
	for _, table := range tables {
		fmt.Printf("table: %s\n", table)
	}
	return nil
}

func checkRedis(ctx context.Context, address, password string) error {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
	})
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		if password != "" && strings.Contains(err.Error(), "no password is set") {
			client.Close()
			client = redis.NewClient(&redis.Options{Addr: address})
			defer client.Close()
			if retryErr := client.Ping(ctx).Err(); retryErr != nil {
				return retryErr
			}
		} else {
			return err
		}
	}

	size, err := client.DBSize(ctx).Result()
	if err != nil {
		return err
	}
	fmt.Printf("redis: ok keys=%d\n", size)
	return nil
}

func loadConfigFallback(path string, cfg *config) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	mysqlBlock, err := extractSection(raw, "Mysql")
	if err != nil {
		return fmt.Errorf("mysql block: %w", err)
	}
	dataSource, err := extractValueFromBlock(mysqlBlock, "DataSource")
	if err != nil {
		return fmt.Errorf("mysql data source: %w", err)
	}
	cfg.Mysql.DataSource = dataSource

	cacheBlock, err := extractSection(raw, "Cache")
	if err != nil {
		return fmt.Errorf("cache block: %w", err)
	}
	host, err := extractValueFromBlock(cacheBlock, "Host")
	if err != nil {
		return fmt.Errorf("cache host: %w", err)
	}
	pass, err := extractValueFromBlock(cacheBlock, "Pass")
	if err != nil {
		return fmt.Errorf("cache pass: %w", err)
	}
	cfg.Cache.Host = host
	cfg.Cache.Pass = pass
	return nil
}

func extractSection(raw []byte, header string) (string, error) {
	lines := bytes.Split(raw, []byte("\n"))
	headerLine := []byte(header + ":")
	for i, line := range lines {
		if bytes.Equal(bytes.TrimSpace(line), headerLine) {
			var block []string
			for j := i + 1; j < len(lines); j++ {
				next := lines[j]
				if len(next) == 0 {
					block = append(block, "")
					continue
				}
				if next[0] == ' ' || next[0] == '\t' {
					block = append(block, string(next))
					continue
				}
				break
			}
			return strings.Join(block, "\n"), nil
		}
	}
	return "", fmt.Errorf("%s not found", header)
}

func extractValueFromBlock(block, key string) (string, error) {
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+":") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, key+":"))
			return strings.Trim(strings.TrimSpace(value), `"'`), nil
		}
	}
	return "", fmt.Errorf("%s not found", key)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
