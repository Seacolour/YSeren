#!/usr/bin/env bash

set -euo pipefail

image="${1:?usage: docker/smoke.sh IMAGE [EXPECTED_VERSION]}"
expected_version="${2:-dev}"
fixture_dir="$(mktemp -d)"
container="yseren-smoke-${GITHUB_RUN_ID:-local}-$$"

cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  fixture_parent="$(dirname "$fixture_dir")"
  resolved_parent="$(cd "$fixture_parent" && pwd -P)"
  resolved_fixture="$(cd "$fixture_dir" && pwd -P)"
  if [ "$resolved_fixture" != "/" ] && [ "$(dirname "$resolved_fixture")" = "$resolved_parent" ]; then
    rm -rf -- "$resolved_fixture"
  else
    echo "Refusing to remove unexpected smoke directory: $resolved_fixture" >&2
  fi
}
trap cleanup EXIT

mkdir -p "$fixture_dir/media"
printf '0123456789abcdef' > "$fixture_dir/media/sample.mp4"
printf 'not media' > "$fixture_dir/media/ignored.zip"

docker run --detach \
  --name "$container" \
  --publish 127.0.0.1::1479 \
  --volume "$fixture_dir/media:/media:ro" \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges:true \
  "$image" >/dev/null

published_address="$(docker port "$container" 1479/tcp | head -n 1)"
host_port="${published_address##*:}"
base_url="http://127.0.0.1:${host_port}"

ready=false
for _ in $(seq 1 30); do
  if curl -fsS "$base_url/api/status" >/dev/null; then
    ready=true
    break
  fi
  sleep 1
done
if [ "$ready" != "true" ]; then
  docker logs "$container" >&2
  echo "YSeren container did not become ready." >&2
  exit 1
fi

test "$(docker exec "$container" id -u)" = "10001"
test "$(docker inspect --format '{{.HostConfig.ReadonlyRootfs}}' "$container")" = "true"
test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/media"}}{{.RW}}{{end}}{{end}}' "$container")" = "false"
docker inspect --format '{{json .HostConfig.CapDrop}}' "$container" | grep -Fq '"ALL"'
docker inspect --format '{{json .HostConfig.SecurityOpt}}' "$container" | grep -Fq '"no-new-privileges:true"'

version_response="$(curl -fsS "$base_url/api/version")"
printf '%s' "$version_response" | grep -Fq "\"version\":\"${expected_version}\""

tree_response="$(curl -fsS "$base_url/api/tree?refresh=1")"
printf '%s' "$tree_response" | grep -Fq 'sample.mp4'
if printf '%s' "$tree_response" | grep -Fq 'ignored.zip'; then
  echo "Non-media file appeared in /api/tree." >&2
  exit 1
fi

range_body="$fixture_dir/range-body"
range_headers="$fixture_dir/range-headers"
range_status="$(curl -sS \
  --dump-header "$range_headers" \
  --output "$range_body" \
  --write-out '%{http_code}' \
  --header 'Range: bytes=2-5' \
  "$base_url/stream/media/sample.mp4")"
test "$range_status" = "206"
test "$(cat "$range_body")" = "2345"
tr -d '\r' < "$range_headers" | grep -Fqx 'Content-Range: bytes 2-5/16'

test "$(curl -sS -o /dev/null -w '%{http_code}' --head "$base_url/stream/media/sample.mp4")" = "200"
test "$(curl -sS -o /dev/null -w '%{http_code}' "$base_url/stream/media/ignored.zip")" = "404"

docker stop --time 10 "$container" >/dev/null
test "$(docker inspect --format '{{.State.ExitCode}}' "$container")" = "0"

echo "Docker smoke test passed for $image ($expected_version)."
