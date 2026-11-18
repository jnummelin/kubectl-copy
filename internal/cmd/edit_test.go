package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestStripHeader(t *testing.T) {
	tests := map[string]string{
		"# a\n#\nkind: Pod\n# kept\n": "kind: Pod\n# kept\n",
		"kind: Pod\n":                 "kind: Pod\n",
		"# only a comment":            "",
		"":                            "",
	}
	for in, want := range tests {
		if got := string(stripHeader([]byte(in))); got != want {
			t.Errorf("stripHeader(%q) = %q, want %q", in, got, want)
		}
	}
}

// fakeEditor replays one edit per launch and records what it was shown.
type fakeEditor struct {
	t     *testing.T
	edits []func(shown string) string
	shown []string
	files []string
}

func (f *fakeEditor) launch(content []byte) ([]byte, string, error) {
	if len(f.shown) >= len(f.edits) {
		f.t.Fatalf("editor launched more than %d times", len(f.edits))
	}
	edit := f.edits[len(f.shown)]
	f.shown = append(f.shown, string(content))
	file := filepath.Join(f.t.TempDir(), "edit.yaml")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		f.t.Fatal(err)
	}
	f.files = append(f.files, file)
	return []byte(edit(string(content))), file, nil
}

func unchanged(s string) string { return s }

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const testManifest = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a-copy\n"

func TestEditAndCreate(t *testing.T) {
	t.Run("creates the edited object", func(t *testing.T) {
		ed := &fakeEditor{t: t, edits: []func(string) string{
			func(s string) string { return strings.Replace(s, "a-copy", "renamed", 1) },
		}}
		var created []*unstructured.Unstructured
		cancelled, err := editAndCreate("# intro\n", []byte(testManifest), ed.launch, func(o *unstructured.Unstructured) error {
			created = append(created, o)
			return nil
		})
		if err != nil || cancelled {
			t.Fatalf("cancelled = %v, err = %v", cancelled, err)
		}
		if len(created) != 1 || created[0].GetName() != "renamed" || created[0].GetKind() != "ConfigMap" {
			t.Errorf("unexpected created objects: %v", created)
		}
		if !strings.HasPrefix(ed.shown[0], "# intro\n#\napiVersion: v1\n") {
			t.Errorf("unexpected editor content:\n%s", ed.shown[0])
		}
		if exists(ed.files[0]) {
			t.Error("temp file was not removed")
		}
	})

	t.Run("unchanged manifest is still created", func(t *testing.T) {
		ed := &fakeEditor{t: t, edits: []func(string) string{unchanged}}
		calls := 0
		cancelled, err := editAndCreate("# intro\n", []byte(testManifest), ed.launch, func(*unstructured.Unstructured) error {
			calls++
			return nil
		})
		if err != nil || cancelled || calls != 1 {
			t.Errorf("cancelled = %v, err = %v, calls = %d", cancelled, err, calls)
		}
	})

	t.Run("empty file cancels", func(t *testing.T) {
		for _, saved := range []string{"", "# intro\n#\n\n"} {
			ed := &fakeEditor{t: t, edits: []func(string) string{func(string) string { return saved }}}
			cancelled, err := editAndCreate("# intro\n", []byte(testManifest), ed.launch, func(*unstructured.Unstructured) error {
				t.Fatal("create must not be called")
				return nil
			})
			if err != nil || !cancelled {
				t.Errorf("saved %q: cancelled = %v, err = %v", saved, cancelled, err)
			}
		}
	})

	t.Run("reopens with the error and keeps the user's edits", func(t *testing.T) {
		ed := &fakeEditor{t: t, edits: []func(string) string{
			func(s string) string { return strings.Replace(s, "a-copy", "taken", 1) },
			func(s string) string { return strings.Replace(s, "name: taken", "name: free", 1) },
		}}
		var names []string
		cancelled, err := editAndCreate("# intro\n", []byte(testManifest), ed.launch, func(o *unstructured.Unstructured) error {
			names = append(names, o.GetName())
			if o.GetName() == "taken" {
				return errors.New("configmaps \"taken\" already exists")
			}
			return nil
		})
		if err != nil || cancelled {
			t.Fatalf("cancelled = %v, err = %v", cancelled, err)
		}
		if !reflect.DeepEqual(names, []string{"taken", "free"}) {
			t.Errorf("create called with %v", names)
		}
		second := ed.shown[1]
		if !strings.Contains(second, "# The copy was not created:\n# configmaps \"taken\" already exists\n") {
			t.Errorf("error not shown in editor:\n%s", second)
		}
		if !strings.Contains(second, "name: taken") {
			t.Errorf("user's edit lost:\n%s", second)
		}
	})

	t.Run("invalid yaml reopens the editor", func(t *testing.T) {
		ed := &fakeEditor{t: t, edits: []func(string) string{
			func(s string) string { return s + "  bad: [unclosed\n" },
			func(s string) string { return strings.Replace(s, "  bad: [unclosed\n", "", 1) },
		}}
		calls := 0
		cancelled, err := editAndCreate("# intro\n", []byte(testManifest), ed.launch, func(*unstructured.Unstructured) error {
			calls++
			return nil
		})
		if err != nil || cancelled || calls != 1 {
			t.Errorf("cancelled = %v, err = %v, calls = %d", cancelled, err, calls)
		}
	})

	t.Run("no changes after an error gives up and keeps the file", func(t *testing.T) {
		ed := &fakeEditor{t: t, edits: []func(string) string{unchanged, unchanged}}
		calls := 0
		cancelled, err := editAndCreate("# intro\n", []byte(testManifest), ed.launch, func(*unstructured.Unstructured) error {
			calls++
			return errors.New("denied")
		})
		if err == nil || cancelled || calls != 1 {
			t.Fatalf("cancelled = %v, err = %v, calls = %d", cancelled, err, calls)
		}
		if !strings.Contains(err.Error(), "denied") || !strings.Contains(err.Error(), ed.files[1]) {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists(ed.files[1]) {
			t.Error("manifest file should be kept for the user")
		}
	})
}
