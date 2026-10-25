#!/bin/bash
# Publishes a release ForeverCore updates from: the bundle (dataset, walk grid, routes, kill profiles, the Windows
# planner) with its manifest, and the dataset alone as before. Needs GH_TOKEN. Usage: tools/release.sh <notes file>
set -euo pipefail
notes="${1:-/dev/null}"
stamp="$(date -u +%Y%m%d-%H%M%S)"
out="$(mktemp -d)"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o "$out/foreverroutes.exe" ./cmd/foreverroutes
go run ./cmd/foreverroutes bundle -out "$out" -version "$stamp" -exe "$out/foreverroutes.exe"
cp data/routes-data.json.gz "$out/routes-data.json.gz"
(cd "$out" && sha256sum routes-data.json.gz > routes-data.json.gz.sha256)
gh release create "data-$stamp" "$out/foreverroutes-bundle.zip" "$out/foreverroutes-bundle.json" \
  "$out/routes-data.json.gz" "$out/routes-data.json.gz.sha256" --title "Data $stamp" --notes-file "$notes" --latest
