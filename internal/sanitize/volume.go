package sanitize

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func persistentVolumeClaim(obj *unstructured.Unstructured) {
	removeField(obj, "spec", "volumeName")
	removeAnnotations(obj,
		"pv.kubernetes.io/bind-completed",
		"pv.kubernetes.io/bound-by-controller",
		"volume.kubernetes.io/selected-node",
		"volume.kubernetes.io/storage-provisioner",
		"volume.beta.kubernetes.io/storage-provisioner",
	)
}

func persistentVolume(obj *unstructured.Unstructured) {
	removeField(obj, "spec", "claimRef")
}
