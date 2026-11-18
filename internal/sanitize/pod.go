package sanitize

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func pod(obj *unstructured.Unstructured) {
	removeField(obj, "spec", "nodeName")
	// Ephemeral containers cannot be set on create.
	removeField(obj, "spec", "ephemeralContainers")

	// The service account admission plugin injects a kube-api-access-<hash>
	// volume; drop it so the copy gets its own.
	keepUnlessInjected := func(item map[string]any) bool {
		name, _ := item["name"].(string)
		return !strings.HasPrefix(name, "kube-api-access-")
	}
	editItems(obj.Object, keepUnlessInjected, "spec", "volumes")
	for _, field := range []string{"containers", "initContainers"} {
		editItems(obj.Object, func(container map[string]any) bool {
			editItems(container, keepUnlessInjected, "volumeMounts")
			return true
		}, "spec", field)
	}
}
