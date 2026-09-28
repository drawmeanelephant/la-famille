#!/bin/bash
# Run the Go suite while retaining machine-readable package/test timing for
# CI artifacts and the Actions summary.
set -euo pipefail

report="${TEST_METRICS_FILE:-${RUNNER_TEMP:-/tmp}/la-famille-test.jsonl}"
mkdir -p "$(dirname "$report")"
go test -json ./... > "$report"

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  python3 - "$report" "$GITHUB_STEP_SUMMARY" <<'PY'
import json
import sys

report_path, summary_path = sys.argv[1:]
packages = []
with open(report_path, encoding="utf-8") as report:
    for line in report:
        event = json.loads(line)
        if event.get("Action") == "pass" and event.get("Package") and "Test" not in event:
            packages.append((event["Package"], event.get("Elapsed", 0)))

total = sum(float(elapsed or 0) for _, elapsed in packages)
with open(summary_path, "a", encoding="utf-8") as summary:
    summary.write("## Go test performance\n")
    summary.write(f"- Packages passed: **{len(packages)}**\n")
    summary.write(f"- Package test time: **{total:.2f}s**\n")
PY
fi

echo "test timing report: $report"
