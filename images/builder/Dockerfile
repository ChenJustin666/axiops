# axiops/builder:stable — AxiOps CI Builder Image
# Uses BuildKit (rootless) to build and push container images inside K8s Jobs.
# The builder reads environment variables injected by kubejob_provider:
#   DOCKERFILE_PATH  — relative path to Dockerfile (default: Dockerfile)
#   BUILD_CONTEXT    — build context directory (default: .)
#   IMAGE_URL        — full image reference to push (e.g. registry/repo:tag)
#   DOCKER_CONFIG    — directory of the mounted registry config.json Secret
#   AXIOPS_RESULT_FILE — where to write the build result JSON (/dev/termination-log)
#   AXIOPS_BUILD_MODE — "dockerfile" (other modes require a different builder)
#   AXIOPS_RESULT_SCHEMA — "axiops.build/v1"
#
# The entrypoint builds the image with BuildKit, pushes it, extracts the digest
# and size, then writes a JSON result to $AXIOPS_RESULT_FILE.

FROM moby/buildkit:v0.21.1-rootless

USER root

# Install runtime dependencies
RUN apk add --no-cache \
      bash \
      jq \
      skopeo \
    && rm -rf /var/cache/apk/*

# Keep the official OCI runtime, subordinate UID/GID mappings and rootless
# dependencies together; copying only buildkitd/rootlesskit loses these.

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh && chmod +x /usr/local/bin/entrypoint.sh

USER 1000:1000
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
