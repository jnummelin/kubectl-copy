package sanitize

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// testCase is a manifest and what Object should turn it into.
type testCase struct {
	name string
	in   string
	want string
}

func run(t *testing.T, tests []testCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := fromYAML(t, tt.in)
			original := in.DeepCopy()

			got := Object(in)

			if want := fromYAML(t, tt.want); !reflect.DeepEqual(got.Object, want.Object) {
				gotYAML, _ := yaml.Marshal(got.Object)
				t.Errorf("Object() mismatch\ngot:\n%s\nwant:\n%s", gotYAML, tt.want)
			}
			if !reflect.DeepEqual(in.Object, original.Object) {
				t.Error("Object() modified its input")
			}
		})
	}
}

func fromYAML(t *testing.T, s string) *unstructured.Unstructured {
	t.Helper()
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(s), &obj.Object); err != nil {
		t.Fatalf("bad test yaml: %v", err)
	}
	return obj
}

func TestCommon(t *testing.T) {
	run(t, []testCase{
		{
			name: "server owned state is removed",
			in: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  namespace: my-ns
  uid: 0a1b
  resourceVersion: "42"
  generation: 3
  creationTimestamp: "2026-01-01T00:00:00Z"
  deletionTimestamp: "2026-01-02T00:00:00Z"
  deletionGracePeriodSeconds: 30
  generateName: settings-
  finalizers: [example.com/protect]
  ownerReferences:
  - apiVersion: v1
    kind: Pod
    name: owner
    uid: ffff
  managedFields:
  - manager: kubectl
  labels:
    app: demo
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: "{}"
data:
  key: value
status:
  anything: goes
`,
			want: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  namespace: my-ns
  labels:
    app: demo
data:
  key: value
`,
		},
		{
			name: "user annotations are kept",
			in: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: a
  annotations:
    deployment.kubernetes.io/revision: "4"
    example.com/owner: team-a
`,
			want: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: a
  annotations:
    example.com/owner: team-a
`,
		},
		{
			name: "custom resources get the common rules",
			in: `
apiVersion: example.com/v1alpha1
kind: Widget
metadata:
  name: w
  uid: 0a1b
  finalizers: [example.com/cleanup]
spec:
  size: 3
status:
  ready: true
`,
			want: `
apiVersion: example.com/v1alpha1
kind: Widget
metadata:
  name: w
spec:
  size: 3
`,
		},
		{
			name: "kind rules are scoped to their API group",
			in: `
apiVersion: example.com/v1
kind: Service
metadata:
  name: not-a-core-service
spec:
  clusterIP: 10.0.0.10
`,
			want: `
apiVersion: example.com/v1
kind: Service
metadata:
  name: not-a-core-service
spec:
  clusterIP: 10.0.0.10
`,
		},
	})
}

func TestGroupRules(t *testing.T) {
	register := func(gk schema.GroupKind, fn Func) {
		rules[gk] = fn
		t.Cleanup(func() { delete(rules, gk) })
	}
	mark := func(by string) Func {
		return func(obj *unstructured.Unstructured) {
			order, _, _ := unstructured.NestedStringSlice(obj.Object, "order")
			_ = unstructured.SetNestedStringSlice(obj.Object, append(order, by), "order")
		}
	}
	register(schema.GroupKind{Group: "example.com"}, mark("group"))
	register(schema.GroupKind{Group: "example.com", Kind: "Widget"}, mark("kind"))

	run(t, []testCase{
		{
			name: "group rules run before kind rules",
			in:   "{apiVersion: example.com/v1, kind: Widget, metadata: {name: w}}",
			want: "{apiVersion: example.com/v1, kind: Widget, metadata: {name: w}, order: [group, kind]}",
		},
		{
			name: "group rules apply to every kind in the group",
			in:   "{apiVersion: example.com/v2, kind: Gadget, metadata: {name: g}}",
			want: "{apiVersion: example.com/v2, kind: Gadget, metadata: {name: g}, order: [group]}",
		},
		{
			name: "group rules do not apply to other groups",
			in:   "{apiVersion: other.example.com/v1, kind: Widget, metadata: {name: w}}",
			want: "{apiVersion: other.example.com/v1, kind: Widget, metadata: {name: w}}",
		},
		{
			name: "group rules do not apply to the core group",
			in:   "{apiVersion: v1, kind: ConfigMap, metadata: {name: c}}",
			want: "{apiVersion: v1, kind: ConfigMap, metadata: {name: c}}",
		},
	})
}
