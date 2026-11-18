package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// editAndCreate runs the edit loop: open the manifest in the editor, create the
// result, and on failure reopen the editor with the error shown on top, like
// kubectl edit does. It reports cancelled when the user saved an empty file.
func editAndCreate(
	intro string,
	manifest []byte,
	launch func([]byte) (edited []byte, file string, err error),
	create func(*unstructured.Unstructured) error,
) (cancelled bool, err error) {
	var lastErr error
	for {
		content := []byte(intro)
		if lastErr != nil {
			content = append(content, commented("\nThe copy was not created:\n"+lastErr.Error())...)
		}
		content = append(content, '#', '\n')
		content = append(content, manifest...)

		edited, file, err := launch(content)
		if err != nil {
			return false, err
		}
		body := stripHeader(edited)
		if len(bytes.TrimSpace(body)) == 0 {
			os.Remove(file)
			return true, nil
		}
		if lastErr != nil && bytes.Equal(body, manifest) {
			return false, fmt.Errorf("%w\nNo changes were made after the error; your manifest is saved in %s", lastErr, file)
		}

		obj := &unstructured.Unstructured{}
		lastErr = yaml.Unmarshal(body, &obj.Object)
		if lastErr == nil {
			lastErr = create(obj)
		}
		os.Remove(file)
		if lastErr == nil {
			return false, nil
		}
		manifest = body
	}
}

func header(sourceRef string, source *unstructured.Unstructured) string {
	var b strings.Builder
	b.WriteString("Copy of " + sourceRef + ". Edit the object below; it is created when you\n")
	b.WriteString("save and exit. Lines beginning with '#' are ignored, and an empty file\n")
	b.WriteString("cancels the copy.\n")
	for _, ref := range source.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			fmt.Fprintf(&b, "\nNOTE: the original is controlled by %s/%s. If the copy keeps labels\n", ref.Kind, ref.Name)
			b.WriteString("matching that controller's selector, the controller may adopt or delete it.\n")
		}
	}
	return commented(b.String())
}

// commented prefixes every line of s with "# ".
func commented(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b.WriteString(strings.TrimRight("# "+line, " ") + "\n")
	}
	return b.String()
}

// stripHeader drops the leading comment lines that kubectl-copy put on top of
// the manifest.
func stripHeader(content []byte) []byte {
	for len(content) > 0 && content[0] == '#' {
		i := bytes.IndexByte(content, '\n')
		if i < 0 {
			return nil
		}
		content = content[i+1:]
	}
	return content
}
