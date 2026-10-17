# Implementation Status - 2026-08-12

## Completed

- Scoped RBAC, project/service/environment/build/cluster isolation and
  read-only developer defaults.
- Immutable service revisions, environment variable overrides, workflow DAG,
  production approval, leases, recovery, YAML import and YAML deployment with
  snapshots/rollback.
- Git webhook signature validation, branch filters, delivery ledger and
  duplicate-build idempotency.
- Build logs, SSE streaming, image digest and artifact validation.
- Project-scoped notifications with encrypted credentials, durable outbox,
  retries and Feishu/DingTalk/WeCom/email/webhook channels.
- Host Agent registration, heartbeat, pull polling, command leases, project /
  host-group isolation, artifact checksum verification, versioned execution,
  health checks and rollback.

## Helm Provider Update - 2026-08-18

The official `helm.sh/helm/v3 v3.18.5` SDK is now installed and registered in
the Workflow Worker. Rendering, atomic upgrade/install, durable idempotency,
release revision snapshots and audited rollback are implemented and tested.
Real-cluster upgrade and rollback validation remains an environment acceptance
task; no fake renderer is registered as a production provider.

## Verification

The following passed in this workspace:

```powershell
$env:GOCACHE='E:\axiops\.gocache'
go test -count=1 ./internal/cicd/... ./cmd/...
go vet ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./cmd/axiops-host-agent ./migrations
go build ./cmd/cicd-api ./cmd/workflow-worker ./cmd/axiops-host-agent ./cmd/axiops-migrate
git diff --check
```

No database, Kubernetes cluster or external notification endpoint was used.
