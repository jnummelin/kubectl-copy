# kubectl-copy

A kubectl plugin that copies a resource under a new name.

```
kubectl copy pod/my-pod -n my-ns
```

It reads the resource, strips the state that belongs to the original, opens the
result in your editor (like `kubectl edit`) and creates it when the editor exits.

## Install

```
go build -o kubectl-copy .
mv kubectl-copy /usr/local/bin/   # anywhere on your PATH
```

## Usage

```
kubectl copy (TYPE/NAME | TYPE NAME) [NEW_NAME] [--no-edit] [kubectl flags]
```

- The copy is named `NEW_NAME`, or `<name>-copy` by default. You can change the
  name, the namespace or anything else in the editor.
- Saving an empty file cancels the copy.
- If the API server rejects the copy, the editor reopens with the error on top.
  Exiting without changes then gives up and leaves your manifest in a temp file.
- `--no-edit` creates the copy straight away.
- The editor comes from `KUBE_EDITOR` or `EDITOR`, falling back to `vi`.
- The copy is created, never applied, so an existing object is never overwritten.

## What is removed

From every resource: `status`, and from `metadata`: `uid`, `resourceVersion`,
`generation`, `creationTimestamp`, `deletionTimestamp`,
`deletionGracePeriodSeconds`, `selfLink`, `managedFields`, `ownerReferences`,
`finalizers`, `generateName`, plus the `last-applied-configuration` and
`deployment.kubernetes.io/revision` annotations.

Kind specific:

| Kind | Removed |
| --- | --- |
| Pod | `spec.nodeName`, `spec.ephemeralContainers`, the injected `kube-api-access-*` volume and its mounts |
| Service | `clusterIP`/`clusterIPs` (kept when `None`), `nodePort`s, `healthCheckNodePort` |
| PersistentVolumeClaim | `spec.volumeName`, binding and provisioner annotations |
| PersistentVolume | `spec.claimRef` |
| Job | generated `spec.selector` and `controller-uid`/`job-name` labels (kept with `manualSelector: true`) |

Copying a pod that is controlled by a ReplicaSet, Job, etc. keeps its labels, so
the controller may adopt or delete the copy. The editor shows a note when that
applies; change the labels there.

## Adding rules

Sanitizing lives in [internal/sanitize](internal/sanitize). Common rules for
every object are in `common.go`; other rules each have their own file. To add
one, custom resources included:

1. Write a `Func` in a new file, using the helpers in `helpers.go`.
2. Register it in the `rules` map in `sanitize.go`, by group and kind. Leave the
   kind out to cover every kind in the group.
3. Add a test next to it: a manifest in, the expected manifest out.

Rules run in order: common, then group, then kind.

## License

Apache License 2.0, see [LICENSE](LICENSE).
