package sanitize

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// Helpers for writing Funcs. All of them are no-ops when the field is missing.

// removeField removes the field at path.
func removeField(obj *unstructured.Unstructured, path ...string) {
	unstructured.RemoveNestedField(obj.Object, path...)
}

// removeKeys removes keys from the map at path, and the map itself if that
// leaves it empty.
func removeKeys(obj *unstructured.Unstructured, path []string, keys ...string) {
	m, found, _ := unstructured.NestedMap(obj.Object, path...)
	if !found {
		return
	}
	for _, k := range keys {
		delete(m, k)
	}
	if len(m) == 0 {
		removeField(obj, path...)
		return
	}
	_ = unstructured.SetNestedMap(obj.Object, m, path...)
}

// removeAnnotations removes the given metadata.annotations.
func removeAnnotations(obj *unstructured.Unstructured, keys ...string) {
	removeKeys(obj, []string{"metadata", "annotations"}, keys...)
}

// editItems calls edit on every map item of the list at path. Items for which
// edit returns false are removed, and so is the list if that leaves it empty.
func editItems(content map[string]any, edit func(item map[string]any) (keep bool), path ...string) {
	items, found, _ := unstructured.NestedSlice(content, path...)
	if !found {
		return
	}
	kept := make([]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok && !edit(m) {
			continue
		}
		kept = append(kept, item)
	}
	if len(kept) == 0 {
		unstructured.RemoveNestedField(content, path...)
		return
	}
	_ = unstructured.SetNestedSlice(content, kept, path...)
}
