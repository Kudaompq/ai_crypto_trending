#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
backend_port="${PORT:-8080}"
backend_pid=""
frontend_pid=""
build_dir=""

cleanup() {
    trap - EXIT INT TERM
    for pid in "$frontend_pid" "$backend_pid"; do
        if [[ -n "$pid" ]]; then
            kill "$pid" 2>/dev/null || true
            wait "$pid" 2>/dev/null || true
        fi
    done
    if [[ -n "$build_dir" ]]; then
        rm -f "$build_dir/server"
        rmdir "$build_dir" 2>/dev/null || true
    fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [[ -f "$root_dir/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "$root_dir/.env"
    set +a
fi

if [[ -z "${WATCHLIST_DATABASE_URL:-}" ]]; then
    echo "缺少 WATCHLIST_DATABASE_URL。请复制 .env.example 为 .env 并配置 PostgreSQL。" >&2
    exit 1
fi

for command in go npm; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "缺少 $command，请先安装后重试。" >&2
        exit 1
    fi
done

if [[ ! -d "$root_dir/frontend/node_modules" ]]; then
    echo "安装前端依赖..."
    (cd "$root_dir/frontend" && npm ci)
fi

echo "编译后端..."
build_dir="$(mktemp -d "${TMPDIR:-/tmp}/crypto-trending.XXXXXX")"
(cd "$root_dir" && go build -o "$build_dir/server" ./backend/cmd/server)

echo "启动后端：http://localhost:$backend_port"
(cd "$root_dir/backend" && PORT="$backend_port" exec "$build_dir/server") &
backend_pid=$!

echo "启动前端：http://localhost:5173"
(cd "$root_dir/frontend" && VITE_API_PROXY_TARGET="http://localhost:$backend_port" exec ./node_modules/.bin/vite --strictPort) &
frontend_pid=$!

echo "按 Ctrl+C 停止两个服务。"
while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$frontend_pid" 2>/dev/null; do
    sleep 1 &
    wait $! || true
done

echo "服务已退出，正在停止另一服务。" >&2
exit 1
