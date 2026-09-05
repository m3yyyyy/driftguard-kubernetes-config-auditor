# Contributing

Before opening a pull request, run:

```powershell
gofmt -w .
go mod tidy
go mod verify
go vet ./...
go test ./...
go build -trimpath ./cmd/driftguard
```

CI additionally runs `go test -race ./...` on Linux. Add tests for behavior changes. Keep cluster access behind the read-only `audit.Reader` interface, preserve the versioned report contract, and never include compared values in findings.
