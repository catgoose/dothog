#!/usr/bin/env bash
# Generate Go package documentation as Markdown using gomarkdoc.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCS_DIR="$ROOT_DIR/docs/packages"

if ! command -v gomarkdoc &>/dev/null; then
  echo "gomarkdoc not found. Install via:"
  echo "  go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest"
  exit 1
fi

mkdir -p "$DOCS_DIR"

# Map of import-path suffix → output filename
declare -A PACKAGES=(
  ["internal/config"]="config"
  ["internal/logger"]="logger"
  ["internal/shared"]="shared"
  ["internal/routes"]="routes"
  ["internal/routes/handler"]="routes-handler"
  ["internal/routes/htmx"]="routes-htmx"
  ["internal/routes/hypermedia"]="routes-hypermedia"
  ["internal/routes/middleware"]="routes-middleware"
  ["internal/routes/params"]="routes-params"
  ["internal/routes/response"]="routes-response"
  ["internal/database"]="database"
  ["internal/database/repository"]="database-repository"
  ["internal/demo"]="demo"
  ["internal/ssebroker"]="ssebroker"
)

for pkg in "${!PACKAGES[@]}"; do
  out="${PACKAGES[$pkg]}"
  echo "Generating docs for $pkg → $out.md"
  gomarkdoc --output "$DOCS_DIR/$out.md" "./$pkg" 2>/dev/null || {
    echo "  (skipped — gomarkdoc failed for $pkg)"
  }
done

echo ""
echo "Done! Package docs written to docs/packages/"
