package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestHelmRollbackVerifiesHistoricalInventoryAndImmutableImages(t *testing.T) {
	manifest := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\n  namespace: test\nspec:\n  template:\n    spec:\n      containers:\n      - name: app\n        image: registry.example/app@sha256:" + strings.Repeat("a", 64) + "\n")
	inventory, err := helmManifestInventory(manifest, "test")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(inventory)
	req := HelmRollbackRequest{Namespace: "test", ExpectedInventoryDigest: hex.EncodeToString(hash[:])}
	if err := validateHelmRollbackHistory(req, manifest); err != nil {
		t.Fatal(err)
	}
	tag := []byte(strings.Replace(string(manifest), "@sha256:"+strings.Repeat("a", 64), ":latest", 1))
	if err := validateHelmRollbackHistory(req, tag); err == nil {
		t.Fatal("mutable tag accepted")
	}
	req.ExpectedInventoryDigest = strings.Repeat("b", 64)
	if err := validateHelmRollbackHistory(req, manifest); err == nil {
		t.Fatal("changed inventory accepted")
	}
	req.Namespace = "prod"
	if err := validateHelmRollbackHistory(req, manifest); err == nil {
		t.Fatal("other namespace accepted")
	}
}
