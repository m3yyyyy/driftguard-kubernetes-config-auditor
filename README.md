# DriftGuard

DriftGuard is a read-only Kubernetes configuration drift auditor. It compares manifests stored in Git with live cluster resources and produces deterministic, automation-friendly findings without printing compared values.

## What it detects

- **In sync**: every field declared in the desired manifest matches the live resource.
- **Modified**: one or more desired fields differ from the live resource.
- **Missing**: a desired resource does not exist in the cluster.
- **Unexpected**: a live resource labelled `driftguard.io/managed=true` has no matching desired manifest.
- **Error**: the cluster could not provide enough evidence to classify a resource safely.

Kubernetes adds defaults and controller-owned fields at runtime. DriftGuard compares the desired fields as a subset of the live object, ignores known runtime metadata and ignores `status`.

## Safety boundaries

- Uses only Kubernetes discovery, `get`, and label-scoped `list` operations.
- Never applies, patches, deletes, scales, or restarts resources.
- Rejects `Secret` manifests instead of reading or reporting sensitive values.
- Reports changed JSON paths, never the old or new values.
- Requires every desired object to carry `driftguard.io/managed: "true"` so unexpected-resource detection stays explicitly scoped.
- Caps input at 100 objects and 1 MiB per manifest file.

See [Threat model](docs/threat-model.md) for the complete trust boundary.

## Requirements

- Go 1.27 or a downloaded release binary
- Access to a Kubernetes cluster through a standard kubeconfig
- Read permission for the resource types being audited

For the local demo, use Docker Desktop, `kubectl`, and `kind`.

## Build and test

```powershell
go mod download
go test ./...
go vet ./...
go build -trimpath -o bin/driftguard.exe ./cmd/driftguard
```

CI additionally runs the Go race detector on Linux.

## Commands

Validate manifests without contacting a cluster:

```powershell
go run ./cmd/driftguard validate --path examples/desired
```

Audit the current cluster context:

```powershell
go run ./cmd/driftguard audit --path examples/desired
```

Choose a context and save JSON evidence:

```powershell
go run ./cmd/driftguard audit --path examples/desired --context kind-driftguard --format json --output reports/audit.json
```

Supported report formats are `table`, `json`, and `markdown`.

## Exit codes

| Code | Meaning |
|---:|---|
| `0` | Audit completed and all managed resources are in sync |
| `2` | Modified, missing, or unexpected resources were detected |
| `64` | Invalid command arguments or desired manifests |
| `70` | Kubernetes evidence or another internal operation failed |

## Local drift demonstration

Create a local cluster:

```powershell
kind create cluster --name driftguard --wait 5m
```

Apply the desired state:

```powershell
kubectl apply -f examples/desired
kubectl wait --for=condition=Available deployment/web -n driftguard-demo --timeout=120s
```

Run a clean audit:

```powershell
go run ./cmd/driftguard audit --path examples/desired --context kind-driftguard
```

Introduce controlled drift:

```powershell
kubectl scale deployment/web -n driftguard-demo --replicas=3
go run ./cmd/driftguard audit --path examples/desired --context kind-driftguard
```

The second audit exits with code `2` and reports `/spec/replicas`, without exposing values. Restore the desired state afterward:

```powershell
kubectl apply -f examples/desired
```

Delete the disposable cluster when it is no longer needed:

```powershell
kind delete cluster --name driftguard
```

## Reports

JSON reports use the versioned `driftguard/v1` schema. Findings are sorted by resource identity, and changed paths use JSON Pointer notation. A timestamp and live cluster context are included so reports can be retained as audit evidence.

## Continuous integration and releases

Pull requests and pushes to `main` run formatting, dependency verification, static analysis, race-enabled tests, and builds on Windows, Linux, and macOS.

Tags matching `v*` publish Windows, Linux, Intel macOS, and Apple Silicon macOS binaries with a `SHA256SUMS` file.

## License

MIT
