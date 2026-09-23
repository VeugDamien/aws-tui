// Package state persists small bits of user state (e.g. the last used profile)
// across runs in a JSON file under the user's config directory.
package state

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// State holds the persisted user state. New fields can be added over time; unknown
// fields in the file are ignored on load and preserved fields default to zero.
type State struct {
	LastProfile string `json:"last_profile"`
}

// dirName is the application's config subdirectory.
const dirName = "aws-tui"

// fileName is the state file name.
const fileName = "state.json"

// configDir returns the base configuration directory for the current OS.
//   - Windows: %AppData% (via os.UserConfigDir).
//   - Unix/macOS: $XDG_CONFIG_HOME, falling back to ~/.config (so macOS uses
//     ~/.config rather than ~/Library/Application Support, per project choice).
func configDir() (string, error) {
	if runtime.GOOS == "windows" {
		return os.UserConfigDir()
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// Path returns the full path to the state file.
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dirName, fileName), nil
}

// Load reads the state file. A missing file is not an error: it returns a zero
// State so first runs behave gracefully.
func Load() (State, error) {
	var s State
	path, err := Path()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, nil
		}
		return s, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		// Corrupt file: treat as empty rather than failing the app.
		return State{}, nil
	}
	return s, nil
}

// Save writes the state file atomically, creating the config directory if needed.
func Save(s State) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	// Write to a temp file then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
