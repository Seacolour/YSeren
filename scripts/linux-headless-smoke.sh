#!/usr/bin/env bash
set -euo pipefail

binary=${1:-./yseren}
if [[ ! -x "$binary" ]]; then
  echo "Linux Headless binary is not executable: $binary" >&2
  exit 1
fi
binary=$(realpath "$binary")

work_dir=$(mktemp -d)
server_pid=""
base_url=""
server_log="$work_dir/server.log"
config_path="$work_dir/yseren.yaml"

cleanup() {
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
    kill -TERM "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  chmod -R u+rwX "$work_dir" 2>/dev/null || true
  rm -rf -- "$work_dir"
}
trap cleanup EXIT

fail() {
  echo "Linux Headless smoke test failed: $*" >&2
  if [[ -f "$server_log" ]]; then
    echo "--- server log ---" >&2
    cat "$server_log" >&2
  fi
  exit 1
}

expect_status() {
  local expected=$1
  shift
  local actual
  actual=$(curl -sS -o /dev/null -w '%{http_code}' "$@")
  [[ "$actual" == "$expected" ]] || fail "HTTP status $actual, expected $expected: $*"
}

start_server() {
  : >"$server_log"
  "$binary" -config "$config_path" >"$server_log" 2>&1 &
  server_pid=$!
  base_url=""

  for _ in $(seq 1 100); do
    if ! kill -0 "$server_pid" 2>/dev/null; then
      wait "$server_pid" || true
      fail "server exited during startup"
    fi
    base_url=$(grep -Eo 'http://localhost:[0-9]+/' "$server_log" | head -n 1 || true)
    if [[ -n "$base_url" ]] && curl -fsS "${base_url}api/status" >/dev/null; then
      return
    fi
    sleep 0.1
  done
  fail "server did not become ready"
}

media_dir="$work_dir/媒体 source"
outside_dir="$work_dir/outside"
mkdir -p "$media_dir/Season 1" "$media_dir/Music" "$media_dir/blocked" "$outside_dir"
printf '0123456789' >"$media_dir/Season 1/示例 #1.MP4"
printf 'abcdefghij' >"$media_dir/Music/theme.Mp3"
printf 'archive' >"$media_dir/archive.zip"
printf 'unreadable' >"$media_dir/unreadable.mp4"
printf 'blocked' >"$media_dir/blocked/secret.mp4"
printf 'outside' >"$outside_dir/outside.mp4"
dd if=/dev/zero of="$media_dir/large.mp4" bs=1M count=4 status=none
ln -s 'Season 1/示例 #1.MP4' "$media_dir/inside-link.mp4"
ln -s '../outside/outside.mp4' "$media_dir/escape-link.mp4"
chmod 000 "$media_dir/unreadable.mp4" "$media_dir/blocked"

cat >"$config_path" <<EOF
server:
  port: 0
  log_level: info

sources:
  - name: linux
    path: "$media_dir"
EOF

file "$binary" | grep -q 'ELF 64-bit' || fail "binary is not a 64-bit ELF"
if ldd "$binary" >"$work_dir/ldd.log" 2>&1; then
  fail "binary unexpectedly has dynamic dependencies"
fi
grep -Eq 'not a dynamic executable|statically linked' "$work_dir/ldd.log" || fail "static-link check was inconclusive"

