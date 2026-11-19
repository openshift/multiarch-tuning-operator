#!/bin/bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <version> [targets...]" >&2
    exit 1
fi
INPUT_VERSION="$1"
shift
# Explicit catalog targets (directories or OCP versions, e.g. "fbc-v4-22 4.23").
# Empty means every configured catalog (legacy behavior kept for existing callers).
TARGETS="${*:-${FBC_TARGETS:-}}"
TARGETS="${TARGETS//,/ }"
OS_FLAVOR="${FBC_OS:-rhel9}"

# Release configuration (MULTIARCH-6541)
CONFIGURED_TARGETS="fbc-v4-16 fbc-v4-17 fbc-v4-18 fbc-v4-19 fbc-v4-20 fbc-v4-21 fbc-v4-22 fbc-v4-23 fbc-v5-0"
RHEL10_MIN_OCP="4.22"

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

# Body of the olm.channel section of an index.yaml
channel_section() {
    awk '
        /^schema: olm.channel$/ { in_channel = 1; next }
        (/^schema: / || /^---$/) && in_channel { exit }
        in_channel { print }
    ' "$1"
}

# Ensure version starts with "v"
if [[ "$INPUT_VERSION" != v* ]]; then
  VERSION="v$INPUT_VERSION"
else
  VERSION="$INPUT_VERSION"
fi

if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(\+[A-Za-z0-9][A-Za-z0-9._-]*)?$ ]]; then
    echo "Error: invalid version '$INPUT_VERSION' (expected X.Y.Z, optionally with a +commit suffix)"
    exit 1
fi

ENTRY_NAME="multiarch-tuning-operator.$VERSION"

TARGETS=$(select_targets) || exit 1

for dir in $TARGETS; do
  YAML_PATH="$dir/catalog/multiarch-tuning-operator/index.yaml"

  if [[ ! -f "$YAML_PATH" ]]; then
    echo "Error: $YAML_PATH not found"
    exit 1
  fi

  echo "Processing $YAML_PATH"

  # Skip catalogs whose channel already contains this entry (idempotent reruns)
  if channel_section "$YAML_PATH" | grep -q "name: $ENTRY_NAME\$"; then
    echo "$YAML_PATH already contains $ENTRY_NAME, skipping"
    continue
  fi

  # Extract head entry of the stable channel
  PREVIOUS_ENTRY=$(channel_section "$YAML_PATH" | awk '
    $1 == "entries:" { in_entries=1; next }
    in_entries && $1 == "-" && $2 == "name:" { print $3; exit }
  ')

  if [[ -z "$PREVIOUS_ENTRY" ]]; then
    echo "Error: No previous entry found in $YAML_PATH"
    exit 1
  fi

  if [[ "$PREVIOUS_ENTRY" == "$ENTRY_NAME" ]]; then
    echo "Error: refusing to make $ENTRY_NAME replace itself in $YAML_PATH"
    exit 1
  fi

  # Insert the new entry after the entries: key of the olm.channel section,
  # preserving every other byte of the file
  awk -v name="$ENTRY_NAME" -v replaces="$PREVIOUS_ENTRY" '
    BEGIN { in_channel = 0; inserted = 0; insert_after = -1 }
    /^schema: olm.channel$/ { in_channel = 1 }
    (/^schema: / || /^---$/) && in_channel && !/^schema: olm.channel$/ { in_channel = 0 }
    {
      lines[NR] = $0
      if (in_channel && $1 == "entries:" && !inserted) {
        insert_after = NR
        inserted = 1
      }
    }
    END {
      if (!inserted) exit 1
      for (i = 1; i <= NR; i++) {
        printf "%s", lines[i]
        if (i == insert_after) printf "\n  - name: %s\n    replaces: %s", name, replaces
        if (i < NR) printf "\n"
      }
    }
  ' "$YAML_PATH" > "$YAML_PATH.tmp" && mv "$YAML_PATH.tmp" "$YAML_PATH"

  echo "Updated $YAML_PATH with new entry: $ENTRY_NAME replaces $PREVIOUS_ENTRY"
done

echo "Done."
