package audit

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDiffPathsIgnoresLiveDefaultedAndRuntimeFields(t *testing.T) {
	desired := object("v1", "ConfigMap", "demo", "settings", map[string]any{
		"data": map[string]any{"mode": "safe"},
	})
	live := desired.DeepCopy()
	live.SetResourceVersion("42")
	live.Object["status"] = map[string]any{"ignored": true}
	live.Object["extra-controller-field"] = "ignored"

	if paths := DiffPaths(desired, live); len(paths) != 0 {
		t.Fatalf("expected no drift, got %v", paths)
	}
}

func TestDiffPathsReportsPathsWithoutValues(t *testing.T) {
	desired := object("v1", "ConfigMap", "demo", "settings", map[string]any{
		"data": map[string]any{"mode": "safe", "team": "platform"},
	})
	live := desired.DeepCopy()
	live.Object["data"].(map[string]any)["mode"] = "unsafe"
	delete(live.Object["data"].(map[string]any), "team")

	want := []string{"/data/mode", "/data/team"}
	if got := DiffPaths(desired, live); !reflect.DeepEqual(got, want) {
		t.Fatalf("DiffPaths() = %v, want %v", got, want)
	}
}

func TestDiffPathsTreatsEquivalentNumericTypesAsEqual(t *testing.T) {
	desired := object("apps/v1", "Deployment", "demo", "web", map[string]any{
		"spec": map[string]any{"replicas": float64(2)},
	})
	live := desired.DeepCopy()
	live.Object["spec"].(map[string]any)["replicas"] = int64(2)

	if paths := DiffPaths(desired, live); len(paths) != 0 {
		t.Fatalf("expected equivalent numeric values to match, got %v", paths)
	}
}

func object(apiVersion, kind, namespace, name string, fields map[string]any) *unstructured.Unstructured {
	raw := map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]any{
				"driftguard.io/managed": "true",
			},
		},
	}
	for key, value := range fields {
		raw[key] = value
	}
	return &unstructured.Unstructured{Object: raw}
}
