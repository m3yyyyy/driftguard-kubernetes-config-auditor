package audit

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/manifest"
)

// Reader exposes only the read operations required by an audit.
type Reader interface {
	ContextName() string
	EffectiveNamespace(*unstructured.Unstructured, string) (string, error)
	Get(context.Context, *unstructured.Unstructured, string) (*unstructured.Unstructured, error)
	ListManaged(context.Context, *unstructured.Unstructured, string) ([]*unstructured.Unstructured, error)
}

// Run compares desired objects with their live Kubernetes counterparts.
func Run(ctx context.Context, desired []*unstructured.Unstructured, reader Reader, defaultNamespace string) Report {
	report := Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Context:       reader.ContextName(),
		Findings:      make([]Finding, 0, len(desired)),
	}

	expected := make(map[string]struct{}, len(desired))
	type scope struct {
		key       string
		exemplar  *unstructured.Unstructured
		namespace string
	}
	scopes := make(map[string]scope)

	for _, object := range desired {
		namespace, err := reader.EffectiveNamespace(object, defaultNamespace)
		if err != nil {
			report.add(Finding{
				Resource: fallbackResourceID(object),
				Status:   StatusError,
				Message:  "resolve Kubernetes resource mapping: " + err.Error(),
			})
			continue
		}

		resource := resourceID(object, namespace)
		if _, duplicate := expected[resource]; duplicate {
			report.add(Finding{
				Resource: resource,
				Status:   StatusError,
				Message:  "multiple desired manifests resolve to the same live resource",
			})
			continue
		}
		expected[resource] = struct{}{}
		scopeKey := object.GroupVersionKind().String() + "|" + namespace
		scopes[scopeKey] = scope{key: scopeKey, exemplar: object, namespace: namespace}

		live, getErr := reader.Get(ctx, object, defaultNamespace)
		switch {
		case apierrors.IsNotFound(getErr):
			report.add(Finding{Resource: resource, Status: StatusMissing, Message: "desired resource does not exist in the cluster"})
		case getErr != nil:
			report.add(Finding{Resource: resource, Status: StatusError, Message: "read live resource: " + getErr.Error()})
		default:
			paths := DiffPaths(object, live)
			if len(paths) == 0 {
				report.add(Finding{Resource: resource, Status: StatusInSync, Message: "desired fields match the live resource"})
			} else {
				report.add(Finding{Resource: resource, Status: StatusModified, Paths: paths, Message: "one or more desired fields differ"})
			}
		}
	}

	orderedScopes := make([]scope, 0, len(scopes))
	for _, item := range scopes {
		orderedScopes = append(orderedScopes, item)
	}
	sort.Slice(orderedScopes, func(i, j int) bool { return orderedScopes[i].key < orderedScopes[j].key })

	for _, item := range orderedScopes {
		liveObjects, err := reader.ListManaged(ctx, item.exemplar, defaultNamespace)
		if err != nil {
			report.add(Finding{
				Resource: resourceScopeID(item.exemplar, item.namespace),
				Status:   StatusError,
				Message:  "list managed resources: " + err.Error(),
			})
			continue
		}
		for _, live := range liveObjects {
			resource := resourceID(live, live.GetNamespace())
			if _, exists := expected[resource]; !exists {
				report.add(Finding{
					Resource: resource,
					Status:   StatusUnexpected,
					Message:  fmt.Sprintf("live resource has %s=true but no desired manifest", manifest.ManagedLabel),
				})
			}
		}
	}

	sort.SliceStable(report.Findings, func(i, j int) bool {
		if report.Findings[i].Resource == report.Findings[j].Resource {
			return report.Findings[i].Status < report.Findings[j].Status
		}
		return report.Findings[i].Resource < report.Findings[j].Resource
	})
	return report
}

func (report *Report) add(finding Finding) {
	report.Findings = append(report.Findings, finding)
	report.Summary.Total++
	switch finding.Status {
	case StatusInSync:
		report.Summary.InSync++
	case StatusModified:
		report.Summary.Modified++
	case StatusMissing:
		report.Summary.Missing++
	case StatusUnexpected:
		report.Summary.Unexpected++
	case StatusError:
		report.Summary.Errors++
	}
}

func resourceID(object *unstructured.Unstructured, namespace string) string {
	location := object.GetName()
	if namespace != "" {
		location = namespace + "/" + location
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s %s", object.GetAPIVersion(), object.GetKind(), location))
}

func fallbackResourceID(object *unstructured.Unstructured) string {
	return resourceID(object, object.GetNamespace())
}

func resourceScopeID(object *unstructured.Unstructured, namespace string) string {
	location := "cluster scope"
	if namespace != "" {
		location = "namespace " + namespace
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s (%s)", object.GetAPIVersion(), object.GetKind(), location))
}
