#!/bin/bash
# entrypoint.sh — AxiOps Builder Entrypoint
# Builds Dockerfile with BuildKit, pushes to registry, writes result to termination-log.

set -euo pipefail

# Read configuration from environment
DOCKERFILE_PATH="${DOCKERFILE_PATH:-Dockerfile}"
BUILD_CONTEXT="${BUILD_CONTEXT:-.}"
IMAGE_URL="${IMAGE_URL:-}"
IMAGE_REPO="${IMAGE_REPO:-}"
RESULT_FILE="${AXIOPS_RESULT_FILE:-/dev/termination-log}"
BUILD_MODE="${AXIOPS_BUILD_MODE:-dockerfile}"
RESULT_SCHEMA="${AXIOPS_RESULT_SCHEMA:-axiops.build/v1}"

echo "=== AxiOps Builder ==="
echo "Build mode: ${BUILD_MODE}"
echo "Dockerfile: ${DOCKERFILE_PATH}"
echo "Context: ${BUILD_CONTEXT}"
echo "Image: ${IMAGE_URL}"

# Other modes need their own builder and result contract.
if [[ "${BUILD_MODE}" != "dockerfile" ]]; then
  echo "ERROR: This builder supports dockerfile mode only" >&2
  exit 1
fi

# Validate required variables
if [[ -z "${IMAGE_URL}" ]]; then
  echo "ERROR: IMAGE_URL is not set" >&2
  exit 1
fi

if [[ ! -f "${BUILD_CONTEXT}/${DOCKERFILE_PATH}" ]]; then
  echo "ERROR: Dockerfile not found at ${BUILD_CONTEXT}/${DOCKERFILE_PATH}" >&2
  exit 1
fi

# ENOSPC from clone/unshare may indicate a zero/exhausted namespace quota.
# The Job must not silently alter this node-level security setting.
fail_startup() {
  local message="AXIOPS_BUILD_ERROR: $*"
  echo "${message}" >&2
  printf '%s\n' "${message}" > "${RESULT_FILE}" 2>/dev/null || true
  exit 1
}
if [[ -r /proc/sys/user/max_user_namespaces ]]; then
  USER_NAMESPACE_LIMIT=$(cat /proc/sys/user/max_user_namespaces)
  echo "Node user.max_user_namespaces=${USER_NAMESPACE_LIMIT}"
  if [[ "${USER_NAMESPACE_LIMIT}" == "0" ]]; then
    fail_startup "ROOTLESS_USER_NAMESPACE_DISABLED: user.max_user_namespaces=0. The CI node disables user namespaces required by RootlessKit. Ask the node administrator to enable a nonzero quota on CI nodes; changing build CPU/memory or registry credentials will not fix this."
  fi
fi
if [[ "${EUID}" != "0" && -r /proc/sys/kernel/unprivileged_userns_clone ]] && [[ "$(cat /proc/sys/kernel/unprivileged_userns_clone)" == "0" ]]; then
  fail_startup "ROOTLESS_USER_NAMESPACE_DISABLED: kernel.unprivileged_userns_clone=0. The CI node blocks unprivileged user namespaces required by RootlessKit."
fi

