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

type importConfig struct {
	Legacy   config.MySQLConfig
	Target   config.MySQLConfig
	Provider string `json:",default=legacy_autoops"`
}

type manifest struct {
	Mappings []identity.ImportEntry `json:"mappings"`
}

func main() {
	configFile := flag.String("f", "configs/identity-import.yaml", "database config file")
	manifestFile := flag.String("manifest", "", "reviewed JSON mapping manifest")
	apply := flag.Bool("apply", false, "write validated mappings; default is validation only")
	flag.Parse()
	if *manifestFile == "" {
		log.Fatal("-manifest is required")
	}

	var cfg importConfig
	conf.MustLoad(*configFile, &cfg)
	mappings, err := readManifest(*manifestFile)
	if err != nil {
		log.Fatalf("read manifest: %v", err)
	}
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

	result, err := identity.NewImporter(legacyDB, targetDB).Import(context.Background(), cfg.Provider, mappings, *apply)
	if err != nil {
		log.Fatalf("import identity mappings: %v", err)
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	if !*apply {
		fmt.Fprintln(os.Stderr, "validation completed without writing data; rerun with --apply after review")
	}
}

func readManifest(path string) ([]identity.ImportEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value manifest
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value.Mappings, nil
}
