package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Version-lockstep test — the repo carries the version in more than one place
// (the VERSION file consumed by release tooling, the binary's version constant
// feeding cobra's --version, and CHANGELOG.md's latest section). This test
// fails the suite the moment any one surface drifts from the others.

func TestVersionFileMatchesBinaryConstant(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	fileVersion := strings.TrimSpace(string(b))
	if fileVersion == "" {
		t.Fatalf("VERSION file is empty")
	}
	if fileVersion != version {
		t.Fatalf("VERSION file = %q, binary version constant = %q", fileVersion, version)
	}
}

func TestRootCmdVersionOutput(t *testing.T) {
	// cobra registers --version from rootCmd.Version; assert in-process
	// (no subprocess, no binary availability guards needed).
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}
	want := "ofdcli version " + version
	if got := buf.String(); !strings.Contains(got, want) {
		t.Fatalf("--version output = %q, want it to contain %q", got, want)
	}
}

func TestChangelogCarriesCurrentVersion(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	if !strings.Contains(string(b), "## ["+version+"]") {
		t.Fatalf("CHANGELOG.md has no section header for the current version %q", version)
	}
}

// TestWriteCmd_SameInputOutputRejected is the CLI-level regression test for
// the v0.2.0 defect where `ofdcli write -i X.ofd -o X.ofd` truncated the
// still-open input archive and left the document at 0 bytes. The command
// must refuse the invocation and leave the file byte-identical.
func TestWriteCmd_SameInputOutputRejected(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "examples", "sample.ofd"))
	if err != nil {
		t.Fatalf("read examples/sample.ofd: %v", err)
	}
	f := filepath.Join(t.TempDir(), "inplace.ofd")
	if err := os.WriteFile(f, src, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// main() registers the subcommands on rootCmd; tests must do the same.
	rootCmd.AddCommand(readCmd, writeCmd, infoCmd)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"write", "-i", f, "-o", f})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("write with -i == -o returned nil error, want a same-file error")
	}
	if !strings.Contains(buf.String(), "same file") && !strings.Contains(fmt.Sprint(err), "same file") {
		t.Fatalf("error should name the same-file conflict, got: %v / %q", err, buf.String())
	}
	after, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("re-read fixture: %v", err)
	}
	if !bytes.Equal(src, after) {
		t.Fatalf("REPRODUCED (v0.2.0 truncated the input to 0 bytes): fixture was modified, %d -> %d bytes", len(src), len(after))
	}
}
