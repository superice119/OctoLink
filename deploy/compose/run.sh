#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ── Bootstrap: copy .env.*.example → .env.* if missing ──────────────────────
echo "Checking env files..."
bootstrapped=0
for example in .env.*.example; do
  target="${example%.example}"
  if [ ! -f "$target" ]; then
    cp "$example" "$target"
    echo "  Created $target from $example"
    bootstrapped=1
  fi
done
if [ "$bootstrapped" -eq 0 ]; then
  echo "  All env files already present."
fi

# ── Verify NATS credentials are consistent across env files ─────────────────
# (optional: warn user if they customised one but not the others)
NATS_PW_IN_NATS=$(grep '^NATS_PW=' .env.nats 2>/dev/null | cut -d= -f2)
if [ "${NATS_PW_IN_NATS}" = "change_me_in_production" ]; then
  echo ""
  echo "⚠  WARNING: NATS password is still the default dev placeholder."
  echo "   Edit .env.nats and update NATS_PW, then update NATS_URL in other .env.* files."
  echo "   (For local dev/QA this is fine; DO NOT use in production.)"
  echo ""
fi

# ── Start the stack ──────────────────────────────────────────────────────────
COMPOSE_PROFILES=nats,controller,cwmp,mqtt,stomp,ws,adapter,frontend,portainer docker compose up -d
