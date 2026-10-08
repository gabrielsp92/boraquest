#!/bin/sh
# Fails unless total statement coverage in coverage.out meets the threshold.
set -eu
threshold="${COVERAGE_THRESHOLD:-100}"
total=$(go tool cover -func=coverage.out | awk '/^total:/ {sub("%", "", $3); print $3}')
echo "total coverage: ${total}% (required: ${threshold}%)"
awk -v t="$total" -v min="$threshold" 'BEGIN { exit (t + 0 >= min + 0) ? 0 : 1 }' || {
  echo "coverage below threshold" >&2
  exit 1
}
