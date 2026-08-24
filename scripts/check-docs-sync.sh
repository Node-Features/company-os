#!/usr/bin/env bash
# Lightweight documentation/code consistency checks.
#
# This does NOT replace human doc review — it catches the mechanical class
# of drift docs/audit/fixed-doc-sync-audit.md's contradiction list found by
# hand: a missing/invalid Status line, an audit doc nobody indexed, and a
# docs/domain/*.md claiming APPROVED for a Go package that is still a
# zero-type doc-only stub. Exit non-zero (and print every violation, not
# just the first) on any failure so CI shows the whole list in one run.
#
# Run locally: bash scripts/check-docs-sync.sh
set -uo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail=0
note() { echo "FAIL: $*"; fail=1; }

# --- 1. Every ADR has a recognized Status value. -----------------------
# Catches: a new ADR added without a Status line, or a typo'd one
# (e.g. "Aproved") that would silently read as neither proposed nor
# accepted anywhere that greps for the exact word.
for f in docs/adr/ADR-*.md; do
  status=$(grep -m1 -E '^Status:' "$f" | sed -E 's/^Status:[[:space:]]*([A-Z]+).*/\1/')
  case "$status" in
    PROPOSED|APPROVED|REJECTED|SUPERSEDED) ;;
    *) note "$f — missing or unrecognized Status line (got: '${status:-<none>}')" ;;
  esac
done

# --- 2. Every docs/audit/*.md (except README.md) is indexed. -----------
# Catches: a fixed-*.md/gap-*.md written but never linked from
# docs/audit/README.md's table, which is how this repo's own audit
# process expects every finding to be discoverable (docs/audit/README.md:
# "read findings.md once... each [gap doc] is self-contained").
for f in docs/audit/*.md; do
  base=$(basename "$f")
  [ "$base" = "README.md" ] && continue
  grep -qF "($base)" docs/audit/README.md || note "$f — not referenced in docs/audit/README.md's index table"
done

# --- 3. docs/domain/*.md claiming APPROVED has real Go types, unless -----
# --- it's a known, tracked stub. ----------------------------------------
# Catches: a domain doc flipped to APPROVED (or a new one added) whose
# internal/domain/<name> package is still doc-only — the exact class of
# drift docs/audit/backlog-p2-p4.md's P3 "doc-only stubs" row named.
#
# KNOWN_STUB_DOMAINS is a deliberate, visible allowlist, not a way to
# silence this check — shrink it as a stub gets filled in with real
# types; do not grow it without also filing a docs/audit/backlog-p2-p4.md
# row (or equivalent) for the new gap. Last reconciled 2026-08-24.
KNOWN_STUB_DOMAINS="artifact department evaluation evidence metric resource workspace"
domain_go_root="apps/companyd/internal/domain"
for f in docs/domain/*.md; do
  name=$(basename "$f" .md)
  status=$(grep -m1 -E '^Status:' "$f" | sed -E 's/^Status:[[:space:]]*([A-Z]+).*/\1/')
  [ "$status" = "APPROVED" ] || continue
  is_known_stub=0
  for stub in $KNOWN_STUB_DOMAINS; do
    [ "$stub" = "$name" ] && is_known_stub=1 && break
  done
  [ "$is_known_stub" = 1 ] && continue

  pkg="$domain_go_root/$name"
  [ -d "$pkg" ] || continue # not every domain doc maps 1:1 to a package name (e.g. cross-cutting docs) — not this check's job to guess
  if ! grep -rlE '^type ' "$pkg"/*.go >/dev/null 2>&1; then
    note "$f — Status: APPROVED but $pkg/ has no 'type' declarations (looks like an undisclosed doc-only stub — add it to KNOWN_STUB_DOMAINS above with a backlog row, or fill in real types)"
  fi
done

# --- 4. Every KNOWN_STUB_DOMAINS entry really is still a stub. ---------
# The inverse of #3: once someone fills in real types, this check forces
# KNOWN_STUB_DOMAINS to shrink instead of silently going stale itself.
for stub in $KNOWN_STUB_DOMAINS; do
  pkg="$domain_go_root/$stub"
  if [ -d "$pkg" ] && grep -rlE '^type ' "$pkg"/*.go >/dev/null 2>&1; then
    note "internal/domain/$stub now has real types but is still listed in KNOWN_STUB_DOMAINS — remove it from the allowlist in this script and update docs/domain/$stub.md + its doc.go comment"
  fi
done

if [ "$fail" -eq 0 ]; then
  echo "check-docs-sync: all checks passed"
fi
exit "$fail"
