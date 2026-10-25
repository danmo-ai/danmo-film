# Unified build output paths — source from scripts/*.sh
# Layout: out/frontend/dist, out/server/, out/run/

if [[ -n "${_DQ_OUT_PATHS_LOADED:-}" ]]; then
  return 0 2>/dev/null || true
fi
_DQ_OUT_PATHS_LOADED=1

_DQ_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export DQ_ROOT="$(cd "$_DQ_SCRIPT_DIR/.." && pwd)"
export DQ_OUT="$DQ_ROOT/out"
export DQ_FRONTEND_DIST="$DQ_OUT/frontend/dist"
export DQ_SERVER_DIR="$DQ_OUT/server"
export DQ_RUN_DIR="$DQ_OUT/run"
export DQ_PROJECT="${DQ_PROJECT:-danmo-film}"
# Port convention: backend 78xx / frontend 58xx (same suffix per project)
export DQ_BACKEND_PORT="${DQ_BACKEND_PORT:-7803}"
export DQ_FRONTEND_PORT="${DQ_FRONTEND_PORT:-5803}"

dq_ensure_out_layout() {
  mkdir -p \
    "$DQ_FRONTEND_DIST" \
    "$DQ_SERVER_DIR" \
    "$DQ_RUN_DIR"
}

# Free a TCP listen port so `make dev` is not blocked by a leftover go run / vite.
dq_free_listen_port() {
  local port="$1"
  local pids
  pids="$(lsof -nP -t -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  [[ -n "$pids" ]] || return 0
  echo "==> Port $port already in use (pids: $(echo "$pids" | tr '\n' ' ')); stopping so ${DQ_PROJECT} can bind"
  echo "$pids" | xargs kill -TERM 2>/dev/null || true
  sleep 0.4
  pids="$(lsof -nP -t -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    echo "$pids" | xargs kill -KILL 2>/dev/null || true
  fi
}

dq_wait_http() {
  local url="$1"
  local log="$2"
  local i
  for i in $(seq 1 30); do
    if curl -sf -m 1 "$url" >/dev/null 2>&1; then
      return 0
    fi
    if [[ -n "$log" && -f "$log" ]] && grep -q "address already in use" "$log"; then
      echo "failed to bind (see $log):" >&2
      cat "$log" >&2
      return 1
    fi
    sleep 0.2
  done
  echo "timed out waiting for $url" >&2
  [[ -n "$log" && -f "$log" ]] && tail -n 20 "$log" >&2
  return 1
}
