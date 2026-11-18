package sanitize

import "testing"

func TestJob(t *testing.T) {
	run(t, []testCase{
		{
			name: "generated selector and labels are removed",
			in: `
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
  labels:
    batch.kubernetes.io/controller-uid: abcd
    batch.kubernetes.io/job-name: migrate
    controller-uid: abcd
    job-name: migrate
spec:
  selector:
    matchLabels:
      batch.kubernetes.io/controller-uid: abcd
  template:
    metadata:
      labels:
        app: migrate
        batch.kubernetes.io/controller-uid: abcd
        batch.kubernetes.io/job-name: migrate
        controller-uid: abcd
        job-name: migrate
`,
			want: `
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
spec:
  template:
    metadata:
      labels:
        app: migrate
`,
		},
		{
			name: "manual selector is left alone",
			in: `
apiVersion: batch/v1
kind: Job
metadata:
  name: a
spec:
  manualSelector: true
  selector:
    matchLabels:
      job-name: mine
`,
			want: `
apiVersion: batch/v1
kind: Job
metadata:
  name: a
spec:
  manualSelector: true
  selector:
    matchLabels:
      job-name: mine
`,
		},
	})
}
