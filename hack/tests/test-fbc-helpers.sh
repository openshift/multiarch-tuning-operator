#!/bin/bash
#
# Fixture-based tests for hack/build-indexs.sh and hack/update-graph.sh
# (MULTIARCH-6541). Exercises the real helper entry points against a fixture
# repo layout with stubbed opm/oc binaries. No network, no real catalogs.
#
# Run with: make test-fbc

set -u

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PASS=0
FAIL=0
WORK=""
OUT=$(mktemp)
TEST_IMAGE="quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator-bundle@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; [ -s "$OUT" ] && sed 's/^/    /' "$OUT"; }

setup_work() {
    WORK=$(mktemp -d)
    local dir
    for dir in fbc-v4-16 fbc-v4-22 fbc-v5-0; do
        mkdir -p "$WORK/$dir/catalog/multiarch-tuning-operator"
        cat > "$WORK/$dir/catalog/multiarch-tuning-operator/index.yaml" <<EOF
defaultChannel: stable
description: |
  Multiarch Tuning Operator test fixture catalog ($dir).
icon:
  base64data: iVBORw0KGgoAAAANSUhEUg==
  mediatype: image/png
name: multiarch-tuning-operator
schema: olm.package
---
schema: olm.channel
package: multiarch-tuning-operator
name: stable
entries:
  - name: multiarch-tuning-operator.v1.4.1
    replaces: multiarch-tuning-operator.v1.4.0
  - name: multiarch-tuning-operator.v1.4.0
    replaces: multiarch-tuning-operator.v1.3.4
---
schema: olm.deprecations
package: multiarch-tuning-operator
entries:
  - reference:
      schema: olm.bundle
      name: multiarch-tuning-operator.v1.3.4
    message: "multiarch-tuning-operator.v1.3.4 is deprecated. Please upgrade to a supported version (>= v1.4.0)."
EOF
        cp "$WORK/$dir/catalog/multiarch-tuning-operator/index.yaml" "$WORK/$dir.orig"
    done

    # opm stub: renders a canned olm.bundle carrying the digest of the given
    # image; OPM_STUB_MODE=badrepo emits an unrewritable tag-based reference
    mkdir -p "$WORK/bin"
    cat > "$WORK/bin/opm" <<'EOF'
#!/bin/bash
IMAGE=""
MIGRATE=0
for arg in "$@"; do
    case "$arg" in
        --migrate-level) MIGRATE=1 ;;
        *@sha256:*) IMAGE="$arg" ;;
    esac
done
[ -n "${OPM_STUB_LOG:-}" ] && echo "migrate=$MIGRATE" >> "$OPM_STUB_LOG"
[ -z "$IMAGE" ] && exit 1
DIGEST="${IMAGE##*@sha256:}"
if [ "${OPM_STUB_MODE:-}" == "badrepo" ]; then
    cat <<EOF2
image: quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator-bundle:v1.4.2
name: multiarch-tuning-operator.v1.4.2
package:
  name: multiarch-tuning-operator
  version: 1.4.2
schema: olm.bundle
EOF2
    exit 0
fi
cat <<EOF2
image: quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator-bundle@sha256:${DIGEST}
name: multiarch-tuning-operator.v1.4.2
package:
  name: multiarch-tuning-operator
  version: 1.4.2
relatedImages:
- image: quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator-bundle@sha256:${DIGEST}
  name: ""
schema: olm.bundle
EOF2
EOF
    # oc stub: resolves only the snapshot named "good-snapshot"
    cat > "$WORK/bin/oc" <<'EOF'
#!/bin/bash
[ "$1 $2" == "describe snapshot" ] || exit 1
if [ "$3" == "good-snapshot" ]; then
    echo "Container Image:  quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator-bundle@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
fi
exit 0
EOF
    chmod +x "$WORK/bin/opm" "$WORK/bin/oc"

    cp -r "$REPO_ROOT/hack" "$WORK/hack"
    cp "$REPO_ROOT/Makefile" "$WORK/Makefile"
    export OPM_STUB_LOG="$WORK/opm.log"; : > "$OPM_STUB_LOG"
    export OPM_STUB_MODE=""
    export PATH="$WORK/bin:$PATH"
}

run() { (cd "$WORK" && "$@") > "$OUT" 2>&1; }
run_ok() { local d="$1"; shift; if run "$@"; then pass "$d"; else fail "$d"; fi; }
run_fail() { local d="$1"; shift; if run "$@"; then fail "$d (unexpectedly succeeded)"; else pass "$d"; fi; }
contains() { local f="$1" p="$2" d="$3"; if grep -q "$p" "$WORK/$f"; then pass "$d"; else fail "$d"; fi; }
unchanged() { local d="$1" desc="$2"; if cmp -s "$WORK/$d.orig" "$WORK/$d/catalog/multiarch-tuning-operator/index.yaml"; then pass "$desc"; else fail "$desc"; fi; }
state() { (cd "$WORK" && find . -name index.yaml | sort | xargs sha256sum); }

IDX=fbc-v4-22/catalog/multiarch-tuning-operator/index.yaml
OLD=fbc-v4-16/catalog/multiarch-tuning-operator/index.yaml
NEW=fbc-v5-0/catalog/multiarch-tuning-operator/index.yaml

