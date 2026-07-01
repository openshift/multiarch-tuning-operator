#!/bin/bash
# Tests for fbc-update-final-pipeline.yaml and tag-release-commit-pipeline.yaml
# Validates the shell script logic embedded in the Tekton pipeline steps.
#
# Prerequisites:
#   - git remote 'downstream' pointing at openshift/multiarch-tuning-operator
#   - downstream/fbc branch fetched (git fetch downstream fbc)
#
# Usage: bash .tekton/tests/test-pipeline-scripts.sh

set -uo pipefail

PASS=0
FAIL=0
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PIPELINE_FILE="$REPO_ROOT/.tekton/fbc-update-final-pipeline.yaml"
TAG_PIPELINE_FILE="$REPO_ROOT/.tekton/tag-release-commit-pipeline.yaml"

# Create a temp directory for test fixtures
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass() { ((PASS++)); echo "  PASS: $1"; }
fail() { ((FAIL++)); echo "  FAIL: $1"; }

# ── Setup: extract real FBC index.yaml for testing ──────────────────────
echo "Setting up test fixtures..."
git show downstream/fbc:fbc-v4-16/catalog/multiarch-tuning-operator/index.yaml > "$TMPDIR/index-v4-16.yaml" 2>/dev/null || {
    echo "ERROR: Cannot fetch downstream/fbc branch. Run: git fetch downstream fbc"
    exit 1
}
git show downstream/fbc:fbc-v4-17/catalog/multiarch-tuning-operator/index.yaml > "$TMPDIR/index-v4-17.yaml" 2>/dev/null

# ═══════════════════════════════════════════════════════════════════════
# FBC UPDATE FINAL PIPELINE TESTS
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " FBC Update Final Pipeline Tests"
echo "═══════════════════════════════════════════════════════════════"

# ── Test: awk channel extractor only returns channel section ─────────
echo ""
echo "── Channel extractor ──"

