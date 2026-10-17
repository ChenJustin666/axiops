# AxiOps Local CI/CD Development

## Required Configuration

The Platform API, CI/CD API and Workflow Worker must use the same MySQL
database and the same `AXIOPS_CREDENTIAL_ENCRYPTION_KEY`. The key encrypts
credential material; changing it prevents decryption of existing values.

Do not commit this secret to YAML. Set it for the current PowerShell session:

```powershell
$env:AXIOPS_CREDENTIAL_ENCRYPTION_KEY = '<generate-and-store-in-your-secret-manager>'
```

For local development, initialize the stable ignored key file once:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\init-local-secrets.ps1
```

The services load keys in this order: `AXIOPS_CREDENTIAL_ENCRYPTION_KEY`, YAML
value, `AXIOPS_CREDENTIAL_ENCRYPTION_KEY_FILE`, then
`local/credential-encryption.key`. Never use `-Rotate` after encrypted
credentials have been created unless a re-encryption migration is planned.

Ensure `configs/platform-api.yaml`, `configs/cicd-api.yaml` and
`configs/workflow-worker.yaml` point to the same test database. Their database
addresses are intentionally not changed by source code because each deployment
environment owns that decision.

The committed local Worker config sets
`DatabaseConfigFile: configs/platform-api.yaml`; this makes the Worker reuse
the Platform API `MySQL` section without copying its password. A production
Worker config may omit `DatabaseConfigFile` and provide its own `MySQL` section.

## Backend Startup

Use the unified local startup script. It initializes the local key, builds the
commands into `bin/` (avoiding Windows Application Control blocking `go run`
temporary executables), verifies migrations, and opens one PowerShell window
for each API and Worker:

```powershell
.\scripts\start-dev.ps1
```

Each window displays its service logs directly. Close the three windows or
press `Ctrl+C` in each window to stop the local services.

For manual troubleshooting, run migration verification first:

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

Then start the services in separate terminals:

```powershell
go run ./cmd/platform-api -f configs/platform-api.yaml
go run ./cmd/cicd-api -f configs/cicd-api.yaml
go run ./cmd/workflow-worker -f configs/workflow-worker.yaml
```

Check endpoints:

```powershell
Invoke-RestMethod http://127.0.0.1:8090/api/v1/system/health
Invoke-RestMethod http://127.0.0.1:8091/api/v1/system/health
```

## Frontend Proxy Contract

The frontend development server uses distinct browser paths because Platform
API and CI/CD API both own `/api/v1` internally:

| Browser base path | Backend target | Backend receives |
| --- | --- | --- |
| `/api/platform/v1` | `http://127.0.0.1:8090` | `/api/v1` |
| `/api/cicd/v1` | `http://127.0.0.1:8091` | `/api/v1` |

New CI/CD UI API modules must call `/api/cicd/v1/...`; platform and RBAC UI
modules must call `/api/platform/v1/...`. Existing legacy proxy paths remain
unchanged during the gradual migration.
