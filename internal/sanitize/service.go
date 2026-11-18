package sanitize

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func service(obj *unstructured.Unstructured) {
	// Keep "None": that is the user's intent (headless), not allocated state.
	if ip, _, _ := unstructured.NestedString(obj.Object, "spec", "clusterIP"); ip != "None" {
		removeField(obj, "spec", "clusterIP")
		removeField(obj, "spec", "clusterIPs")
	}
	removeField(obj, "spec", "healthCheckNodePort")
	editItems(obj.Object, func(port map[string]any) bool {
		delete(port, "nodePort")
		return true
	}, "spec", "ports")
}
