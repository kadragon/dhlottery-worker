#!/usr/bin/env bash
# Truncates oversized Bash tool output before it enters the agent context.
# Keeps first 50 + last 100 lines; omits the middle with a count notice.

set -euo pipefail

MAX_LINES="${BASH_OUTPUT_MAX_LINES:-200}"
HEAD_KEEP=50
TAIL_KEEP=100

payload=$(cat)
output=$(jq -r '.tool_response // empty' <<<"$payload")
[[ -z "$output" ]] && { printf '%s' "$payload"; exit 0; }

line_count=$(printf '%s' "$output" | wc -l | tr -d ' ')
[[ "$line_count" -le "$MAX_LINES" ]] && { printf '%s' "$payload"; exit 0; }

head_part=$(printf '%s' "$output" | head -n "$HEAD_KEEP")
tail_part=$(printf '%s' "$output" | tail -n "$TAIL_KEEP")
omitted=$((line_count - HEAD_KEEP - TAIL_KEEP))

truncated="${head_part}
… [truncated ${omitted} lines — re-run with a narrower command if the middle matters] …
${tail_part}"

jq --arg t "$truncated" '.tool_response = $t' <<<"$payload"
