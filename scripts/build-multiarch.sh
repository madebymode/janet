#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/build-multiarch.sh [options]

Options:
  --web-tag <tag>   Web image tag (default: troyxmccall/janet:web)
  --bot-tag <tag>   Bot image tag (default: troyxmccall/janet:bot)
  --platforms <p>   Platforms list (default: linux/amd64,linux/arm64)
  --context <name>  Docker context name (default: builder)
  --builder <name>  Buildx builder name (default: janet-builder)
  --no-login        Skip docker login prompt
  -h, --help        Show this help

Examples:
  scripts/build-multiarch.sh
  scripts/build-multiarch.sh --web-tag troyxmccall/janet:web --bot-tag troyxmccall/janet:bot
USAGE
}

WEB_TAG="troyxmccall/janet:web"
BOT_TAG="troyxmccall/janet:bot"
PLATFORMS="linux/amd64,linux/arm64"
BUILDER_CONTEXT="builder"
BUILDER_NAME="janet-builder"
DO_LOGIN=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --web-tag)
      WEB_TAG="$2"
      shift 2
      ;;
    --bot-tag)
      BOT_TAG="$2"
      shift 2
      ;;
    --platforms)
      PLATFORMS="$2"
      shift 2
      ;;
    --context)
      BUILDER_CONTEXT="$2"
      shift 2
      ;;
    --builder)
      BUILDER_NAME="$2"
      shift 2
      ;;
    --no-login)
      DO_LOGIN=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac

done

cleanup() {
  echo "Cleaning up..."
  docker context use default >/dev/null 2>&1 || true
  docker buildx rm "$BUILDER_NAME" >/dev/null 2>&1 || true
  docker context rm "$BUILDER_CONTEXT" >/dev/null 2>&1 || true
}

trap 'echo "Error on line $LINENO"' ERR
trap cleanup SIGINT SIGTERM

cleanup

docker context create "$BUILDER_CONTEXT" >/dev/null
export DOCKER_CLI_EXPERIMENTAL=enabled

docker buildx create \
  --name "$BUILDER_NAME" \
  --driver docker-container \
  --use \
  "$BUILDER_CONTEXT" >/dev/null

docker buildx inspect --bootstrap >/dev/null

if [[ $DO_LOGIN -eq 1 ]]; then
  docker login
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "$ROOT_DIR"

echo "Building and pushing web image: $WEB_TAG"
docker buildx build \
  --platform "$PLATFORMS" \
  -t "$WEB_TAG" \
  -f janet-server/Dockerfile \
  --attest type=sbom \
  --attest type=provenance \
  --push .

echo "Building and pushing bot image: $BOT_TAG"
docker buildx build \
  --platform "$PLATFORMS" \
  -t "$BOT_TAG" \
  -f cmd/janet-bot/Dockerfile \
  --attest type=sbom \
  --attest type=provenance \
  --push .

echo "Done."

cleanup
