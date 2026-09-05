package kube

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/manifest"
)

type Client struct {
	contextName string
	dynamic     dynamic.Interface
	mapper      meta.RESTMapper
}

// New loads the standard kubeconfig and creates a read-only dynamic client.
func New(contextName string) (*Client, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	deferred := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)

	rawConfig, err := deferred.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	resolvedContext := contextName
	if resolvedContext == "" {
		resolvedContext = rawConfig.CurrentContext
	}
	if resolvedContext == "" {
		return nil, fmt.Errorf("kubeconfig has no current context")
	}

	restConfig, err := deferred.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build Kubernetes client configuration: %w", err)
	}
	restConfig.UserAgent = "driftguard"

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create dynamic Kubernetes client: %w", err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes discovery client: %w", err)
	}

	return &Client{
		contextName: resolvedContext,
		dynamic:     dynamicClient,
		mapper:      restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient)),
	}, nil
}

func (client *Client) ContextName() string {
	return client.contextName
}

func (client *Client) EffectiveNamespace(object *unstructured.Unstructured, defaultNamespace string) (string, error) {
	mapping, err := client.mapping(object.GroupVersionKind())
	if err != nil {
		return "", err
	}
	if mapping.Scope.Name() != meta.RESTScopeNameNamespace {
		return "", nil
	}
	if object.GetNamespace() != "" {
		return object.GetNamespace(), nil
	}
	if defaultNamespace == "" {
		return "default", nil
	}
	return defaultNamespace, nil
}

func (client *Client) Get(ctx context.Context, object *unstructured.Unstructured, defaultNamespace string) (*unstructured.Unstructured, error) {
	resource, _, err := client.resource(object, defaultNamespace)
	if err != nil {
		return nil, err
	}
	live, err := resource.Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return live, nil
}

func (client *Client) ListManaged(ctx context.Context, exemplar *unstructured.Unstructured, defaultNamespace string) ([]*unstructured.Unstructured, error) {
	resource, _, err := client.resource(exemplar, defaultNamespace)
	if err != nil {
		return nil, err
	}
	list, err := resource.List(ctx, metav1.ListOptions{LabelSelector: manifest.ManagedLabel + "=true"})
	if err != nil {
		return nil, err
	}
	objects := make([]*unstructured.Unstructured, 0, len(list.Items))
	for index := range list.Items {
		objects = append(objects, list.Items[index].DeepCopy())
	}
	return objects, nil
}

func (client *Client) resource(object *unstructured.Unstructured, defaultNamespace string) (dynamic.ResourceInterface, string, error) {
	mapping, err := client.mapping(object.GroupVersionKind())
	if err != nil {
		return nil, "", err
	}
	namespace := ""
	resource := client.dynamic.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespace = object.GetNamespace()
		if namespace == "" {
			namespace = defaultNamespace
		}
		if namespace == "" {
			namespace = "default"
		}
		return resource.Namespace(namespace), namespace, nil
	}
	return resource, namespace, nil
}

func (client *Client) mapping(gvk schema.GroupVersionKind) (*meta.RESTMapping, error) {
	mapping, err := client.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("map %s: %w", gvk.String(), err)
	}
	return mapping, nil
}
