package build

// DefaultPlatformBuildCommand is the command used by the standard AxiOps
// builder image. The image entrypoint is deliberately overridden by the Job
// provider, so this command must invoke the builder entrypoint explicitly.
// The entrypoint builds the repository Dockerfile with BuildKit, pushes the
// image, and writes the immutable digest result to AXIOPS_RESULT_FILE.
const DefaultPlatformBuildCommand = "exec /usr/local/bin/entrypoint.sh"
