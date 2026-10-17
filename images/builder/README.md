# AxiOps Builder Image

This directory contains the Dockerfile for `axiops/builder:stable`, the standard CI builder image used by AxiOps to build and push container images inside Kubernetes Jobs.

## What it does

1. Starts a rootless BuildKit daemon inside the container
2. Builds the Dockerfile specified by `DOCKERFILE_PATH` in `BUILD_CONTEXT`
3. Pushes the image to `IMAGE_URL`
4. Extracts the image digest and size
5. Writes a JSON result to `AXIOPS_RESULT_FILE` (usually `/dev/termination-log`)

## Building the image

```bash
docker build -t axiops/builder:stable ./images/builder
docker push axiops/builder:stable
```

Or use a specific registry:

```bash
docker build -t sinozo-registry.ap-southeast-1.cr.aliyuncs.com/platform/builder:v1.0.1 ./images/builder
docker push sinozo-registry.ap-southeast-1.cr.aliyuncs.com/platform/builder:v1.0.1
```

## Environment variables (injected by kubejob_provider)

- `DOCKERFILE_PATH` — relative path to Dockerfile (default: `Dockerfile`)
- `BUILD_CONTEXT` — build context directory (default: `.`)
- `IMAGE_URL` — full image reference to push (e.g. `registry/repo:tag`)
- `IMAGE_REPO` — full output repository, without a tag
- `DOCKER_CONFIG` — directory containing config.json, mounted read-only from a per-build Kubernetes Secret
- `REGISTRY_USERNAME` — registry username (optional, for auth)
- `REGISTRY_PASSWORD` — registry password (optional, for auth)
- `AXIOPS_RESULT_FILE` — where to write the build result JSON (default: `/dev/termination-log`)
- `AXIOPS_BUILD_MODE` — `dockerfile`; other modes fail explicitly
- `AXIOPS_RESULT_SCHEMA` — always `axiops.build/v1`

## Result format

The builder writes a JSON result to `$AXIOPS_RESULT_FILE`:

```json
{
  "schema": "axiops.build/v1",
  "image_digest": "sha256:abc123...",
  "image_size": 123456789
}
```

This is read by `kubejob_provider.go` from the `builder` container's termination log.

## Security

- Runs as non-root user (UID 1000)
- Inherits the official rootless BuildKit image, including its OCI runtime and UID/GID mappings.
- The CI Job currently requires privileged mode and disables the BuildKit process sandbox. Run this only in the dedicated CI namespace.
- Registry passwords are encrypted in the platform and injected using Secrets, not build snapshots or plaintext Job environment variables.

## JAR image mode with a single-stage Dockerfile

Select **JAR 镜像** in the service build dialog. Maven / JDK 17 is filled in automatically:

```sh
mvn -B -ntp -Dmaven.repo.local="${CACHE_DIR:-/workspace/.cache/maven}" clean package
```

The command runs at the repository root and includes tests. It can be edited (for example,
`mvn -pl ad-gateway -am clean package` for a multi-module project). Gradle / JDK 17 is also available.
The service's `build_image` and `build_command` describe Java compilation in this mode.
The Job runs `git-clone`, then `jar-build` as ordered init containers, sharing `/workspace`.
Only after successful compilation does the platform BuildKit container build and push the existing
Dockerfile using the normal digest result contract. The platform builder still receives `dockerfile`
mode, so the existing v1.0.1 image works without rebuilding or publishing a new builder image.

For `COPY target/ad-gateway-1.0.0.jar app.jar`, Maven must produce that exact path under the
selected build context. Check `artifactId`, `version` / `finalName`, and `.dockerignore`.
Multi-module builds may need a module build context or a module prefix in `COPY`.
The JAR must be executable with `java -jar` (for example a repackaged Spring Boot JAR).
Java options and port 9099 in the user's runtime Dockerfile remain runtime settings.

Compilation shares the selected cache and resource requests/limits, but does not mount registry
push credentials. Compilation failures appear under `[jar-build]` and prevent image construction.

Run the real Java smoke test with:

```powershell
powershell -ExecutionPolicy Bypass -File images/builder/smoke-test.ps1 -Jar
```

It compiles `ad-gateway-1.0.0.jar`, uses a single-stage Temurin 17 Dockerfile, pushes to a temporary
local registry, verifies the digest, exports that digest and executes its `java -jar` ENTRYPOINT.
Use `-MavenImage` to select a reachable Maven 3.9 / JDK 17 mirror.

## Dockerfile smoke test

From the repository root:

```powershell
docker build -t axiops/builder:handoff-fix images/builder
powershell -ExecutionPolicy Bypass -File images/builder/smoke-test.ps1
```

This starts an isolated local registry, builds a scratch image using rootless BuildKit,
pushes it, and verifies that the reported manifest digest can be read from the registry.
It does not push to ACR. Configure the CI/CD registry catalog and bind it in the service's
build dialog before testing ACR. After publishing v1.0.1, update the platform builder image;
existing queued builds keep their original execution snapshots.
