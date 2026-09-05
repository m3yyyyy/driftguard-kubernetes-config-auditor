package manifest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	ManagedLabel       = "driftguard.io/managed"
	maxManifestBytes   = 1 << 20
	maxManifestObjects = 100
)

// Load reads Kubernetes YAML or JSON objects from a file or directory.
func Load(root string) ([]*unstructured.Unstructured, error) {
	files, err := manifestFiles(root)
	if err != nil {
		return nil, err
	}

	objects := make([]*unstructured.Unstructured, 0)
	seen := make(map[string]string)
	for _, path := range files {
		decoded, decodeErr := decodeFile(path)
		if decodeErr != nil {
			return nil, decodeErr
		}
		for _, object := range decoded {
			if len(objects) >= maxManifestObjects {
				return nil, fmt.Errorf("manifest set exceeds the %d-object limit", maxManifestObjects)
			}
			if validationErr := validate(object); validationErr != nil {
				return nil, fmt.Errorf("%s: %w", path, validationErr)
			}
			key := identityKey(object)
			if previous, exists := seen[key]; exists {
				return nil, fmt.Errorf("duplicate object %s in %s and %s", key, previous, path)
			}
			seen[key] = path
			objects = append(objects, object)
		}
	}

	if len(objects) == 0 {
		return nil, errors.New("no Kubernetes objects found")
	}
	return objects, nil
}

func manifestFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect manifest path: %w", err)
	}
	if !info.IsDir() {
		if !supportedExtension(root) {
			return nil, fmt.Errorf("unsupported manifest extension: %s", filepath.Ext(root))
		}
		return []string{root}, nil
	}

	files := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if supportedExtension(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk manifest directory: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func supportedExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

func decodeFile(path string) ([]*unstructured.Unstructured, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.Size() > maxManifestBytes {
		return nil, fmt.Errorf("%s exceeds the 1 MiB file limit", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	decoder := utilyaml.NewYAMLOrJSONDecoder(file, 4096)
	objects := make([]*unstructured.Unstructured, 0)
	for document := 1; ; document++ {
		raw := make(map[string]any)
		if decodeErr := decoder.Decode(&raw); decodeErr != nil {
			if errors.Is(decodeErr, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode %s document %d: %w", path, document, decodeErr)
		}
		if len(raw) == 0 {
			continue
		}
		objects = append(objects, &unstructured.Unstructured{Object: raw})
	}
	return objects, nil
}

func validate(object *unstructured.Unstructured) error {
	if object.GetAPIVersion() == "" {
		return errors.New("apiVersion is required")
	}
	if object.GetKind() == "" {
		return errors.New("kind is required")
	}
	if object.GetName() == "" {
		return errors.New("metadata.name is required; generateName is not supported")
	}
	if strings.EqualFold(object.GetKind(), "List") {
		return errors.New("List objects are not supported; provide individual resources")
	}
	if strings.EqualFold(object.GetKind(), "Secret") {
		return errors.New("Secret objects are rejected to prevent sensitive-data comparison")
	}
	if object.GetLabels()[ManagedLabel] != "true" {
		return fmt.Errorf("metadata.labels[%q] must equal %q", ManagedLabel, "true")
	}
	return nil
}

func identityKey(object *unstructured.Unstructured) string {
	return strings.Join([]string{
		object.GetAPIVersion(),
		object.GetKind(),
		object.GetNamespace(),
		object.GetName(),
	}, "|")
}
