package sanitize

import "testing"

func TestPersistentVolumeClaim(t *testing.T) {
	run(t, []testCase{{
		name: "binding state is removed",
		in: `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data
  annotations:
    pv.kubernetes.io/bind-completed: "yes"
    pv.kubernetes.io/bound-by-controller: "yes"
    volume.kubernetes.io/selected-node: node-1
    volume.kubernetes.io/storage-provisioner: example.com/csi
spec:
  volumeName: pvc-1234
  storageClassName: standard
`,
		want: `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data
spec:
  storageClassName: standard
`,
	}})
}

func TestPersistentVolume(t *testing.T) {
	run(t, []testCase{{
		name: "claim reference is removed",
		in: `
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-1
spec:
  storageClassName: standard
  claimRef:
    kind: PersistentVolumeClaim
    name: data
    namespace: my-ns
`,
		want: `
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-1
spec:
  storageClassName: standard
`,
	}})
}
