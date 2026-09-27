#!/usr/bin/env bash
# PreToolUse: block agent writes to .env files (secrets). Exit 2 = deny + stderr to model.
set -euo pipefail
command -v jq >/dev/null 2>&1 || exit 0
file=$(jq -r '.tool_input.file_path // empty')
base=$(basename -- "$file")
if [[ "$base" == .env || "$base" == .env.* || "$base" == *.env ]]; then
  echo "BLOCKED: do not edit $base — secrets live in GitHub Actions secrets / local env" >&2
  exit 2
fi
exit 0
