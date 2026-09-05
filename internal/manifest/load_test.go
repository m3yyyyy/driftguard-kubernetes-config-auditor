package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMultiDocumentManifest(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "objects.yaml")
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: one
  namespace: demo
  labels:
    driftguard.io/managed: "true"
---
apiVersion: v1
kind: Service
metadata:
  name: two
  namespace: demo
  labels:
    driftguard.io/managed: "true"
spec:
  selector:
    app: demo
  ports:
    - port: 80
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	objects, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 {
		t.Fatalf("got %d objects, want 2", len(objects))
	}
}

func TestLoadRejectsSecrets(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "secret.yaml")
	content := `apiVersion: v1
kind: Secret
metadata:
  name: credentials
  labels:
    driftguard.io/managed: "true"
stringData:
  password: do-not-read
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "Secret objects are rejected") {
		t.Fatalf("expected Secret rejection, got %v", err)
	}
}

func TestLoadRequiresManagedLabel(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "unmanaged.yaml")
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: unmanaged
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), ManagedLabel) {
		t.Fatalf("expected managed-label error, got %v", err)
	}
}
