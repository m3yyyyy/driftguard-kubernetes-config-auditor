# Threat model

## Protected assets

- Kubernetes API credentials and kubeconfig material
- Secret values and application configuration values
- Cluster availability and workload state
- Integrity of audit evidence

## Trust assumptions

- The selected kubeconfig context belongs to the intended cluster.
- The Kubernetes API server and local kubeconfig are trusted.
- Desired manifests come from a reviewed repository.
- The operator reviews the selected context before auditing.

## Controls

- DriftGuard implements no create, update, patch, delete, exec, log, or port-forward operation.
- Every live request is discovery, `get`, or a managed-label-scoped `list`.
- `Secret` manifests are rejected before a client is created.
- Compared values are never included in findings; only JSON paths are reported.
- Unknown or unreadable live state is classified as an error and exits non-zero.
- Input count and individual file size are bounded.
- Output files are created with owner-only permissions where the operating system supports them.

## Residual risks

- A kubeconfig may grant more permissions than DriftGuard needs. Use a least-privilege identity in shared environments.
- Resource names, namespaces, kinds, changed field paths, cluster context, and network timing can still be operationally sensitive.
- A compromised API server can provide false state.
- Desired-subset comparison intentionally ignores live-only fields. It is not a full admission-policy engine.
- Array ordering is significant and may create drift if an API legitimately reorders a declared list.

Review reports before sharing them and run DriftGuard with ordinary user privileges.
