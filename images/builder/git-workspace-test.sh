#!/bin/sh
# Uses real local Git checkout, without a remote service or real credentials.
# Run on Linux to verify the permissions shared with the non-root builder.
set -eu
source_dir=$(cd "$(dirname "$0")/../../internal/cicd/build" && pwd)
temp_root=$(cd "${TMPDIR:-/tmp}" && pwd -P)
fixture=$(mktemp -d "$temp_root/axiops-git-workspace.XXXXXXXX")
cleanup_fixture() {
  local resolved
  resolved=$(cd "$fixture" && pwd -P) || return
  case "$resolved" in
    "$temp_root"/axiops-git-workspace.*) rm -rf -- "$resolved" ;;
    *) echo "Refusing unexpected fixture cleanup: $resolved" >&2 ;;
  esac
}
trap cleanup_fixture EXIT
umask 022
mkdir -p "$fixture/origin" "$fixture/bin"
real_git=$(command -v git)
git -C "$fixture/origin" init -q
git -C "$fixture/origin" config user.email fixture@example.invalid
git -C "$fixture/origin" config user.name fixture
printf 'FROM scratch\nCOPY payload /payload\n' > "$fixture/origin/Dockerfile"
printf 'fixture\n' > "$fixture/origin/payload"
mkdir "$fixture/origin/scripts"
printf '#!/bin/sh\nexit 0\n' > "$fixture/origin/scripts/run.sh"
chmod 755 "$fixture/origin/scripts/run.sh"
git -C "$fixture/origin" add .
git -C "$fixture/origin" commit -qm fixture
branch=$(git -C "$fixture/origin" symbolic-ref --short HEAD)

# Inspect the helper while fetch is running; never print its contents/token.
cat > "$fixture/bin/git" <<'EOF'
#!/bin/sh
set -eu
if [ -n "${GIT_TOKEN:-}" ] && [ -n "${GIT_ASKPASS:-}" ]; then
  test "$(stat -c '%a' "$GIT_ASKPASS")" = 700
  if grep -Fq "$GIT_TOKEN" "$GIT_ASKPASS"; then
    echo 'credential was embedded in the helper' >&2
    exit 1
  fi
  : > "$FIXTURE/helper-checked"
fi
exec "$REAL_GIT" "$@"
EOF
chmod 755 "$fixture/bin/git"

for scenario in authenticated anonymous; do
  workspace="$fixture/$scenario"
  mkdir "$workspace"
  sed -e "s@/tmp/axiops-git-@$fixture/git-@g" \
    -e "s@/dev/termination-log@$fixture/result@g" \
    -e "s@/workspace@$workspace@g" \
    "$source_dir/git_clone.sh" > "$fixture/clone.sh"
  token=''
  test "$scenario" = anonymous || token='fixture-placeholder-token'
  (
    # The checkout must not inherit a restrictive launcher umask either.
    umask 077
    export PATH="$fixture/bin:$PATH" FIXTURE="$fixture" REAL_GIT="$real_git"
    export GIT_TOKEN="$token" GIT_URL="$fixture/origin" GIT_BRANCH="$branch" GIT_COMMIT=''
    sh "$fixture/clone.sh" > "$fixture/output" 2>&1
  ) || { cat "$fixture/output"; exit 1; }
  test "$(stat -c '%a' "$workspace/Dockerfile")" = 644
  test "$(stat -c '%a' "$workspace/payload")" = 644
  test "$(stat -c '%a' "$workspace/scripts")" = 755
  test "$(stat -c '%a' "$workspace/scripts/run.sh")" = 755
  cmp "$workspace/Dockerfile" "$fixture/origin/Dockerfile"
  test ! -e "$fixture/git-askpass"
  test ! -e "$fixture/git-fetch.log"
done
test -f "$fixture/helper-checked"
echo 'PASS: real Git checkout remains readable, executable bits survive, credential helper stays private and is removed'