# ── build-indexs.sh ─────────────────────────────────────────────
echo "── build-indexs.sh ──"

setup_work
run_ok "selected targets update only the selected catalogs" ./hack/build-indexs.sh "$TEST_IMAGE" 4.22 fbc-v5-0
contains "$IDX" "image: registry.redhat.io/multiarch-tuning/multiarch-tuning-operator-bundle@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" "selected catalog references the publication repo with the exact digest"
contains "$NEW" "registry.redhat.io/multiarch-tuning/multiarch-tuning-operator-bundle@sha256:bbbb" "additional configured newer target updated"
! grep -q "quay.io/redhat-user-workloads" "$WORK/$IDX" && pass "no internal references remain" || fail "no internal references remain"
unchanged fbc-v4-16 "excluded older catalog remains byte-for-byte unchanged"

setup_work
run_fail "unknown target is rejected" ./hack/build-indexs.sh "$TEST_IMAGE" fbc-v4-99
unchanged fbc-v4-16 "no partial edits on unknown target"

setup_work
run_fail "malformed target is rejected" ./hack/build-indexs.sh "$TEST_IMAGE" foo
setup_work
run_fail "image without digest is rejected" ./hack/build-indexs.sh "quay.io/redhat-user-workloads/multiarch-tuning-operator-bundle:v1.4.2" 4.22
setup_work
run_fail "unresolvable snapshot is rejected" ./hack/build-indexs.sh "no-such-snapshot"

setup_work
run_fail "RHEL 10 input cannot target an older catalog" env FBC_OS=rhel10 ./hack/build-indexs.sh "$TEST_IMAGE" 4.16
unchanged fbc-v4-16 "no partial edits on ineligible RHEL 10 target"

setup_work
run_ok "RHEL 10 default restricts selection to eligible catalogs" env FBC_OS=rhel10 ./hack/build-indexs.sh "$TEST_IMAGE"
contains "$IDX" "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" "eligible catalog updated"
unchanged fbc-v4-16 "ineligible older catalog untouched by RHEL 10 default"

setup_work
run_ok "legacy invocation (no targets) updates every catalog" ./hack/build-indexs.sh "$TEST_IMAGE"
contains "$OLD" "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" "legacy default includes the older catalog"
AFTER_FIRST=$(state)
run_ok "rerun with the same bundle is idempotent" ./hack/build-indexs.sh "$TEST_IMAGE"
[ "$(state)" == "$AFTER_FIRST" ] && pass "no catalog changes on rerun" || fail "no catalog changes on rerun"

setup_work
export OPM_STUB_MODE=badrepo
run_fail "unrewritable internal reference fails verification" ./hack/build-indexs.sh "$TEST_IMAGE" 4.22
unset OPM_STUB_MODE
unchanged fbc-v4-22 "no partial edits on rewrite verification failure"

setup_work
run_ok "make build-index forwards TARGETS" env make -C "$WORK" build-index SNAPSHOT="$TEST_IMAGE" TARGETS=4.22
contains "$IDX" "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" "make target updated the selected catalog"
unchanged fbc-v4-16 "make target left the excluded catalog unchanged"

# ── update-graph.sh ─────────────────────────────────────────────
echo "── update-graph.sh ──"

setup_work
run_ok "explicit target adds the channel entry" ./hack/update-graph.sh 1.4.2 4.22
contains "$IDX" "  - name: multiarch-tuning-operator.v1.4.2" "new entry added to the selected catalog"
contains "$IDX" "replaces: multiarch-tuning-operator.v1.4.1" "new entry replaces the previous head"
(cd "$WORK" && awk '/^schema: olm.channel/{c=1} c && /^entries:/{getline; print; exit}' "$IDX") | grep -q "v1.4.2" \
    && pass "entry inserted under the olm.channel entries key" || fail "entry inserted under the olm.channel entries key"
unchanged fbc-v4-16 "excluded catalog remains byte-for-byte unchanged"
unchanged fbc-v5-0 "unselected newer catalog remains byte-for-byte unchanged"

setup_work
run_ok "first invocation succeeds" ./hack/update-graph.sh v1.4.2 4.22
AFTER_GRAPH=$(state)
run_ok "repeat invocation is idempotent" ./hack/update-graph.sh v1.4.2 4.22
[ "$(state)" == "$AFTER_GRAPH" ] && pass "no duplicate entries on rerun" || fail "no duplicate entries on rerun"

setup_work
run_fail "invalid version is rejected" ./hack/update-graph.sh 1.4 4.22
setup_work
run_fail "unknown target is rejected" ./hack/update-graph.sh 1.4.2 fbc-v4-99
setup_work
run_fail "RHEL 10 input cannot target an older catalog" env FBC_OS=rhel10 ./hack/update-graph.sh 1.4.2 4.16
setup_work
run_ok "commit-based patch version is accepted" ./hack/update-graph.sh 1.4.2+abc1234 4.22
contains "$IDX" "  - name: multiarch-tuning-operator.v1.4.2+abc1234" "commit-suffixed entry added"

# ── Summary ─────────────────────────────────────────────────────
echo ""
echo "Results: $PASS passed, $FAIL failed"

[ -n "$WORK" ] && rm -rf "$WORK"
rm -f "$OUT"
(( FAIL > 0 )) && exit 1
exit 0
