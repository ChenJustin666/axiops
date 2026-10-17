package main

import (
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestBuildResourcesHaveNoImplicitConfigDefaults(t *testing.T) {
	for _, input := range []string{"{}", "ResourceCPU: \"\"\nResourceMemory: \"\"\n"} {
		var defaults config.BuildDefaultsConfig
		if err := conf.LoadFromYamlBytes([]byte(input), &defaults); err != nil {
			t.Fatal(err)
		}
		if defaults.ResourceCPU != "" || defaults.ResourceMemory != "" {
			t.Fatalf("empty resources received hidden defaults: %+v", defaults)
		}
	}
	var defaults config.BuildDefaultsConfig
	if err := conf.LoadFromYamlBytes([]byte("ResourceCPU: 500m\nResourceMemory: 2Gi\n"), &defaults); err != nil {
		t.Fatal(err)
	}
	if defaults.ResourceCPU != "500m" || defaults.ResourceMemory != "2Gi" {
		t.Fatalf("explicit resources changed: %+v", defaults)
	}
}
