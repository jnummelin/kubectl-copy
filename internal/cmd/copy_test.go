package cmd

import (
	"bytes"
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args         []string
		wantResource string
		wantName     string
		wantErr      bool
	}{
		{args: []string{"pod/a"}, wantResource: "pod/a"},
		{args: []string{"pod/a", "b"}, wantResource: "pod/a", wantName: "b"},
		{args: []string{"widgets.example.com/a"}, wantResource: "widgets.example.com/a"},
		{args: []string{"pod"}, wantErr: true},
		{args: []string{"pod", "a"}, wantErr: true},
		{args: []string{"pod/"}, wantErr: true},
		{args: []string{"/a"}, wantErr: true},
		{args: []string{"pod/a/b"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			res, name, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if res != tt.wantResource || name != tt.wantName {
				t.Errorf("got (%q, %q), want (%q, %q)", res, name, tt.wantResource, tt.wantName)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	version = "v1.2.3"
	t.Cleanup(func() { version = "" })

	var out bytes.Buffer
	cmd := NewCopyCommand(genericiooptions.IOStreams{In: strings.NewReader(""), Out: &out, ErrOut: &out})
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "kubectl-copy v1.2.3\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if f := cmd.Flags().ShorthandLookup("v"); f != nil {
		t.Errorf("-v must stay free, but is the shorthand of --%s", f.Name)
	}
}
