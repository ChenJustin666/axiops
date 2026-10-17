# AxiOps Code Source And AI RCA

## Code source model

`code_source.project_id = 0` is a platform-level connection. It is reusable
from every project for which the caller has `code_source:view` permission.
Project-scoped connections keep their project id and are only visible in that
project. Global connections are created through `POST /api/v1/code-sources`;
the create, update, delete and credential-management operations remain
platform-scoped.

Supported providers:

- GitLab
- Gitea
- GitHub (including Enterprise API `/api/v3`)
- Gitee

Both `http://` and `https://` service URLs are accepted. Plain HTTP should be
limited to trusted internal networks; production deployments should use TLS or
an internal mTLS gateway. Tokens are stored only as encrypted credentials and
are never returned by the API.

The UI path is:

1. Open `CI/CD -> Code Sources` and create one company connection.
2. Test the connection and inspect the repository list.
3. In a project service, select the code source, then select a repository.
4. The repository clone URL and default branch are filled automatically. A
   manually entered URL remains supported for existing services.

Project-scoped credential lists include platform `git_token` metadata so an
existing project-level source can reference the company token without copying
or exposing it. The platform credential page itself still lists only
platform-scoped credentials.

## Repository API

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/code-sources?project_id=0` | `code_source:view` | List platform connections |
| `POST` | `/api/v1/code-sources` | `code_source:create` at platform scope | Create a reusable connection |
| `GET` | `/api/v1/code-sources/{id}/repositories` | `code_source:view` | List safe repository metadata |
| `POST` | `/api/v1/code-sources/{id}/check` | `code_source:test` | Perform a repository-list request to validate credentials and endpoint |

Repository responses contain only id, name, full name, clone URL, default
branch and visibility. They do not contain tokens or credential payloads.

## AI root-cause analysis design

AI RCA is a separate capability from the Git provider. The provider supplies
repository metadata, commits and selected files; the build/workflow subsystem
supplies the failed node, structured status and bounded logs.

Planned endpoint:

`POST /api/v1/builds/{id}/root-cause-analysis`

The server must enforce the caller's project, service, build, code-source and
log permissions before collecting context. The context is limited by byte and
file-count budgets and is redacted before any model call:

- access tokens, passwords, cookies and authorization headers;
- kubeconfig, private keys and Kubernetes Secret values;
- database URLs containing credentials;
- environment values marked secret.

The first analysis layer is deterministic and local. It recognizes image pull
failures, Dockerfile/build-tool failures, test failures, Kubernetes probe
failures, RBAC denials and resource exhaustion. An optional model layer then
returns structured JSON:

- root-cause summary;
- evidence and related log line ranges;
- confidence;
- suggested remediation;
- rollback recommendation and risk.

The UI must show the evidence and confidence separately from the generated
recommendation. AI output is advisory and cannot deploy, approve, rotate
credentials or roll back without an explicit user action and the normal AxiOps
authorization checks. Every request and model response is audit logged with a
redacted context hash.

## Verification record

- No new SQL migration is required for this feature.
- `go test ./pkg/gitprovider ./internal/cicd/codesource` passes.
- REST provider tests cover Gitea repository/file responses and GitHub Enterprise
  API path construction.
- Real GitLab/Gitea connectivity and Kubernetes deployment E2E remain pending
  until an internal endpoint and test token are supplied.

## Troubleshooting a failed connection test

The CI/CD API records the provider category without writing the token:

`local/dev/cicd-api.err.log`

For an internal Git service, first check that the AxiOps host can reach the
endpoint directly. A response such as HTTP 401 proves the network path is
working and means the token is invalid or lacks repository-read permission.
The API returns this as `code=30009` with an actionable authentication message.

The local REST client bypasses `HTTP_PROXY`, `HTTPS_PROXY` and `ALL_PROXY` for
private, loopback and link-local IP addresses. Public Git services continue to
use the configured proxy. Internal DNS names that resolve to private addresses
should also be listed in `NO_PROXY` at process startup.
