package core

import (
	"reflect"
	"testing"
)

func TestEnsureNoTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"fetch adds flag", []string{"fetch", "--quiet", "origin"}, []string{"fetch", "--no-tags", "--quiet", "origin"}},
		{"pull adds flag", []string{"pull"}, []string{"pull", "--no-tags"}},
		{"pull already has flag", []string{"pull", "--no-tags"}, []string{"pull", "--no-tags"}},
		{"fetch explicit tags unchanged", []string{"fetch", "--tags", "origin"}, []string{"fetch", "--tags", "origin"}},
		{"status unchanged", []string{"status", "--porcelain"}, []string{"status", "--porcelain"}},
		{"empty", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureNoTags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ensureNoTags(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
