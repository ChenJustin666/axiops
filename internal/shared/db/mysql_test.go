package db

import (
	"context"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
	"os"
	"testing"
	"time"
)

func TestMySQLSessionClock(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL integration test not configured")
	}
	var cfg config.PlatformAPIConfig
	if conf.Load(path, &cfg) != nil {
		t.Fatal("configuration unavailable")
	}
	database, err := NewMySQL(&cfg.MySQL)
	if err != nil {
		t.Fatal("database unavailable")
	}
	defer database.Close()
	// Verify separate physical connections, not just the first pooled session.
	for i := 0; i < 2; i++ {
		conn, err := database.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		var now, utc time.Time
		var zone string
		if err = conn.QueryRowContext(context.Background(), "SELECT NOW(),UTC_TIMESTAMP(),@@session.time_zone").Scan(&now, &utc, &zone); err != nil {
			t.Fatal(err)
		}
		if zone != "+00:00" || !now.Equal(utc) || time.Since(now) > 15*time.Second || time.Until(now) > 15*time.Second {
			t.Fatalf("database clock inconsistent: zone=%s offset=%v", zone, time.Since(now))
		}
	}
}
