package main

import (
	"bytes"
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
