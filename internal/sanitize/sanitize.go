// Package sanitize removes from an object everything that belongs to the
// original, so that what is left can be created as a new object.
//
// Sanitizing runs the common rules that apply to every object (see common.go),
// then the rules registered for the object's API group, then the rules
// registered for its kind.
//
// To add rules, write a Func in its own file and register it in rules below.
// Custom resources work the same way as built-in kinds.
package sanitize

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Func removes state from obj in place.
type Func func(obj *unstructured.Unstructured)

// rules holds the rules that apply to only some objects. They are keyed by
// group and kind, so they apply to every version of the API. An entry without
// a Kind applies to every kind in its group:
//
//	{Group: "example.com"}:                 every kind in example.com
//	{Group: "example.com", Kind: "Widget"}: Widgets only
//	{Kind: "Pod"}:                          Pods, in the core group
var rules = map[schema.GroupKind]Func{
	{Kind: "Pod"}:                   pod,
	{Kind: "Service"}:               service,
	{Kind: "PersistentVolumeClaim"}: persistentVolumeClaim,
	{Kind: "PersistentVolume"}:      persistentVolume,
	{Group: "batch", Kind: "Job"}:   job,
}

// Object returns a sanitized copy of obj. obj itself is not modified.
func Object(obj *unstructured.Unstructured) *unstructured.Unstructured {
	out := obj.DeepCopy()
	common(out)

	gk := out.GroupVersionKind().GroupKind()
	if group, ok := rules[schema.GroupKind{Group: gk.Group}]; ok {
		group(out)
	}
	if kind, ok := rules[gk]; ok && gk.Kind != "" {
		kind(out)
	}
	return out
}
