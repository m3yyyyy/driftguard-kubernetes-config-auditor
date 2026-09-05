# Security policy

Please use GitHub private vulnerability reporting for this repository. Do not include kubeconfig files, bearer tokens, client certificates, private cluster endpoints, Secret manifests, or production audit reports in public issues.

DriftGuard is intentionally read-only and rejects Kubernetes Secrets. A change that introduces mutation, reads secret payloads, prints compared values, or broadens resource listing requires an explicit threat-model update and maintainer review.
