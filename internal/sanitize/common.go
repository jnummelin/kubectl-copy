package sanitize

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// Metadata fields that are owned by the API server or describe the lifecycle
// of the original object.
var serverMetadataFields = []string{
	"uid",
	"resourceVersion",
	"generation",
	"creationTimestamp",
	"deletionTimestamp",
	"deletionGracePeriodSeconds",
	"selfLink",
	"managedFields",
	"ownerReferences",
	"finalizers",
	"generateName",
}

// Annotations that controllers and tooling write about the original object,
// whatever its kind.
var statefulAnnotations = []string{
	"kubectl.kubernetes.io/last-applied-configuration",
	"deployment.kubernetes.io/revision",
}

// common applies the rules that hold for every kind.
func common(obj *unstructured.Unstructured) {
	removeField(obj, "status")
	for _, f := range serverMetadataFields {
		removeField(obj, "metadata", f)
	}
	removeAnnotations(obj, statefulAnnotations...)
}
