package cmd

import (
	"strings"
	"testing"
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
