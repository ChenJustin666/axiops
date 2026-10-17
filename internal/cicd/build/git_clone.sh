#!/bin/sh
set -eu
export LC_ALL=C GIT_TERMINAL_PROMPT=0
# Checkout is shared with a non-root builder. Keep source files readable and
# directories traversable; restrict only the credential helper below.
umask 022
cleanup() { rm -f /tmp/axiops-git-askpass /tmp/axiops-git-fetch.log; }
trap cleanup EXIT
if [ -n "${GIT_TOKEN:-}" ]; then
  (
    umask 077
    cat >/tmp/axiops-git-askpass <<'EOF'
#!/bin/sh
case "$1" in
  *Username*) printf '%s' 'oauth2' ;;
  *) printf '%s' "$GIT_TOKEN" ;;
esac
EOF
    chmod 700 /tmp/axiops-git-askpass
  )
  export GIT_ASKPASS=/tmp/axiops-git-askpass
fi
git init /workspace
cd /workspace
git remote add origin "$GIT_URL"
ref="${GIT_COMMIT:-${GIT_BRANCH:-HEAD}}"
# Fetch is safe to retry into the same repository; arbitrary user build scripts
# are not. A process timeout also bounds DNS/TCP stalls before HTTP low-speed
# detection becomes effective. The Job/host execution has its own total deadline.
attempt=1
while :; do
  result=0
  timeout -s TERM 120 git -c http.lowSpeedLimit=1024 -c http.lowSpeedTime=30 fetch --depth=1 -- origin "$ref" >/tmp/axiops-git-fetch.log 2>&1 || result=$?
  cat /tmp/axiops-git-fetch.log
  [ "$result" -eq 0 ] && break
  if [ "$attempt" -ge 3 ]; then
    echo "AXIOPS_BUILD_ERROR: GIT_FETCH_RETRIES_EXHAUSTED: code fetch failed after 3 attempts" >/dev/termination-log
    exit "$result"
  fi
  if [ "$result" -ne 124 ] && [ "$result" -ne 143 ] && ! grep -Eiq 'timed out|timeout|Could not resolve|Failed to connect|Connection reset|Connection refused|remote end hung up|early EOF|TLS connection was non-properly terminated|HTTP[^0-9]*(429|502|503|504)|requested URL returned error: (429|502|503|504)' /tmp/axiops-git-fetch.log; then
    echo "AXIOPS_BUILD_ERROR: GIT_FETCH_FAILED: check repository URL, branch and credentials; see git-clone output" >/dev/termination-log
    exit "$result"
  fi
  echo "[retry] Git transport failure; retry $attempt/2 in $((attempt * 3)) seconds"
  sleep "$((attempt * 3))"
  attempt=$((attempt + 1))
done
git checkout --detach FETCH_HEAD
if [ -n "${GIT_COMMIT:-}" ]; then test "$(git rev-parse HEAD)" = "$GIT_COMMIT"; fi
git log -1 --format='commit=%H%nsubject=%s'
git rev-parse HEAD >/dev/termination-log
