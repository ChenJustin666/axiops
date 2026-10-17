# Runs after checkout. History collection has bounded depth, time and output;
# its failure must never change HEAD or fail an otherwise valid build.
set -eu
tmp=$(mktemp -d) || exit 1
trap 'rm -f "$tmp/commits" "$tmp/files" "$tmp/fetch"; rmdir "$tmp"' EXIT
head=$(git rev-parse HEAD) || exit 1
base=${GIT_CHANGE_BASE:-}
format='%H%x00%an%x00%ae%x00%aI%x00%s'
git log -1 -z --format="$format" >"$tmp/commits" || exit 1
: >"$tmp/files" || exit 1
emit() {
  ct=0; ft=0
  # Drop oversized data as a whole, never emit a broken UTF-8/NUL record.
  if [ "$(wc -c <"$tmp/commits")" -gt 65536 ]; then
    git log -1 -z --format="$format" >"$tmp/commits" || exit 1
    ct=1
    if [ "$(wc -c <"$tmp/commits")" -gt 65536 ]; then : >"$tmp/commits"; fi
  fi
  if [ "$(wc -c <"$tmp/files")" -gt 65536 ]; then : >"$tmp/files"; ft=1; fi
  printf 'AXIOPS_GIT_CHANGESET_V1|%s|%s|%s|%s|%s|' "$1" "$head" "$base" "$ct" "$ft"
  base64 <"$tmp/commits" | tr -d '\r\n'
  printf '|'
  base64 <"$tmp/files" | tr -d '\r\n'
  printf '\n'
}
if [ -z "$base" ]; then emit first_build; exit 0; fi
# Publish HEAD immediately, even if history fetching subsequently times out.
emit unavailable
if [ "$base" != "$head" ] && ! git merge-base --is-ancestor "$base" "$head" 2>/dev/null; then
  timeout -s TERM 30 git -c http.lowSpeedLimit=1024 -c http.lowSpeedTime=10 fetch --no-tags --deepen=200 -- origin "$head" >"$tmp/fetch" 2>&1 || exit 0
fi
if ! git merge-base --is-ancestor "$base" "$head" 2>/dev/null; then
  echo '[changes] Baseline is outside the available history or the branch was rewritten; only the current commit is shown.'
  exit 0
fi
# max-count includes one sentinel so the receiver can flag truncation.
git log --max-count=101 -z --format="$format" "$base..$head" -- >"$tmp/commits" || exit 1
git diff --no-ext-diff --no-renames --name-status -z "$base" "$head" -- >"$tmp/files" || exit 1
emit complete
