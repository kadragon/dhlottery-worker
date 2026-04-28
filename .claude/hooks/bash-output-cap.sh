#!/usr/bin/env bash
# Truncates oversized Bash tool output before it enters the agent context.
# Keeps first 50 + last 100 lines; omits the middle with a count notice.

set -euo pipefail

MAX_LINES="${BASH_OUTPUT_MAX_LINES:-200}"
HEAD_KEEP=50
TAIL_KEEP=100

# Avoid hanging when invoked directly from a terminal (no piped input)
[[ -t 0 ]] && exit 0

# Pass through if jq is unavailable rather than breaking every Bash call
command -v jq >/dev/null 2>&1 || { cat; exit 0; }

payload=$(cat)

# tool_response may be a plain string or {"output":"...",...} depending on tool type
output=$(jq -r 'if (.tool_response | type) == "object" then .tool_response.output else .tool_response end // empty' <<<"$payload" 2>/dev/null) || { printf -- '%s' "$payload"; exit 0; }
[[ -z "$output" ]] && { printf -- '%s' "$payload"; exit 0; }

line_count=$(printf -- '%s' "$output" | wc -l | tr -d ' ')
[[ "$line_count" -le "$MAX_LINES" ]] && { printf -- '%s' "$payload"; exit 0; }

head_part=$(printf -- '%s' "$output" | head -n "$HEAD_KEEP")
tail_part=$(printf -- '%s' "$output" | tail -n "$TAIL_KEEP")
omitted=$((line_count - HEAD_KEEP - TAIL_KEEP))

truncated="${head_part}
… [truncated ${omitted} lines — re-run with a narrower command if the middle matters] …
${tail_part}"

# Write back to the same field position (object.output or plain string)
if jq -e '(.tool_response | type) == "object"' <<<"$payload" >/dev/null 2>&1; then
  jq --arg t "$truncated" '.tool_response.output = $t' <<<"$payload" 2>/dev/null || printf -- '%s' "$payload"
else
  jq --arg t "$truncated" '.tool_response = $t' <<<"$payload" 2>/dev/null || printf -- '%s' "$payload"
fi
