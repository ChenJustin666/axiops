package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ChenJustin666/axiops/internal/platform/identity"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/db"
	"github.com/zeromicro/go-zero/core/conf"
)

type inventoryConfig struct {
	Legacy   config.MySQLConfig
	Target   config.MySQLConfig
	Provider string `json:",default=legacy_autoops"`
	Limit    int    `json:",default=100"`
}

func main() {
	configFile := flag.String("f", "configs/identity-inventory.yaml", "config file path")
	flag.Parse()

	var cfg inventoryConfig
	conf.MustLoad(*configFile, &cfg)
	legacyDB, err := db.NewMySQL(&cfg.Legacy)
	if err != nil {
		log.Fatalf("connect legacy database: %v", err)
	}
	defer legacyDB.Close()
	targetDB, err := db.NewMySQL(&cfg.Target)
	if err != nil {
		log.Fatalf("connect target database: %v", err)
	}
	defer targetDB.Close()

	report, err := identity.NewInventory(legacyDB, targetDB).Report(context.Background(), cfg.Provider, cfg.Limit)
	if err != nil {
		log.Fatalf("build identity inventory: %v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatalf("write identity inventory: %v", err)
	}
	fmt.Fprintln(os.Stderr, "identity inventory completed without writing data")
}
