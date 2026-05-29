#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./run.sh                # run locally with in-memory storage (default)
#   ./run.sh docker         # run via Docker (in-memory)
#   ./run.sh postgres       # run via Docker with PostgreSQL
#   ./run.sh test           # run all tests with race detector
#   ./run.sh test:coverage  # run all tests with coverage report

MODE=${1:-local}

# Load .env if it exists
if [ -f .env ]; then
  set -a; source .env; set +a
fi

case "$MODE" in
  local)
    swag init -g cmd/api/main.go -o docs --outputTypes go --quiet
    go run ./cmd/api
    ;;

  docker)
    docker compose --profile memory up --build
    ;;

  postgres)
    docker compose --profile postgres up --build
    ;;

  test)
    go test ./... -race -count=1
    ;;

  test:coverage)
    go test ./... -race -count=1 -coverprofile=coverage.out
    go tool cover -func=coverage.out
    rm -f coverage.out
    ;;

  *)
    echo "Usage: $0 [local|docker|postgres|test|test:coverage]"
    exit 1
    ;;
esac