# Start BuildKit daemon in the background (rootless mode)
echo "Starting BuildKit daemon (rootless)..."
BUILD_STATE_DIR="${HOME}/.local/share/axiops-buildkit"
mkdir -p "${BUILD_STATE_DIR}"
export BUILDKIT_HOST="unix://${BUILD_STATE_DIR}/buildkitd.sock"
BUILDKIT_PID=""
cleanup() {
  if [[ -n "${BUILDKIT_PID}" ]]; then
    kill "${BUILDKIT_PID}" 2>/dev/null || true
    wait "${BUILDKIT_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

rootlesskit \
  --net=host \
  --copy-up=/etc \
  --state-dir="${BUILD_STATE_DIR}/rootlesskit" \
  buildkitd \
  --addr="${BUILDKIT_HOST}" \
  --root="${BUILD_STATE_DIR}/root" \
  --oci-worker-no-process-sandbox \
  >/tmp/buildkitd.log 2>&1 &

BUILDKIT_PID=$!
echo "BuildKit daemon launching (PID: ${BUILDKIT_PID})"

# Wait for BuildKit to be ready
for i in {1..30}; do
  if ! kill -0 "${BUILDKIT_PID}" 2>/dev/null; then
    BUILDKIT_EXIT=0
    wait "${BUILDKIT_PID}" || BUILDKIT_EXIT=$?
    cat /tmp/buildkitd.log >&2
    fail_startup "BUILDKIT_START_FAILED: BuildKit exited before becoming ready (exit=${BUILDKIT_EXIT}). See the daemon error above. For fork/unshare ENOSPC, check the node user.max_user_namespaces quota before assuming disk exhaustion."
  fi
  if buildctl debug info >/dev/null 2>&1; then
    echo "BuildKit is ready."
    break
  fi
  if [[ $i -eq 30 ]]; then
    cat /tmp/buildkitd.log >&2
    fail_startup "BUILDKIT_START_TIMEOUT: BuildKit did not become ready within 30 seconds. See the daemon log above."
  fi
  sleep 1
done

# Prefer the read-only Docker config mounted from the per-build Secret.
if [[ ! -f "${DOCKER_CONFIG:-${HOME}/.docker}/config.json" ]] && [[ -n "${REGISTRY_USERNAME:-}" ]] && [[ -n "${REGISTRY_PASSWORD:-}" ]]; then
  export DOCKER_CONFIG="${HOME}/.docker"
  mkdir -p "${DOCKER_CONFIG}"
  umask 077
  REGISTRY_HOST="${IMAGE_URL%%/*}"
  AUTH_VALUE=$(printf '%s' "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" | base64 | tr -d '\n')
  jq -n --arg host "${REGISTRY_HOST}" --arg auth "${AUTH_VALUE}" \
    '{auths: {($host): {auth: $auth}}}' > "${DOCKER_CONFIG}/config.json"
  unset AUTH_VALUE
fi

# Build and push the image
echo "Building image: ${IMAGE_URL}"
buildctl build \
  --frontend dockerfile.v0 \
  --local context="${BUILD_CONTEXT}" \
  --local dockerfile="${BUILD_CONTEXT}" \
  --opt filename="${DOCKERFILE_PATH}" \
  --output type=image,name="${IMAGE_URL}",push=true \
  --metadata-file /tmp/axiops-build-metadata.json \
  --progress=plain

echo "Image pushed successfully: ${IMAGE_URL}"

# BuildKit reports the pushed manifest/index digest. config.digest identifies
# the image config blob and cannot be used as a deployable image reference.
IMAGE_DIGEST=$(jq -er '."containerimage.digest"' /tmp/axiops-build-metadata.json)
AUTH_ARGS=()
if [[ -f "${DOCKER_CONFIG:-${HOME}/.docker}/config.json" ]]; then
  AUTH_ARGS=(--authfile "${DOCKER_CONFIG:-${HOME}/.docker}/config.json")
fi
IMAGE_REPOSITORY="${IMAGE_URL%:*}"
INSPECT_OUTPUT=$(skopeo inspect "${AUTH_ARGS[@]}" --raw "docker://${IMAGE_REPOSITORY}@${IMAGE_DIGEST}")
IMAGE_SIZE=$(printf '%s' "${INSPECT_OUTPUT}" | jq '[.config.size // 0, (.layers[]?.size // 0)] | add')

# If still no digest, fail
if [[ ! "${IMAGE_DIGEST}" =~ ^sha256:[a-f0-9]{64}$ ]]; then
  echo "ERROR: Failed to extract image digest" >&2
  exit 1
fi

echo "Image digest: ${IMAGE_DIGEST}"
echo "Image size: ${IMAGE_SIZE}"

# Write result to termination log
cat > "${RESULT_FILE}" <<EOF
{
  "schema": "${RESULT_SCHEMA}",
  "image_digest": "${IMAGE_DIGEST}",
  "image_size": ${IMAGE_SIZE}
}
EOF

echo "Build result written to ${RESULT_FILE}"
echo "=== Build Complete ==="

# Clean up BuildKit daemon
kill "${BUILDKIT_PID}" 2>/dev/null || true
wait "${BUILDKIT_PID}" 2>/dev/null || true

exit 0