start_server
port=${base_url%/}
port=${port##*:}

status_json=$(curl -fsS "${base_url}api/status")
if grep -Fq "$media_dir" <<<"$status_json"; then
  fail "/api/status exposed the source path"
fi
EXPECTED_PORT="$port" python3 -c '
import json, os, sys
value = json.load(sys.stdin)
assert value["state"] == "running"
assert value["source"] == "linux"
assert value["port"] == int(os.environ["EXPECTED_PORT"])
assert value["urls"]
' <<<"$status_json"

tree_json=$(curl -fsS "${base_url}api/tree?refresh=1")
python3 -c '
import json, sys
root = json.load(sys.stdin)["root"]
nodes = {}
def visit(node):
    nodes[node.get("name", "")] = node
    for child in node.get("children", []):
        visit(child)
visit(root)
for name in ("示例 #1.MP4", "theme.Mp3", "inside-link.mp4", "large.mp4"):
    assert name in nodes, name
for name in ("archive.zip", "unreadable.mp4", "escape-link.mp4", "secret.mp4"):
    assert name not in nodes, name
assert nodes["inside-link.mp4"]["size"] == 10
' <<<"$tree_json"

curl -fsS "${base_url}stream/linux/inside-link.mp4" -o "$work_dir/full.bin"
cmp "$media_dir/Season 1/示例 #1.MP4" "$work_dir/full.bin"

curl -fsSI "${base_url}stream/linux/inside-link.mp4" >"$work_dir/head.headers"
grep -qi '^Accept-Ranges: bytes' "$work_dir/head.headers" || fail "HEAD omitted Accept-Ranges"
grep -qi '^Content-Length: 10' "$work_dir/head.headers" || fail "HEAD returned the wrong length"

curl -fsS -H 'Range: bytes=2-5' -D "$work_dir/range.headers" \
  "${base_url}stream/linux/inside-link.mp4" -o "$work_dir/range.bin"
printf '2345' >"$work_dir/expected-range.bin"
cmp "$work_dir/expected-range.bin" "$work_dir/range.bin"
grep -qi '^Content-Range: bytes 2-5/10' "$work_dir/range.headers" || fail "bounded Range returned the wrong Content-Range"

expect_status 206 -H 'Range: bytes=4-' "${base_url}stream/linux/inside-link.mp4"
expect_status 206 -H 'Range: bytes=-4' "${base_url}stream/linux/inside-link.mp4"
expect_status 416 -H 'Range: bytes=99-' "${base_url}stream/linux/inside-link.mp4"
expect_status 416 -H 'Range: bytes=0-1,4-5' "${base_url}stream/linux/inside-link.mp4"
expect_status 416 -H 'Range: bytes=abc-def' "${base_url}stream/linux/inside-link.mp4"
expect_status 404 "${base_url}stream/linux/archive.zip"
expect_status 404 "${base_url}stream/linux/unreadable.mp4"
expect_status 404 "${base_url}stream/linux/escape-link.mp4"

playlist=$(curl -fsS "${base_url}playlist.m3u")
grep -Fq '/stream/linux/inside-link.mp4' <<<"$playlist" || fail "playlist omitted a safe media file"
if grep -Eq 'archive\.zip|unreadable\.mp4|escape-link\.mp4' <<<"$playlist"; then
  fail "playlist exposed an inaccessible file"
fi
curl -fsS "${base_url}api/version" | grep -Fq '"version"' || fail "/api/version response is invalid"
curl -fsS "$base_url" | grep -Fq '<title>YSeren</title>' || fail "embedded Web Player was not served"

sed "s/port: 0/port: $port/" "$config_path" >"$work_dir/conflict.yaml"
if "$binary" -config "$work_dir/conflict.yaml" >"$work_dir/conflict.log" 2>&1; then
  fail "a second process unexpectedly acquired the active port"
fi
grep -Eq 'address already in use|bind:' "$work_dir/conflict.log" || fail "port conflict did not report the bind failure"

curl -sS --limit-rate 1k "${base_url}stream/linux/large.mp4" -o /dev/null &
client_pid=$!
sleep 0.2
kill -TERM "$client_pid" 2>/dev/null || true
wait "$client_pid" 2>/dev/null || true
expect_status 200 "${base_url}api/status"

kill -TERM "$server_pid"
wait "$server_pid" || fail "SIGTERM shutdown returned a failure"
server_pid=""

start_server
kill -INT "$server_pid"
wait "$server_pid" || fail "SIGINT shutdown returned a failure"
server_pid=""

echo "Linux Headless smoke test passed"