CHANNEL_OUTPUT=$(awk '
  /^schema: olm.channel$/ { in_channel=1; next }
  (/^schema: / || /^---$/) && in_channel { exit }
  in_channel { print }
' "$TMPDIR/index-v4-16.yaml")

# Should contain channel entries
if echo "$CHANNEL_OUTPUT" | grep -q "name: multiarch-tuning-operator.v"; then
    pass "Channel extractor finds version entries"
else
    fail "Channel extractor finds version entries"
fi

# Should NOT contain bundle data (image: lines)
if echo "$CHANNEL_OUTPUT" | grep -q "^image:"; then
    fail "Channel extractor excludes bundle image: lines"
else
    pass "Channel extractor excludes bundle image: lines"
fi

# Should NOT contain schema: olm.bundle
if echo "$CHANNEL_OUTPUT" | grep -q "schema: olm.bundle"; then
    fail "Channel extractor excludes schema: olm.bundle"
else
    pass "Channel extractor excludes schema: olm.bundle"
fi

# Should contain entries: keyword
if echo "$CHANNEL_OUTPUT" | grep -q "^entries:"; then
    pass "Channel extractor includes entries: keyword"
else
    fail "Channel extractor includes entries: keyword"
fi

# Count: should find ALL channel entries (currently 7)
ENTRY_COUNT=$(echo "$CHANNEL_OUTPUT" | grep -c "name: multiarch-tuning-operator.v" || true)
if [ "$ENTRY_COUNT" -ge 7 ]; then
    pass "Channel extractor finds all $ENTRY_COUNT entries (>= 7)"
else
    fail "Channel extractor finds all entries (got $ENTRY_COUNT, expected >= 7)"
fi

# Works the same on v4-17 format
CHANNEL_OUTPUT_17=$(awk '
  /^schema: olm.channel$/ { in_channel=1; next }
  (/^schema: / || /^---$/) && in_channel { exit }
  in_channel { print }
' "$TMPDIR/index-v4-17.yaml")

ENTRY_COUNT_17=$(echo "$CHANNEL_OUTPUT_17" | grep -c "name: multiarch-tuning-operator.v" || true)
if [ "$ENTRY_COUNT" -eq "$ENTRY_COUNT_17" ]; then
    pass "Channel extractor consistent between v4-16 and v4-17 ($ENTRY_COUNT entries)"
else
    fail "Channel extractor inconsistent: v4-16=$ENTRY_COUNT, v4-17=$ENTRY_COUNT_17"
fi

# ── Test: channel extractor version matching ─────────────────────────
echo ""
echo "── Version matching in channel ──"

# Existing version should be found
if echo "$CHANNEL_OUTPUT" | grep -q "name: multiarch-tuning-operator.v1.3.0$"; then
    pass "Finds existing version v1.3.0 in channel"
else
    fail "Finds existing version v1.3.0 in channel"
fi

# Non-existent version should NOT be found
if echo "$CHANNEL_OUTPUT" | grep -q "name: multiarch-tuning-operator.v9.9.9$"; then
    fail "Rejects non-existent version v9.9.9"
else
    pass "Rejects non-existent version v9.9.9"
fi

# Partial version match should not match (v1.3.0 should not match v1.3.0+abc)
if echo "$CHANNEL_OUTPUT" | grep -q "name: multiarch-tuning-operator.v1.3.0+"; then
    fail "Does not false-match patch version v1.3.0+"
else
    pass "Does not false-match patch version v1.3.0+"
fi

# ── Test: bundle block removal ───────────────────────────────────────
echo ""
echo "── Bundle block removal ──"

# Create test fixture with multiple blocks
cat > "$TMPDIR/test-removal.yaml" << 'FIXTURE'
---
defaultChannel: stable
name: test-operator
schema: olm.package
---
schema: olm.channel
name: stable
entries:
  - name: test-operator.v1.0.0
  - name: test-operator.v1.1.0
---
image: quay.io/test@sha256:aaa
name: test-operator.v1.0.0
package: test-operator
properties:
  - type: olm.package
    value:
      packageName: test-operator
      version: 1.0.0
schema: olm.bundle
---
image: quay.io/test@sha256:bbb
name: test-operator.v1.1.0
package: test-operator
properties:
  - type: olm.package
    value:
      packageName: test-operator
      version: 1.1.0
schema: olm.bundle
FIXTURE

# Helper: remove a bundle block by name from an index.yaml
remove_bundle() {
  awk -v bundle="$2" '
  BEGIN { buf = ""; skip = 0 }
  /^---$/ {
    if (!skip && buf != "") { printf "%s", buf }
    buf = $0 "\n"; skip = 0; next
  }
  { buf = buf $0 "\n" }
  $0 == "name: " bundle { skip = 1 }
  END { if (!skip && buf != "") printf "%s", buf }
  ' "$1"
}

# Remove v1.0.0 bundle
REMOVAL_OUTPUT=$(remove_bundle "$TMPDIR/test-removal.yaml" "test-operator.v1.0.0")

# Should still contain v1.1.0 bundle
if echo "$REMOVAL_OUTPUT" | grep -q "name: test-operator.v1.1.0"; then
    pass "Bundle removal preserves non-target bundle (v1.1.0)"
else
    fail "Bundle removal preserves non-target bundle (v1.1.0)"
fi

# Should NOT contain v1.0.0 bundle
if echo "$REMOVAL_OUTPUT" | grep -q "image: quay.io/test@sha256:aaa"; then
    fail "Bundle removal removes target bundle image (v1.0.0)"
else
    pass "Bundle removal removes target bundle image (v1.0.0)"
fi

# Should preserve channel entries (even though they reference v1.0.0)
if echo "$REMOVAL_OUTPUT" | grep -q "name: test-operator.v1.0.0"; then
    # Channel entry has "  - name: test-operator.v1.0.0" (indented)
    # This is OK — we only remove the bundle block, not channel entries
    pass "Bundle removal preserves channel entry referencing same version"
else
    fail "Bundle removal preserves channel entry referencing same version"
fi

# Should preserve package block
if echo "$REMOVAL_OUTPUT" | grep -q "schema: olm.package"; then
    pass "Bundle removal preserves olm.package block"
else
    fail "Bundle removal preserves olm.package block"
fi

# Should preserve channel block
if echo "$REMOVAL_OUTPUT" | grep -q "schema: olm.channel"; then
    pass "Bundle removal preserves olm.channel block"
else
    fail "Bundle removal preserves olm.channel block"
fi

# Removing non-existent bundle should preserve everything
NOOP_OUTPUT=$(remove_bundle "$TMPDIR/test-removal.yaml" "test-operator.v9.9.9")

ORIG_LINES=$(wc -l < "$TMPDIR/test-removal.yaml")
NOOP_LINES=$(echo "$NOOP_OUTPUT" | wc -l)
if [ "$ORIG_LINES" -eq "$NOOP_LINES" ]; then
    pass "Bundle removal is no-op for non-existent version (line count preserved)"
else
    fail "Bundle removal is no-op for non-existent version (orig=$ORIG_LINES, got=$NOOP_LINES)"
fi

# ── Test: insertion awk in_channel boundary ──────────────────────────
echo ""
echo "── Insertion awk in_channel boundary ──"

# Simulate inserting a new entry — write output to file for reliable comparison
INSERT_FILE="$TMPDIR/insert-output.yaml"
awk -v entry="multiarch-tuning-operator.v1.4.0" -v prev="1.3.0" '
/^  - name: / && !inserted && in_channel {
  print "  - name: " entry
  print "    replaces: multiarch-tuning-operator.v" prev
  inserted = 1
}
/^schema: olm.channel$/ { in_channel = 1 }
(/^schema: / && !/^schema: olm.channel$/) || /^---$/ { in_channel = 0 }
{ print }
' "$TMPDIR/index-v4-16.yaml" > "$INSERT_FILE"

# Should have exactly one insertion
INSERT_COUNT=$(grep -c "name: multiarch-tuning-operator.v1.4.0" "$INSERT_FILE" || true)
if [ "$INSERT_COUNT" -eq 1 ]; then
    pass "Insertion awk inserts exactly once"
else
    fail "Insertion awk inserts exactly once (got $INSERT_COUNT)"
fi

# New entry should come before v1.3.0
FIRST_ENTRY=$(grep "name: multiarch-tuning-operator.v" "$INSERT_FILE" | head -1)
if echo "$FIRST_ENTRY" | grep -q "v1.4.0"; then
    pass "Insertion awk inserts before existing first entry"
else
    fail "Insertion awk inserts before existing first entry (first was: $FIRST_ENTRY)"
fi

# replaces field should be present
if grep -q "replaces: multiarch-tuning-operator.v1.3.0" "$INSERT_FILE"; then
    pass "Insertion awk includes replaces field"
else
    fail "Insertion awk includes replaces field"
fi

# Verify insertion added exactly 2 lines (normalize trailing newlines for comparison)
ORIG_NORMALIZED="$TMPDIR/orig-normalized.yaml"
sed -e '$a\' "$TMPDIR/index-v4-16.yaml" > "$ORIG_NORMALIZED" 2>/dev/null
DIFF_ADDED=$(diff "$ORIG_NORMALIZED" "$INSERT_FILE" | grep "^>" | wc -l)
if [ "$DIFF_ADDED" -eq 2 ]; then
    pass "Insertion awk adds exactly 2 new lines (no other changes)"
else
    fail "Insertion awk adds exactly 2 new lines (got $DIFF_ADDED added lines)"
fi

# ═══════════════════════════════════════════════════════════════════════
# TAG RELEASE COMMIT PIPELINE TESTS
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Tag Release Commit Pipeline Tests"
echo "═══════════════════════════════════════════════════════════════"

# ── Test: early tag existence check via git ls-remote ────────────────
echo ""
echo "── Early tag existence check ──"

REMOTE_URL="https://github.com/openshift/multiarch-tuning-operator.git"

# Existing tag should be detected
EXISTING_SHA=$(git ls-remote --tags "$REMOTE_URL" "refs/tags/v1.3.0" 2>/dev/null | awk '{print $1}')
if [ -n "$EXISTING_SHA" ]; then
    pass "git ls-remote detects existing tag v1.3.0 (SHA: ${EXISTING_SHA:0:12})"
else
    fail "git ls-remote detects existing tag v1.3.0"
fi

# Non-existent tag should return empty
NONEXIST_SHA=$(git ls-remote --tags "$REMOTE_URL" "refs/tags/v99.99.99" 2>/dev/null | awk '{print $1}')
if [ -z "$NONEXIST_SHA" ]; then
    pass "git ls-remote returns empty for non-existent tag v99.99.99"
else
    fail "git ls-remote returns empty for non-existent tag v99.99.99 (got: $NONEXIST_SHA)"
fi

# ── Test: URL parameterization ───────────────────────────────────────
echo ""
echo "── URL parameterization ──"

# Without .git suffix
GIT_URL="https://github.com/openshift/multiarch-tuning-operator"
AUTH_GIT_URL="$GIT_URL"
[[ "$AUTH_GIT_URL" == *.git ]] || AUTH_GIT_URL="${AUTH_GIT_URL}.git"
RESULT="https://oauth2:TOKEN@${AUTH_GIT_URL#https://}"
EXPECTED="https://oauth2:TOKEN@github.com/openshift/multiarch-tuning-operator.git"
if [ "$RESULT" = "$EXPECTED" ]; then
    pass "URL parameterization works without .git suffix"
else
    fail "URL parameterization without .git suffix (got: $RESULT)"
fi

# With .git suffix
GIT_URL="https://github.com/openshift/multiarch-tuning-operator.git"
AUTH_GIT_URL="$GIT_URL"
[[ "$AUTH_GIT_URL" == *.git ]] || AUTH_GIT_URL="${AUTH_GIT_URL}.git"
RESULT="https://oauth2:TOKEN@${AUTH_GIT_URL#https://}"
if [ "$RESULT" = "$EXPECTED" ]; then
    pass "URL parameterization works with .git suffix"
else
    fail "URL parameterization with .git suffix (got: $RESULT)"
fi

# ── Test: pipeline YAML structure ────────────────────────────────────
echo ""
echo "── Pipeline YAML structure ──"

# No hardcoded openshift/multiarch-tuning-operator URLs remain (excluding default param values and PR body)
HARDCODED=$(grep -n "github.com/openshift/multiarch-tuning-operator" "$TAG_PIPELINE_FILE" | \
    grep -v "default:" | grep -v "pr create" | grep -v "^#" || true)
if [ -z "$HARDCODED" ]; then
    pass "No hardcoded repo URLs in tag pipeline (excluding defaults)"
else
    fail "Hardcoded repo URLs found in tag pipeline: $HARDCODED"
fi

# extract-version task uses snapshot param (not release-name)
if grep -q "name: snapshot" "$TAG_PIPELINE_FILE" && ! grep -q "release-name" "$TAG_PIPELINE_FILE"; then
    pass "extract-version uses snapshot parameter"
else
    fail "extract-version should use snapshot, not release-name"
fi

# No -test- suffix in version extraction
if grep -q "\-test-" "$TAG_PIPELINE_FILE"; then
    fail "Debug -test- suffix removed from tag pipeline"
else
    pass "Debug -test- suffix removed from tag pipeline"
fi

# Version comes from bundle CSV (yq eval .spec.version)
if grep -q "yq eval.*spec.version" "$TAG_PIPELINE_FILE"; then
    pass "Version extracted from bundle CSV via yq"
else
    fail "Version should be extracted from bundle CSV"
fi

# Early tag check uses git ls-remote
if grep -q "git ls-remote --tags" "$TAG_PIPELINE_FILE"; then
    pass "Early tag existence check uses git ls-remote"
else
    fail "Early tag existence check should use git ls-remote"
fi

# FBC pipeline checks
echo ""
HARDCODED_FBC=$(grep -n "github.com/openshift/multiarch-tuning-operator" "$PIPELINE_FILE" | \
    grep -v "default:" | grep -v "pr create" | grep -v "^#" || true)
if [ -z "$HARDCODED_FBC" ]; then
    pass "No hardcoded repo URLs in FBC pipeline (excluding defaults)"
else
    fail "Hardcoded repo URLs found in FBC pipeline: $HARDCODED_FBC"
fi

# No grep -A patterns remain in FBC pipeline
if grep -q "grep -A" "$PIPELINE_FILE"; then
    fail "No grep -A patterns in FBC pipeline"
else
    pass "No grep -A patterns in FBC pipeline"
fi

# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Results: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
