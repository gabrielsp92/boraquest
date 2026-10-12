#!/bin/sh
# Prints the Fly app name from fly.toml after checking that the app belongs to the
# Fly organization FLY_ORG (default boraquest). Exits 1 otherwise, so nothing ever
# runs against an app in another organization. Every Fly command in this repo goes
# through it: `app=$(./scripts/fly-app.sh)` then `flyctl ... -a "$app"`.
# Run from back-end/ (where fly.toml lives). Needs flyctl (CI installs it without the
# `fly` alias) and jq.
set -eu

org="${FLY_ORG:-boraquest}"
app=$(sed -n 's/^app *= *"\([^"]*\)".*/\1/p' fly.toml)
[ -n "$app" ] || { echo "no app name in fly.toml" >&2; exit 1; }

status=$(flyctl status --json -a "$app") || {
  echo "flyctl status failed for Fly app $app (is FLY_API_TOKEN valid? does the app exist? create it with: make fly-app-create)" >&2
  exit 1
}
actual=$(printf '%s' "$status" | jq -r '.Organization.Slug')
if [ "$actual" != "$org" ]; then
  echo "Fly app $app belongs to organization '$actual', not '$org'; refusing to continue" >&2
  exit 1
fi
echo "$app"
