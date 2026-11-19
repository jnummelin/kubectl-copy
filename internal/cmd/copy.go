package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jnummelin/kubectl-copy/internal/sanitize"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"sigs.k8s.io/yaml"
)

const fieldManager = "kubectl-copy"

const (
	copyLong = `Copy a resource under a new name.

Reads the resource, strips everything that belongs to the original object
(status, uid, resourceVersion, finalizers, ownerReferences, managedFields and
kind specific allocated state such as a Service's clusterIP), opens the result
in your editor and creates it when the editor exits.

The copy is named NEW_NAME, or <name>-copy if NEW_NAME is not given; you can
also change the name, the namespace or anything else in the editor. Saving an
empty file cancels the copy.

The editor is taken from KUBE_EDITOR or EDITOR, falling back to vi.`

	copyExample = `  # Copy a pod, editing the copy before it is created
  kubectl copy pod/my-pod -n my-ns

  # Copy a deployment to a given name
  kubectl copy deployment/web web-canary

  # Copy without opening the editor
  kubectl copy configmap/settings settings-backup --no-edit`
)

type copyOptions struct {
	genericiooptions.IOStreams
	configFlags *genericclioptions.ConfigFlags
	noEdit      bool
}

// NewCopyCommand returns the kubectl-copy root command.
func NewCopyCommand(streams genericiooptions.IOStreams) *cobra.Command {
	o := &copyOptions{
		IOStreams:   streams,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
	cmd := &cobra.Command{
		Use:          "kubectl copy TYPE/NAME [NEW_NAME]",
		Short:        "Copy a resource under a new name",
		Long:         copyLong,
		Example:      copyExample,
		Args:         cobra.RangeArgs(1, 2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd.Context(), args)
		},
	}
	cmd.SetIn(streams.In)
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)
	cmd.Flags().BoolVar(&o.noEdit, "no-edit", false, "Create the copy without opening an editor")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

// parseArgs returns the TYPE/NAME of the resource to copy and the optional
// new name.
func parseArgs(args []string) (source, newName string, err error) {
	kind, name, found := strings.Cut(args[0], "/")
	if !found || kind == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("the resource to copy must be given as TYPE/NAME, for example pod/my-pod; got %q", args[0])
	}
	if len(args) > 1 {
		newName = args[1]
	}
	return args[0], newName, nil
}

func (o *copyOptions) run(ctx context.Context, args []string) error {
	sourceArg, newName, err := parseArgs(args)
	if err != nil {
		return err
	}
	namespace, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	infos, err := resource.NewBuilder(o.configFlags).
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(false, sourceArg).
		SingleResourceType().
		Flatten().
		Do().
		Infos()
	if err != nil {
		return err
	}
	if len(infos) != 1 {
		return fmt.Errorf("expected exactly one resource to copy, got %d", len(infos))
	}
	source, ok := infos[0].Object.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected object type %T", infos[0].Object)
	}
	if newName == "" {
		newName = source.GetName() + "-copy"
	}

	sourceRef := refString(infos[0].Mapping, source)
	create := func(obj *unstructured.Unstructured) error {
		created, mapping, err := o.create(ctx, obj, namespace)
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Out, "%s created (copy of %s)\n", refString(mapping, created), sourceRef)
		return nil
	}

	copied := sanitize.Object(source)
	copied.SetName(newName)
	if o.noEdit {
		return create(copied)
	}

	manifest, err := yaml.Marshal(copied.Object)
	if err != nil {
		return err
	}
	edit := editor.NewDefaultEditor([]string{"KUBE_EDITOR", "EDITOR"})
	launch := func(content []byte) ([]byte, string, error) {
		return edit.LaunchTempFile("kubectl-copy-", ".yaml", bytes.NewReader(content))
	}
	cancelled, err := editAndCreate(header(sourceRef, source), manifest, launch, create)
	if cancelled {
		fmt.Fprintln(o.ErrOut, "Copy cancelled.")
	}
	return err
}

func (o *copyOptions) create(ctx context.Context, obj *unstructured.Unstructured, defaultNamespace string) (*unstructured.Unstructured, *meta.RESTMapping, error) {
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" {
		return nil, nil, errors.New("manifest must set apiVersion and kind")
	}
	mapper, err := o.configFlags.ToRESTMapper()
	if err != nil {
		return nil, nil, err
	}
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, nil, err
	}
	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	var ri dynamic.ResourceInterface = client.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(defaultNamespace)
		}
		ri = client.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	}
	// Create rather than apply: a copy must never overwrite an existing object.
	created, err := ri.Create(ctx, obj, metav1.CreateOptions{FieldManager: fieldManager})
	return created, mapping, err
}

func refString(mapping *meta.RESTMapping, obj *unstructured.Unstructured) string {
	// Same form as kubectl prints: pod/name, deployment.apps/name.
	kind := strings.ToLower(obj.GetKind())
	if mapping != nil && mapping.Resource.Group != "" {
		kind += "." + mapping.Resource.Group
	}
	if ns := obj.GetNamespace(); ns != "" {
		return fmt.Sprintf("%s/%s in namespace %s", kind, obj.GetName(), ns)
	}
	return kind + "/" + obj.GetName()
}
