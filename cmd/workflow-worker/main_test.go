package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/config"
)

type fakeWorkflowTicker struct {
	cancel context.CancelFunc
	ticks  int
	err    error
}

type fakeBuildProcessor struct {
	cancel    context.CancelFunc
	processed bool
	calls     int
}

func (f *fakeBuildProcessor) ProcessOne(context.Context, string) (bool, error) {
	f.calls++
	if f.cancel != nil {
		f.cancel()
	}
	return f.processed, nil
}

func (f *fakeWorkflowTicker) Tick(context.Context, string) error {
	f.ticks++
	if f.cancel != nil {
		f.cancel()
	}
	return f.err
}

func TestRunWorkerTicksImmediatelyAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &fakeWorkflowTicker{cancel: cancel}
	runWorker(ctx, worker, "worker-1", time.Hour, nil)
	if worker.ticks != 1 {
		t.Fatalf("ticks=%d", worker.ticks)
	}
}

func TestRunWorkerReportsTickError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &fakeWorkflowTicker{err: errors.New("database unavailable")}
	var reported error
	runWorker(ctx, worker, "worker-1", time.Hour, func(err error) {
		reported = err
		cancel()
	})
	if reported == nil {
		t.Fatal("tick error was not reported")
	}
}

func TestValidateConfigRejectsUnsafeDefaults(t *testing.T) {
	c := config.WorkflowWorkerConfig{
		CredentialEncryptionKey: "key", MySQL: config.MySQLConfig{Database: "axiops"},
		Worker: config.WorkerConfig{PollIntervalMS: 2000, LeaseSeconds: 120, BatchSize: 20},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	c.CredentialEncryptionKey = ""
	if err := validateConfig(c); err == nil {
		t.Fatal("missing encryption key was accepted")
	}
}

func TestRunBuildWorkerStopsAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &fakeBuildProcessor{cancel: cancel, processed: true}
	runBuildWorker(ctx, worker, "worker-1:build", time.Hour, nil)
	if worker.calls != 1 {
		t.Fatalf("calls=%d", worker.calls)
	}
}

func TestValidateConfigRequiresBuildTargetWhenEnabled(t *testing.T) {
	c := config.WorkflowWorkerConfig{
		CredentialEncryptionKey: "key", MySQL: config.MySQLConfig{Database: "axiops"},
		Worker: config.WorkerConfig{PollIntervalMS: 2000, LeaseSeconds: 120, BatchSize: 20},
		Build: config.BuildWorkerConfig{
			Enabled: true, PollIntervalMS: 1000, MaxAttempts: 3, RetryAfterSecs: 5, StaleAfterSecs: 300,
		},
	}
	if err := validateConfig(c); err == nil {
		t.Fatal("enabled build worker without cluster target was accepted")
	}
	c.Build.ClusterID, c.Build.Namespace = 2, "axiops-cicd"
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
}

func TestInheritDatabaseConfigLoadsSharedMySQL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "platform-api.yaml")
	data := []byte("MySQL:\n  Host: db.internal\n  Port: 3307\n  User: axiops\n  Password: secret\n  Database: axiops_shared\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	c := config.WorkflowWorkerConfig{
		DatabaseConfigFile: path,
		MySQL:              config.MySQLConfig{Host: "127.0.0.1", Database: "wrong_database"},
	}
	if err := inheritDatabaseConfig(&c); err != nil {
		t.Fatal(err)
	}
	if c.MySQL.Host != "db.internal" || c.MySQL.Port != 3307 || c.MySQL.Database != "axiops_shared" {
		t.Fatalf("unexpected inherited MySQL config: host=%s port=%d database=%s", c.MySQL.Host, c.MySQL.Port, c.MySQL.Database)
	}
}

func TestInheritDatabaseConfigRejectsMissingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "platform-api.yaml")
	if err := os.WriteFile(path, []byte("MySQL:\n  Host: db.internal\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := config.WorkflowWorkerConfig{DatabaseConfigFile: path}
	if err := inheritDatabaseConfig(&c); err == nil {
		t.Fatal("shared MySQL config without a database was accepted")
	}
}
