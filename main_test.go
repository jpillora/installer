package main

import (
	"testing"
)

func Test_pathRegex(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		matches []string
	}{
		{
			name:    "empty",
			path:    "",
			matches: []string{},
		},
		{
			name:    "kots install",
			path:    "/replicatedhq/kots@v0.9.8!",
			matches: []string{"/replicatedhq/kots@v0.9.8!", "/replicatedhq", "replicatedhq", "kots", "@v0.9.8", "v0.9.8", "!"},
		},
		{
			name:    "minimal install",
			path:    "/replicatedhq/kots",
			matches: []string{"/replicatedhq/kots", "/replicatedhq", "replicatedhq", "kots", "", "", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualMatch := pathRe.FindStringSubmatch(tt.path)
			if len(actualMatch) != len(tt.matches) {
				t.Fatalf("expected %d matches, got %d", len(tt.matches), len(actualMatch))
			}

			for idx, match := range tt.matches {
				if match != actualMatch[idx] {
					t.Fatalf("expected %q at position %d, got %q", match, idx, actualMatch[idx])
				}
			}
		})
	}
}
