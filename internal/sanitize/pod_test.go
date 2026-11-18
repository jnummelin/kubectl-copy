package sanitize

import "testing"

func TestPod(t *testing.T) {
	run(t, []testCase{{
		name: "scheduling and injected state is removed",
		in: `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  nodeName: node-1
  ephemeralContainers:
  - name: debugger
  initContainers:
  - name: init
    volumeMounts:
    - name: kube-api-access-abcde
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
  containers:
  - name: app
    volumeMounts:
    - name: data
      mountPath: /data
    - name: kube-api-access-abcde
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
  volumes:
  - name: data
    emptyDir: {}
  - name: kube-api-access-abcde
    projected: {}
`,
		want: `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  initContainers:
  - name: init
  containers:
  - name: app
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    emptyDir: {}
`,
	}})
}
