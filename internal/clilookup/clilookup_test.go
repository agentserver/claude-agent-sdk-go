package clilookup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCLI_ExplicitPath(t *testing.T) {
	// Create a temp file to act as the CLI binary.
	dir := t.TempDir()
	fakeCLI := filepath.Join(dir, "claude")
	if err := os.WriteFile(fakeCLI, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	path, err := FindCLI(fakeCLI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != fakeCLI {
		t.Errorf("expected %q, got %q", fakeCLI, path)
	}
}

func TestFindCLI_ExplicitPathNotFound(t *testing.T) {
	_, err := FindCLI("/nonexistent/path/claude")
	if err == nil {
		t.Fatal("expected error for nonexistent explicit path")
	}
}

func TestFindCLI_SearchPaths(t *testing.T) {
	// Test that FindCLI returns an error listing searched paths when nothing is found.
	_, err := FindCLI("")
	if err == nil {
		// claude might actually be installed; skip the assertion.
		t.Skip("claude CLI found on system, cannot test not-found path")
	}
	notFound, ok := err.(*NotFoundError)
	if !ok {
		t.Fatalf("expected *NotFoundError, got %T", err)
	}
	if len(notFound.SearchedPaths) == 0 {
		t.Error("SearchedPaths should not be empty")
	}
}
