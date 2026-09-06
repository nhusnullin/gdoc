#!/usr/bin/env bash
# revmux-review.sh - custom external review script for ralphex.
#
# Runs revmux (umputun/revmux) instead of codex for ralphex's external review
# phase. ralphex hands us a rendered prompt file (diff instruction, goal, plan
# file, previous review context); we turn that into a revmux task round and
# report its findings back in the plain-text shape ralphex expects.
#
# config (.ralphex/config):
#   external_review_tool = custom
#   custom_review_script = .ralphex/scripts/revmux-review.sh
#
# revmux task rounds stack: reusing one task per feature branch across
# iterations lets revmux carry earlier rounds' findings into the next round,
# same as ralphex's own {{PREVIOUS_REVIEW_CONTEXT}}.

set -euo pipefail

command -v revmux >/dev/null 2>&1 || { echo "error: revmux is required but not found" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "error: jq is required but not found" >&2; exit 1; }

prompt_file="${1:-}"
if [[ -z "$prompt_file" || ! -f "$prompt_file" ]]; then
    echo "error: prompt file not provided or not found" >&2
    exit 1
fi

branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo detached)
task=$(printf '%s' "$branch" | tr -c 'a-zA-Z0-9._-' '-')

tasks_dir="./.revmux/tasks"
task_round_count=0
if [[ -d "$tasks_dir/$task" ]]; then
    task_round_count=$(find "$tasks_dir/$task" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
fi
run=$(printf 'round-%02d' "$((task_round_count + 1))")

new_out=$(revmux new --task "$task" --run "$run")
scope_path=$(echo "$new_out" | jq -r .scope)
cp "$prompt_file" "$scope_path"

set +e
report_json=$(revmux --task "$task" --run "$run" --no-tui)
revmux_exit=$?
set -e

# exit 0 = no findings, 1 = findings reported, 2+ = a real tool error
if [[ "$revmux_exit" -ge 2 ]]; then
    echo "error: revmux failed (exit $revmux_exit)" >&2
    echo "$report_json" >&2
    exit 1
fi

if ! echo "$report_json" | jq -e '.findings' >/dev/null 2>&1; then
    echo "error: revmux produced no parseable report" >&2
    echo "$report_json" >&2
    exit 1
fi

finding_count=$(echo "$report_json" | jq '.findings | length')
if [[ "$finding_count" -eq 0 ]]; then
    echo "NO ISSUES FOUND"
    exit 0
fi

echo "$report_json" | jq -r '
  .findings[]
  | "\(.file):\(.line) - \(.title): \(.body) (fix: \(.fix)) [\(.severity), \(.verdict)]"
'
