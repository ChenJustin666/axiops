package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/db"
	"github.com/ChenJustin666/axiops/migrations"
	"github.com/zeromicro/go-zero/core/conf"
)

func main() {
	configFile := flag.String("f", "configs/platform-api.yaml", "config file path")
	checkOnly := flag.Bool("check-only", false, "verify that no database migrations are pending")
	flag.Parse()

	var c config.PlatformAPIConfig
	conf.MustLoad(*configFile, &c)
	mysqlDB, err := db.NewMySQL(&c.MySQL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer mysqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := migrations.Ensure(ctx, mysqlDB, !*checkOnly, c.Migrations.LockTimeoutSeconds)
	if err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	log.Printf("database migrations ready: current=%s applied=%v", result.Current, result.Applied)
}
