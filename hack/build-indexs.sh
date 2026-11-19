#!/bin/bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <snapshot-or-image> [targets...]" >&2
    exit 1
fi
INPUT="$1"
shift
# Explicit catalog targets (directories or OCP versions, e.g. "fbc-v4-22 4.23").
# Empty means every configured catalog (legacy behavior kept for the release pipeline).
TARGETS="${*:-${FBC_TARGETS:-}}"
TARGETS="${TARGETS//,/ }"
OS_FLAVOR="${FBC_OS:-rhel9}"

# Release configuration (MULTIARCH-6541)
PUBLICATION_REPO="registry.redhat.io/multiarch-tuning/multiarch-tuning-operator-bundle"
CONFIGURED_TARGETS="fbc-v4-16 fbc-v4-17 fbc-v4-18 fbc-v4-19 fbc-v4-20 fbc-v4-21 fbc-v4-22 fbc-v4-23 fbc-v5-0"
RHEL10_MIN_OCP="4.22"

# Function to extract bundle image from snapshot
get_bundle_image_from_snapshot() {
    local snapshot_name="$1"
    echo "Getting bundle image from snapshot: $snapshot_name" >&2
    oc describe snapshot "$snapshot_name" | \
        awk '/Container Image:/ && /bundle/ {print $NF; exit}'
}

normalize_target() {
    case "$1" in
        fbc-v[0-9]*-[0-9]*) echo "$1" ;;
        [0-9]*.[0-9]*|v[0-9]*.[0-9]*) echo "fbc-v${1#v}" | tr '.' '-' ;;
        *) echo "Error: malformed target '$1' (expected fbc-v4-22 or 4.22)" >&2; return 1 ;;
    esac
}

target_version() { echo "${1#fbc-v}" | tr '-' '.'; }

version_ge() {
    local IFS=.
    local -a a=($1) b=($2)
    local i
    for i in 0 1 2; do
        (( ${a[$i]:-0} > ${b[$i]:-0} )) && return 0
        (( ${a[$i]:-0} < ${b[$i]:-0} )) && return 1
    done
    return 0
}

# Resolve and validate the catalog targets before any catalog is modified
select_targets() {
    local -a dirs=()
    local token dir normalized

    if [[ -z "${TARGETS// /}" ]]; then
        echo "Notice: no explicit targets given; updating all configured catalogs present on disk" >&2
        for dir in $CONFIGURED_TARGETS; do
            [[ -d "$dir" ]] || continue
            # RHEL 10 inputs only update eligible catalogs
            if [[ "$OS_FLAVOR" != "rhel10" ]] || version_ge "$(target_version "$dir")" "$RHEL10_MIN_OCP"; then
                dirs+=("$dir")
            fi
        done
    else
        for token in $TARGETS; do
            normalized=$(normalize_target "$token") || exit 1
            echo " $CONFIGURED_TARGETS " | grep -q " $normalized " || {
                echo "Error: unknown target '$token' (configured: $CONFIGURED_TARGETS)" >&2
                exit 1
            }
            [[ -d "$normalized" ]] || {
                echo "Error: catalog directory '$normalized' not found" >&2
                exit 1
            }
            if [[ "$OS_FLAVOR" == "rhel10" ]] && ! version_ge "$(target_version "$normalized")" "$RHEL10_MIN_OCP"; then
                echo "Error: '$normalized' (OCP $(target_version "$normalized")) is not eligible for RHEL 10 releases; requires OCP >= $RHEL10_MIN_OCP" >&2
                exit 1
            fi
            dirs+=("$normalized")
        done
    fi
    printf '%s\n' "${dirs[@]}"
}

# Detect if input is a snapshot or a direct image URL
if [[ "$INPUT" == quay.io/* || "$INPUT" == *"@"* ]]; then
    echo "Using provided bundle image URL directly."
    BUNDLE_IMAGE="$INPUT"
else
    echo "Input appears to be a snapshot. Extracting bundle image..."
    BUNDLE_IMAGE=$(get_bundle_image_from_snapshot "$INPUT")
    if [[ -z "$BUNDLE_IMAGE" ]]; then
        echo "Error: Could not find bundle image in snapshot $INPUT"
        exit 1
    fi
    echo "Found bundle image: $BUNDLE_IMAGE"
fi

# The publication rewrite preserves the exact digest, so the input must be digest-pinned
if ! [[ "$BUNDLE_IMAGE" =~ @sha256:([a-f0-9]{64})$ ]]; then
    echo "Error: bundle image '$BUNDLE_IMAGE' is not digest-pinned (@sha256:<digest>)" >&2
    exit 1
fi
BUNDLE_DIGEST="${BASH_REMATCH[1]}"

TARGETS=$(select_targets) || exit 1

# Yellow color escape code
YELLOW='\033[1;33m'
# Reset color escape code
NC='\033[0m'

echo -e "${YELLOW}Warning: verify opm version is updated${NC}"

# Loop over the selected catalogs
for dir in $TARGETS; do
    echo "Processing directory: $dir"

    # Extract version from directory name (e.g., 4-16 from fbc-v4-16)
    version=$(echo "$dir" | sed -E 's/^fbc-v([0-9]+-[0-9]+).*/\1/')

    pushd "$dir/catalog/multiarch-tuning-operator" > /dev/null

    if [[ ! -f index.yaml ]]; then
        echo "Warning: index.yaml not found in $dir/catalog/multiarch-tuning-operator"
        popd > /dev/null
        continue
    fi

    # Skip catalogs that already contain this bundle
    if grep -q "sha256:${BUNDLE_DIGEST}" index.yaml; then
        echo "Bundle sha256:${BUNDLE_DIGEST:0:12}... already present, skipping"
        popd > /dev/null
        continue
    fi

    render_tmp=$(mktemp)

    if [[ "$version" < "4-17" ]]; then
        opm render --output=yaml "$BUNDLE_IMAGE" > "$render_tmp"
    else
        echo "Using migrate-level option for version $version"
        opm render --output=yaml --migrate-level bundle-object-to-csv-metadata "$BUNDLE_IMAGE" > "$render_tmp"
    fi

    # Fix top-level image field
    sed -i -E "s|(image: )quay.io/redhat-user-workloads[^@]*multiarch-tuning-operator-bundle[^@]*@sha256:([a-f0-9]{64})|\1${PUBLICATION_REPO}@sha256:\2|" "$render_tmp"

    # Fix relatedImages entries
    sed -i -E "s|(- image: )quay.io/redhat-user-workloads[^@]*multiarch-tuning-operator-bundle[^@]*@sha256:([a-f0-9]{64})|\1${PUBLICATION_REPO}@sha256:\2|" "$render_tmp"

    # Verify the rewrite before touching the catalog: the exact digest must survive
    # and no internal bundle reference may remain
    if ! grep -q "image: ${PUBLICATION_REPO}@sha256:${BUNDLE_DIGEST}" "$render_tmp" || \
       grep -q "quay.io/redhat-user-workloads[^@]*multiarch-tuning-operator-bundle" "$render_tmp"; then
        echo "Error: rewrite verification failed for $dir: expected ${PUBLICATION_REPO}@sha256:${BUNDLE_DIGEST}" >&2
        rm -f "$render_tmp"
        popd > /dev/null
        exit 1
    fi

    echo "" >> index.yaml
    cat "$render_tmp" >> index.yaml
    rm -f "$render_tmp"

    echo "Appended bundle data to $dir/catalog/multiarch-tuning-operator/index.yaml"
    popd > /dev/null
done

echo "Done."
