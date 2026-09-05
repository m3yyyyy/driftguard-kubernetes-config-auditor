# Architecture

DriftGuard keeps manifest parsing, Kubernetes access, comparison, and presentation separate so the security boundary is easy to review.

## Flow

1. The manifest loader reads a bounded, sorted set of YAML or JSON files.
2. Local validation rejects unsupported or unsafe objects before cluster access.
3. The Kubernetes adapter resolves each object's Group-Version-Kind through discovery.
4. Exact `get` calls retrieve desired resources, while label-scoped `list` calls find unexpected managed resources.
5. The comparison engine walks only desired fields and emits changed JSON paths.
6. The report renderer produces table, JSON, or Markdown output.

## Comparison semantics

DriftGuard intentionally uses desired-subset comparison. Fields present only in the live object are usually API defaults or controller state and do not count as drift. Fields declared in Git must exist and match in the live object.

The comparison ignores the top-level `status` field and well-known runtime metadata such as resource versions, UIDs, timestamps, generations, managed fields, and deployment revision annotations.

## Unexpected resources

Unexpected-resource discovery never scans an entire cluster indiscriminately. It lists only the resource kinds and namespaces represented in the desired set, using the selector `driftguard.io/managed=true`.

## Determinism

Input files, resource scopes, findings, map keys, and changed paths are sorted. The live timestamp and cluster state naturally vary; the report envelope and classification rules do not.
