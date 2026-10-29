#!/usr/bin/env bash
# Regenerates docs/assets/*.png, docs/assets/suggest.txt and docs/demo.html
# from a fake home folder, so no real paths or files ever appear.
# Needs Google Chrome. Writes ~3.5 GB of filler to $DEMO and removes it after.
set -euo pipefail
cd "$(dirname "$0")/.."

DEMO=${DEMO:-/tmp/tpdv-demo}
PORT=${PORT:-7799}
CHROME=${CHROME:-"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
OUT=docs/assets

python3 scripts/make_demo.py "$DEMO" >/dev/null
FAKE_HOME=$(cd "$DEMO/alex" && pwd -P) # resolve /tmp -> /private/tmp like tpdv does
make build >/dev/null
mkdir -p "$OUT"

export TPDV_DISK=1000000000000,412000000000 # fictional 1 TB disk
HOME=$FAKE_HOME ./bin/tpdv ui "$FAKE_HOME" -port "$PORT" -no-open >"$DEMO/server.log" 2>&1 &
SERVER=$!
PROFILE=$(mktemp -d)
cleanup() { kill "$SERVER" 2>/dev/null || true; rm -rf "$PROFILE" "$DEMO"; }
trap cleanup EXIT

URL=""
for _ in $(seq 50); do
  URL=$(grep -o 'http://127.0.0.1:[0-9]*/?t=[0-9a-f]*' "$DEMO/server.log" | head -1 || true)
  [ -n "$URL" ] && break
  sleep 0.2
done
[ -n "$URL" ] || { echo "server did not start" >&2; cat "$DEMO/server.log" >&2; exit 1; }

shot() { # name url [chrome flags...]
  local name=$1 url=$2; shift 2
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars --no-first-run \
    --user-data-dir="$PROFILE" --force-device-scale-factor=2 --window-size=1440,860 \
    --virtual-time-budget=5000 "$@" --screenshot="$OUT/$name" "$url" 2>/dev/null
  sips -Z 2000 "$OUT/$name" >/dev/null # keep files small enough for the README
  echo "  $OUT/$name"
}
shot treemap.png "$URL"
shot treemap-dark.png "$URL" --force-dark-mode
shot cleanup.png "$URL#clean"
shot cleanup-dark.png "$URL#clean" --force-dark-mode

HOME=$FAKE_HOME ./bin/tpdv report "$FAKE_HOME" -o docs/demo.html >/dev/null
echo "  docs/demo.html"
HOME=$FAKE_HOME NO_COLOR=1 ./bin/tpdv suggest "$FAKE_HOME" -n 3 2>/dev/null >"$OUT/suggest.txt"
echo "  $OUT/suggest.txt"
