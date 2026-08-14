package scripts

import (
	"strings"
	"testing"
)

func TestShellMoveFallbackIsLocaleIndependent(t *testing.T) {
	shell := string(Shell)
	if strings.Contains(shell, "Permission denied") {
		t.Fatal("move fallback depends on an English error message")
	}
	if !strings.Contains(shell, `if [[ $MOVE = "true" ]]; then`) {
		t.Fatal("system move failure does not retry based on move intent")
	}
}
