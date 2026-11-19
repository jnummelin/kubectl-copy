package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"sigs.k8s.io/yaml"
)

// edit is the outcome of one editor session.
type edit struct {
	content []byte
	// saved reports whether the editor wrote the file at all. Quitting without
	// saving (":q!" in vi) leaves it false.
	saved bool
	file  string
}

// launchEditor opens content in the user's editor, from KUBE_EDITOR or EDITOR
// like kubectl edit, and waits for it to exit.
func launchEditor(content []byte) (edit, error) {
	f, err := os.CreateTemp("", "kubectl-copy-*.yaml")
	if err != nil {
		return edit{}, err
	}
	file := f.Name()
	_, err = f.Write(content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		// Backdate the file, so that a save always moves its modification
		// time, whatever the timestamp resolution of the filesystem.
		past := time.Now().Add(-time.Hour)
		err = os.Chtimes(file, past, past)
	}
	var before os.FileInfo
	if err == nil {
		before, err = os.Stat(file)
	}
	if err == nil {
		err = editor.NewDefaultEditor([]string{"KUBE_EDITOR", "EDITOR"}).Launch(file)
	}
	var after os.FileInfo
	if err == nil {
		after, err = os.Stat(file)
	}
	var edited []byte
	if err == nil {
		edited, err = os.ReadFile(file)
	}
	if err != nil {
		os.Remove(file)
		return edit{}, err
	}
	saved := !after.ModTime().Equal(before.ModTime()) || !bytes.Equal(edited, content)
	return edit{content: edited, saved: saved, file: file}, nil
}

// editAndCreate runs the edit loop: open the manifest in the editor, create the
// result, and on failure reopen the editor with the error shown on top, like
// kubectl edit does. It reports cancelled when the user quit the editor without
// saving, or saved an empty file.
func editAndCreate(
	intro string,
	manifest []byte,
	launch func([]byte) (edit, error),
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

		ed, err := launch(content)
		if err != nil {
			return false, err
		}
		body := stripHeader(ed.content)
		if len(bytes.TrimSpace(body)) == 0 {
			os.Remove(ed.file)
			return true, nil
		}
		if lastErr != nil && bytes.Equal(body, manifest) {
			return false, fmt.Errorf("%w\nNo changes were made after the error; your manifest is saved in %s", lastErr, ed.file)
		}
		if !ed.saved {
			os.Remove(ed.file)
			return true, nil
		}

		obj := &unstructured.Unstructured{}
		lastErr = yaml.Unmarshal(body, &obj.Object)
		if lastErr == nil {
			lastErr = create(obj)
		}
		os.Remove(ed.file)
		if lastErr == nil {
			return false, nil
		}
		manifest = body
	}
}

func header(sourceRef string, source *unstructured.Unstructured) string {
	var b strings.Builder
	b.WriteString("Copy of " + sourceRef + ". Edit the object below; it is created when you\n")
	b.WriteString("save and exit. Lines beginning with '#' are ignored. To cancel the copy,\n")
	b.WriteString("exit without saving or save an empty file.\n")
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
