#!/bin/bash
# Runs without Docker or a cluster. Replaces kernel paths in an isolated copy
# and stubs BuildKit, exercising startup failures without changing host sysctls.
set -euo pipefail
source_dir=$(cd "$(dirname "$0")" && pwd)
temp_root=$(cd "${TMPDIR:-/tmp}" && pwd -P)
fixture=$(mktemp -d "$temp_root/axiops-builder-startup.XXXXXXXX")
cleanup_fixture() {
  local resolved
  resolved=$(cd "$fixture" && pwd -P) || return
  case "$resolved" in
    "$temp_root"/axiops-builder-startup.*) rm -rf -- "$resolved" ;;
    *) echo "Refusing to remove unexpected fixture: $resolved" >&2 ;;
  esac
}
trap cleanup_fixture EXIT
mkdir -p "$fixture/bin" "$fixture/home" "$fixture/workspace"
printf 'FROM scratch\n' > "$fixture/workspace/Dockerfile"
sed -e "s@/proc/sys/user/max_user_namespaces@$fixture/max_user_namespaces@g" \
  -e "s@/proc/sys/kernel/unprivileged_userns_clone@$fixture/userns_clone@g" \
  -e "s@/tmp/buildkitd.log@$fixture/buildkitd.log@g" \
  "$source_dir/entrypoint.sh" > "$fixture/entrypoint.sh"
printf '1\n' > "$fixture/userns_clone"
cat > "$fixture/bin/rootlesskit" <<'EOF'
#!/bin/bash
echo invoked >> "$HOME/invoked"
echo 'test: namespace creation failed' >&2
exit 42
EOF
cat > "$fixture/bin/buildctl" <<'EOF'
#!/bin/bash
exit 1
EOF
cat > "$fixture/bin/sleep" <<'EOF'
#!/bin/bash
echo sleep >> "$HOME/sleeps"
/bin/sleep 0.05
EOF
chmod +x "$fixture/bin/"*
run_failure() {
  local code=0
  PATH="$fixture/bin:$PATH" HOME="$fixture/home" BUILD_CONTEXT="$fixture/workspace" \
    IMAGE_URL=registry.example/team/test:ci AXIOPS_RESULT_FILE="$fixture/result" \
    bash "$fixture/entrypoint.sh" > "$fixture/output" 2>&1 || code=$?
  [[ "$code" == 1 ]] || { cat "$fixture/output"; echo "unexpected exit $code"; exit 1; }
}
printf '0\n' > "$fixture/max_user_namespaces"
run_failure
grep -q 'ROOTLESS_USER_NAMESPACE_DISABLED' "$fixture/result"
[[ ! -f "$fixture/home/invoked" ]] || { echo 'daemon started with zero namespace quota'; exit 1; }
printf '28633\n' > "$fixture/max_user_namespaces"
run_failure
grep -q 'BUILDKIT_START_FAILED.*exit=42' "$fixture/result"
grep -q 'test: namespace creation failed' "$fixture/output"
[[ $(wc -l < "$fixture/home/sleeps") -lt 30 ]] || { echo 'dead daemon waited for timeout'; exit 1; }
echo 'PASS: disabled namespaces fail before launch; dead daemon fails promptly and preserves its cause'
