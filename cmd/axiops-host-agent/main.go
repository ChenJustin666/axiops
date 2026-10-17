package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/hostagent"
	"github.com/zeromicro/go-zero/core/conf"
)

type agentConfig struct {
	ServerURL           string
	RegistrationToken   string `json:",optional"`
	AgentTokenFile      string `json:",default=./data/agent.token"`
	AssetRef            string
	Name                string
	WorkDir             string `json:",default=./data"`
	CACertFile          string `json:",optional"`
	AllowInsecureHTTP   bool   `json:",default=false"`
	PollIntervalSeconds int    `json:",default=5"`
	MaxArtifactBytes    int64  `json:",default=2147483648"`
}

func main() {
	configFile := flag.String("f", "configs/host-agent.yaml", "config file path")
	flag.Parse()
	var config agentConfig
	conf.MustLoad(*configFile, &config)
	client, err := hostagent.NewClient(hostagent.ClientConfig{
		ServerURL: config.ServerURL, RegistrationToken: config.RegistrationToken,
		AgentTokenFile: config.AgentTokenFile, AssetRef: config.AssetRef, Name: config.Name,
		WorkDir: config.WorkDir, CACertFile: config.CACertFile, AllowInsecureHTTP: config.AllowInsecureHTTP,
		PollInterval: time.Duration(config.PollIntervalSeconds) * time.Second, MaxArtifactBytes: config.MaxArtifactBytes,
	})
	if err != nil {
		log.Fatalf("host Agent configuration is invalid: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("AxiOps host Agent started")
	if err := client.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("AxiOps host Agent stopped unexpectedly: %v", err)
	}
}
