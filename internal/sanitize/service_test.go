package sanitize

import "testing"

func TestService(t *testing.T) {
	run(t, []testCase{
		{
			name: "allocated addresses and ports are removed",
			in: `
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  type: NodePort
  clusterIP: 10.0.0.10
  clusterIPs: [10.0.0.10]
  healthCheckNodePort: 31000
  ports:
  - port: 80
    nodePort: 30080
`,
			want: `
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  type: NodePort
  ports:
  - port: 80
`,
		},
		{
			name: "headless service keeps clusterIP None",
			in: `
apiVersion: v1
kind: Service
metadata:
  name: db
spec:
  clusterIP: None
  clusterIPs: [None]
`,
			want: `
apiVersion: v1
kind: Service
metadata:
  name: db
spec:
  clusterIP: None
  clusterIPs: [None]
`,
		},
	})
}
