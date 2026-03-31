package clilookup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// NotFoundError is returned when the claude CLI binary cannot be found.
type NotFoundError struct {
	SearchedPaths []string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("claude CLI not found (searched: %s)", strings.Join(e.SearchedPaths, ", "))
}

// FindCLI locates the claude CLI binary.
// If explicitPath is non-empty, it is used directly (must exist and be executable).
// Otherwise, searches standard locations and $PATH.
func FindCLI(explicitPath string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", &NotFoundError{SearchedPaths: []string{explicitPath}}
		}
		return explicitPath, nil
	}

	var searched []string

	// 1. Check $PATH via exec.LookPath.
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}
	searched = append(searched, "$PATH")

	// 2. Standard installation locations.
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".claude", "local", "claude"),
		"/usr/local/bin/claude",
		"/usr/bin/claude",
	}

	if runtime.GOOS == "darwin" {
		candidates = append(candidates,
			"/opt/homebrew/bin/claude",
		)
	}

	// On Windows, also check .exe variants.
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			candidates = append(candidates, filepath.Join(appData, "npm", "claude.cmd"))
		}
	}

	// 3. Check npm global installations.
	npmGlobal := npmGlobalBinDir()
	if npmGlobal != "" {
		candidates = append(candidates, filepath.Join(npmGlobal, "claude"))
	}

	for _, path := range candidates {
		searched = append(searched, path)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}

	return "", &NotFoundError{SearchedPaths: searched}
}

// npmGlobalBinDir returns the npm global bin directory, or empty string.
// Uses "npm config get prefix" since "npm bin -g" is deprecated since npm 9.
func npmGlobalBinDir() string {
	out, err := exec.Command("npm", "config", "get", "prefix").Output()
	if err != nil {
		return ""
	}
	prefix := strings.TrimSpace(string(out))
	if prefix == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return prefix // npm installs binaries directly in prefix on Windows
	}
	return filepath.Join(prefix, "bin")
}
