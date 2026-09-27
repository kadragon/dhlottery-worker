#!/usr/bin/env bash
# PostToolUse: gofmt the Go file just edited/written. Non-Go paths are ignored.
set -euo pipefail
command -v jq >/dev/null 2>&1 || exit 0
command -v gofmt >/dev/null 2>&1 || exit 0
file=$(jq -r '.tool_input.file_path // empty')
[[ "$file" == *.go && -f "$file" ]] || exit 0
gofmt -w "$file"
