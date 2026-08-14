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

func TestShellInstalledBinaryPermissions(t *testing.T) {
	shell := string(Shell)
	if strings.Contains(shell, "chmod +x") {
		t.Fatal("binary permissions depend on the current umask")
	}
	if !strings.Contains(shell, `chmod 0755 "$TMP_BIN"`) {
		t.Fatal("binary permissions do not ensure global read and execute access")
	}
}
