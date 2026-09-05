package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"version"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("Run() exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), Version) {
		t.Fatalf("version output = %q", stdout.String())
	}
}

func TestValidate(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "configmap.yaml")
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  labels:
    driftguard.io/managed: "true"
data:
  mode: safe
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"validate", "--path", path}, &stdout, &stderr); code != exitOK {
		t.Fatalf("Run() exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid: 1 managed Kubernetes object(s)") {
		t.Fatalf("validate output = %q", stdout.String())
	}
}
