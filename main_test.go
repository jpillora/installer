package main

import (
	"bytes"
	"testing"
)

func TestCommandHandleVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			stdout := bytes.Buffer{}
			cmd := command{version: "1.2.3", stdout: &stdout}
			if !cmd.handleVersion([]string{"installer", flag}) {
				t.Fatal("version flag was not handled")
			}
			if stdout.String() != "1.2.3\n" {
				t.Fatalf("unexpected version output %q", stdout.String())
			}
		})
	}
}

func TestCommandHandleVersionIgnoresOtherArguments(t *testing.T) {
	stdout := bytes.Buffer{}
	cmd := command{version: "1.2.3", stdout: &stdout}
	if cmd.handleVersion([]string{"installer", "--help"}) {
		t.Fatal("non-version arguments were handled")
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected output %q", stdout.String())
	}
}
