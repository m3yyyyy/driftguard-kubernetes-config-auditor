package audit

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fakeReader struct {
	objects []*unstructured.Unstructured
}

func (reader *fakeReader) ContextName() string { return "test-context" }

func (reader *fakeReader) EffectiveNamespace(item *unstructured.Unstructured, defaultNamespace string) (string, error) {
	if item.GetKind() == "Namespace" {
		return "", nil
	}
	if item.GetNamespace() != "" {
		return item.GetNamespace(), nil
	}
	return defaultNamespace, nil
}

func (reader *fakeReader) Get(_ context.Context, desired *unstructured.Unstructured, defaultNamespace string) (*unstructured.Unstructured, error) {
	namespace, _ := reader.EffectiveNamespace(desired, defaultNamespace)
	wanted := resourceID(desired, namespace)
	for _, item := range reader.objects {
		if resourceID(item, item.GetNamespace()) == wanted {
			return item.DeepCopy(), nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "objects"}, desired.GetName())
}

func (reader *fakeReader) ListManaged(_ context.Context, exemplar *unstructured.Unstructured, defaultNamespace string) ([]*unstructured.Unstructured, error) {
	namespace, _ := reader.EffectiveNamespace(exemplar, defaultNamespace)
	result := make([]*unstructured.Unstructured, 0)
	for _, item := range reader.objects {
		if item.GroupVersionKind() == exemplar.GroupVersionKind() && item.GetNamespace() == namespace {
			result = append(result, item.DeepCopy())
		}
	}
	return result, nil
}

func TestRunClassifiesInSyncModifiedMissingAndUnexpected(t *testing.T) {
	inSync := object("v1", "ConfigMap", "demo", "in-sync", map[string]any{"data": map[string]any{"mode": "safe"}})
	modified := object("v1", "ConfigMap", "demo", "modified", map[string]any{"data": map[string]any{"mode": "safe"}})
	missing := object("v1", "ConfigMap", "demo", "missing", map[string]any{"data": map[string]any{"mode": "safe"}})

	liveModified := modified.DeepCopy()
	liveModified.Object["data"].(map[string]any)["mode"] = "changed"
	unexpected := object("v1", "ConfigMap", "demo", "unexpected", map[string]any{"data": map[string]any{"mode": "safe"}})

	result := Run(context.Background(), []*unstructured.Unstructured{inSync, modified, missing}, &fakeReader{
		objects: []*unstructured.Unstructured{inSync.DeepCopy(), liveModified, unexpected},
	}, "default")

	if result.Summary.InSync != 1 || result.Summary.Modified != 1 || result.Summary.Missing != 1 || result.Summary.Unexpected != 1 || result.Summary.Errors != 0 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if !result.Summary.HasDrift() {
		t.Fatal("expected drift to be reported")
	}
}

func TestRunRejectsDuplicatesAfterNamespaceDefaulting(t *testing.T) {
	implicit := object("v1", "ConfigMap", "", "settings", map[string]any{})
	explicit := object("v1", "ConfigMap", "default", "settings", map[string]any{})

	result := Run(context.Background(), []*unstructured.Unstructured{implicit, explicit}, &fakeReader{}, "default")

	if result.Summary.Errors != 1 {
		t.Fatalf("expected one duplicate error, got %+v", result.Summary)
	}
}
