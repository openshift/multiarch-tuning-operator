This folder is meant to be used for the internal Red Hat builds of a file-based catalog.

## Catalog-update helpers

`hack/build-indexs.sh` and `hack/update-graph.sh` update the catalogs
(`fbc-v*/catalog/multiarch-tuning-operator/index.yaml`) on an explicit,
validated set of release targets (MULTIARCH-6541):

```
hack/build-indexs.sh <snapshot-or-image> [targets...]
hack/update-graph.sh <version> [targets...]

make build-index SNAPSHOT=<snapshot-or-image> [TARGETS=<targets>] [OS=rhel9|rhel10]
make update-graph VERSION=<version>            [TARGETS=<targets>] [OS=rhel9|rhel10]
make test-fbc
```

Targets are catalog directories (`fbc-v4-22`) or OCP versions (`4.22`),
given as positional arguments or via `FBC_TARGETS` (`TARGETS=` for the Make
targets). The configured target set, the publication repository and
the RHEL 10 eligibility floor are defined in the release configuration
block at the top of each script; targets outside that list are rejected
before any catalog is modified. With no targets given, every configured
catalog present on disk is updated (legacy behavior, kept for the
main-branch release pipeline, announced on stderr).

- `build-indexs.sh` requires a digest-pinned bundle image (or a Snapshot
  name), rewrites internal `quay.io/redhat-user-workloads` bundle references
  to `registry.redhat.io/multiarch-tuning/multiarch-tuning-operator-bundle`
  preserving the exact digest, and verifies the rewrite before appending.
  Catalogs already containing the bundle digest are skipped.
- `update-graph.sh` accepts `X.Y.Z` versions (optional `+commit` suffix) and
  inserts the entry at the head of the `stable` channel with `replaces:`
  pointing at the previous head. Catalogs already containing the entry are
  skipped, so reruns are idempotent.
- `OS=rhel10` (or `FBC_OS=rhel10`) restricts targets to catalogs eligible
  for RHEL 10, i.e. OCP >= 4.22 (MULTIARCH-6199).

`make test-fbc` runs the fixture-based tests in `hack/tests/`, which
exercise the real helper entry points with stubbed `opm`/`oc` binaries.
