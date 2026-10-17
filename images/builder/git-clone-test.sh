#!/bin/bash
set -euo pipefail
source_dir=$(cd "$(dirname "$0")/../../internal/cicd/build" && pwd)
temp_root=$(cd "${TMPDIR:-/tmp}" && pwd -P)
fixture=$(mktemp -d "$temp_root/axiops-git-retry.XXXXXXXX")
cleanup_fixture() {
  local resolved
  resolved=$(cd "$fixture" && pwd -P) || return
  case "$resolved" in
    "$temp_root"/axiops-git-retry.*) rm -rf -- "$resolved" ;;
    *) echo "Refusing unexpected fixture cleanup: $resolved" >&2 ;;
  esac
}
trap cleanup_fixture EXIT
mkdir -p "$fixture/bin" "$fixture/workspace"
sed -e "s@/tmp/axiops-git-@$fixture/git-@g" -e "s@/dev/termination-log@$fixture/result@g" -e "s@/workspace@$fixture/workspace@g" "$source_dir/git_clone.sh" > "$fixture/clone.sh"
cat > "$fixture/bin/timeout" <<'EOF'
#!/bin/bash
[[ "$1 $2 $3" == '-s TERM 120' ]] || exit 99
shift 3
exec "$@"
EOF
cat > "$fixture/bin/git" <<'EOF'
#!/bin/bash
if [[ "$*" == *'fetch --depth=1'* ]]; then
  count=$(cat "$FIXTURE/count")
  count=$((count+1))
  echo "$count" > "$FIXTURE/count"
  case "$CASE" in
    transient) [[ "$count" -ge 3 ]] && exit 0; echo 'Failed to connect: Connection reset' >&2; exit 128 ;;
    auth) echo 'fatal: Authentication failed' >&2; exit 128 ;;
    timeout) exit 124 ;;
  esac
elif [[ "$1" == checkout ]]; then
  echo checked-out > "$FIXTURE/checkout"
fi
EOF
cat > "$fixture/bin/sleep" <<'EOF'
#!/bin/bash
exit 0
EOF
chmod +x "$fixture/bin/"*
for scenario in transient auth timeout; do
  echo 0 > "$fixture/count"
  rm -f "$fixture/checkout" "$fixture/result"
  result=0
  PATH="$fixture/bin:$PATH" FIXTURE="$fixture" CASE="$scenario" GIT_URL=https://git.example/app.git GIT_BRANCH=main GIT_COMMIT='' \
    bash "$fixture/clone.sh" > "$fixture/output" 2>&1 || result=$?
  case "$scenario" in
    transient) [[ "$result" == 0 && $(cat "$fixture/count") == 3 && -f "$fixture/checkout" ]] ;;
    auth) [[ "$result" == 128 && $(cat "$fixture/count") == 1 && ! -f "$fixture/checkout" ]]; grep -q GIT_FETCH_FAILED "$fixture/result" ;;
    timeout) [[ "$result" == 124 && $(cat "$fixture/count") == 3 && ! -f "$fixture/checkout" ]]; grep -q GIT_FETCH_RETRIES_EXHAUSTED "$fixture/result" ;;
  esac
done
echo 'PASS: Git transport retries, permanent auth failure, bounded timeout, no checkout on failure'
