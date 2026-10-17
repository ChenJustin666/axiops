package config

import (
	"github.com/go-sql-driver/mysql"
	"testing"
	"time"
)

func TestMySQLClockUsesUTCAtBothBoundaries(t *testing.T) {
	cfg, err := mysql.ParseDSN((&MySQLConfig{User: "test", Host: "127.0.0.1", Port: 3306, Database: "test", Charset: "utf8mb4"}).DSN())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ParseTime || cfg.Loc != time.UTC || cfg.Params["time_zone"] != "'+00:00'" {
		t.Fatalf("driver and session must both use UTC")
	}
}
