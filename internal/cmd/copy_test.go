package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		args         []string
		wantResource []string
		wantName     string
		wantErr      bool
	}{
		{args: []string{"pod/a"}, wantResource: []string{"pod/a"}},
		{args: []string{"pod/a", "b"}, wantResource: []string{"pod/a"}, wantName: "b"},
		{args: []string{"pod", "a"}, wantResource: []string{"pod", "a"}},
		{args: []string{"pod", "a", "b"}, wantResource: []string{"pod", "a"}, wantName: "b"},
		{args: []string{"pod"}, wantErr: true},
		{args: []string{"pod/a", "b", "c"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			res, name, err := splitArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(res, tt.wantResource) || name != tt.wantName {
				t.Errorf("got (%v, %q), want (%v, %q)", res, name, tt.wantResource, tt.wantName)
			}
		})
	}
}
