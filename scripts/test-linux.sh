#!/usr/bin/env bash
# Run `go test ./...` for wonton inside a Linux container, reproducing the
# Linux CI job locally from macOS or any Docker-capable host. Any arguments
# passed to this script are appended to `go test` inside the container.
#
#   ./scripts/test-linux.sh
#   ./scripts/test-linux.sh -race ./pty/...
#   ./scripts/test-linux.sh -run TestSession_Resize ./termsession/
#
# `-t` gives the container a TTY so wonton's pty and termsession tests can
# allocate real PTYs. Without it, `test -t 0` inside the PTY child fails and
# the suite flakes.
set -euo pipefail

cd "$(dirname "$0")/.."

IMAGE=wonton-test

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
	docker build -f scripts/Dockerfile.linux -t "$IMAGE" .
fi

if [ "$#" -eq 0 ]; then
	set -- ./...
fi

exec docker run \
	--rm \
	-t \
	-v "$PWD:/src" \
	-w /src \
	"$IMAGE" \
	go test "$@"
