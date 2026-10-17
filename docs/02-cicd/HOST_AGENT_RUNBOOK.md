# AxiOps Host Agent Runbook

## Database

Apply migrations in numeric order through `024_cicd_host_rolling_batch.sql`.
The recommended release commands are:

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

For production, set `Migrations.AutoApply: false`. Run the migration command in
the release job, verify `-check-only`, then start the API and Worker.

## Registration

An operator with `host_agent:create` creates a one-time registration token:

```http
POST /api/v1/projects/{project_id}/host-agent-registration-tokens
Content-Type: application/json

{"host_group_ref":"payment-test","expires_minutes":30,"max_uses":1}
```

The token is hashed in the database and is returned only once. Start the Agent
on the target host with `configs/host-agent.yaml`:

```powershell
go run ./cmd/axiops-host-agent -f configs/host-agent.yaml
```

Set `RegistrationToken` only on first startup. After registration, the Agent
token is stored in `AgentTokenFile` with mode `0600`; subsequent starts do not
need the registration token.

Production requires HTTPS. Use `CACertFile` for a private CA. Keep
`AllowInsecureHTTP: false` outside local development.

## Host Service Revision

Host revisions are imported from a Git `HostService` definition instead of
accepting script text in the import request:

```yaml
apiVersion: axiops.io/v1alpha1
kind: HostService
metadata:
  name: payment
spec:
  os: linux
  artifact:
    source: build
  scripts:
    deploy: scripts/deploy.sh
    health: scripts/health.sh
    rollback: scripts/rollback.sh
  timeoutSeconds: 900
```

For a pinned external artifact, set `source: static`, an absolute HTTPS `url`,
and its 64-character lowercase `sha256`. With `source: build`, the workflow
uses the successful host artifact produced by the matching build node.

Call `/api/v1/services/{id}/host-import/preview` with the definition and three
script paths, then send its `commit_id` as `expected_commit` to
`/api/v1/services/{id}/host-import`. The API reads the immutable commit again,
freezes the three script bodies and their content hash in a `host_spec`
revision, and never returns script bodies in the preview.

Linux definitions accept only `.sh`; Windows definitions accept only `.ps1`.
Deployment selects online Agents from the same project and host group with the
exact target OS. Older host revisions without `os` must be imported again.
The workflow fails closed when the environment is not validated, no matching
Agent is online, or the checksum is invalid.

The Agent accepts only absolute HTTPS artifacts, enforces the configured size
limit, verifies SHA-256 before execution, runs scripts in a versioned release
directory, and exposes only `AXIOPS_RELEASE_DIR`, `AXIOPS_ARTIFACT_PATH` and
`AXIOPS_COMMAND_ID` in addition to a minimal system environment.

## Failure Handling

- An offline Agent is not selected. Expired leases can be reclaimed safely.
- Download failures remove the temporary artifact file.
- Deploy or health-check failures run the rollback script in a bounded timeout.
- Logs are capped at the last 1 MiB per command.
