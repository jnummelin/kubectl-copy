package sanitize

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// Labels the Job controller generates; the API server rejects stale values.
var jobGeneratedLabels = []string{
	"controller-uid",
	"job-name",
	"batch.kubernetes.io/controller-uid",
	"batch.kubernetes.io/job-name",
}

func job(obj *unstructured.Unstructured) {
	// With manualSelector the selector and labels are the user's own.
	if manual, _, _ := unstructured.NestedBool(obj.Object, "spec", "manualSelector"); manual {
		return
	}
	removeField(obj, "spec", "selector")
	removeKeys(obj, []string{"metadata", "labels"}, jobGeneratedLabels...)
	removeKeys(obj, []string{"spec", "template", "metadata", "labels"}, jobGeneratedLabels...)
}
