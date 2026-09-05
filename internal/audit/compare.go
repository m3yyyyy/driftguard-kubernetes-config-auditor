package audit

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const maxReportedPaths = 100

// DiffPaths compares only fields declared by the desired object. Kubernetes
// defaulted and controller-owned fields that appear only in the live object do
// not create false drift.
func DiffPaths(desired, live *unstructured.Unstructured) []string {
	desiredObject := desired.DeepCopy().Object
	removeRuntimeFields(desiredObject)

	paths := make([]string, 0)
	walkDesired(desiredObject, live.Object, "", &paths)
	if len(paths) > maxReportedPaths {
		paths = append(paths[:maxReportedPaths], "/...truncated")
	}
	return paths
}

func removeRuntimeFields(object map[string]any) {
	delete(object, "status")
	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return
	}
	for _, key := range []string{
		"creationTimestamp",
		"deletionGracePeriodSeconds",
		"deletionTimestamp",
		"generation",
		"managedFields",
		"resourceVersion",
		"selfLink",
		"uid",
	} {
		delete(metadata, key)
	}
	annotations, ok := metadata["annotations"].(map[string]any)
	if !ok {
		return
	}
	delete(annotations, "deployment.kubernetes.io/revision")
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
}

func walkDesired(desired, live any, path string, paths *[]string) {
	if len(*paths) > maxReportedPaths {
		return
	}

	switch desiredValue := desired.(type) {
	case map[string]any:
		liveValue, ok := live.(map[string]any)
		if !ok {
			appendPath(paths, path)
			return
		}
		keys := make([]string, 0, len(desiredValue))
		for key := range desiredValue {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := path + "/" + escapeJSONPointer(key)
			liveChild, exists := liveValue[key]
			if !exists {
				appendPath(paths, childPath)
				continue
			}
			walkDesired(desiredValue[key], liveChild, childPath, paths)
		}
	case []any:
		liveValue, ok := live.([]any)
		if !ok || len(desiredValue) != len(liveValue) {
			appendPath(paths, path)
			return
		}
		for index := range desiredValue {
			walkDesired(desiredValue[index], liveValue[index], fmt.Sprintf("%s/%d", path, index), paths)
		}
	default:
		if !reflect.DeepEqual(desired, live) && !numbersEqual(desired, live) {
			appendPath(paths, path)
		}
	}
}

func numbersEqual(left, right any) bool {
	leftNumber, leftOK := asRational(left)
	rightNumber, rightOK := asRational(right)
	return leftOK && rightOK && leftNumber.Cmp(rightNumber) == 0
}

func asRational(value any) (*big.Rat, bool) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return nil, false
	}
	number := new(big.Rat)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number.SetInt64(reflected.Int())
		return number, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		integer := new(big.Int).SetUint64(reflected.Uint())
		number.SetInt(integer)
		return number, true
	case reflect.Float32, reflect.Float64:
		if number.SetFloat64(reflected.Float()) == nil {
			return nil, false
		}
		return number, true
	default:
		return nil, false
	}
}

func appendPath(paths *[]string, path string) {
	if path == "" {
		path = "/"
	}
	*paths = append(*paths, path)
}

func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}
