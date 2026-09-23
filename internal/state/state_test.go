package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// isolateConfig points the state file at a temp directory for the duration of the
// test, on every supported OS.
func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
	return dir
}

func TestSaveLoadRoundTrip(t *testing.T) {
	isolateConfig(t)

	if err := Save(State{LastProfile: "egf-prd"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.LastProfile != "egf-prd" {
		t.Errorf("LastProfile = %q, want %q", got.LastProfile, "egf-prd")
	}
}

func TestLoadMissingFileReturnsZero(t *testing.T) {
	isolateConfig(t)
	got, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if got.LastProfile != "" {
		t.Errorf("expected empty LastProfile, got %q", got.LastProfile)
	}
}

func TestLoadCorruptFileReturnsZero(t *testing.T) {
	isolateConfig(t)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := mkdirAllWrite(path, "{ not valid json"); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load on corrupt file should not error: %v", err)
	}
	if got.LastProfile != "" {
		t.Errorf("expected empty LastProfile on corrupt file, got %q", got.LastProfile)
	}
}

func TestPathHonoursXDG(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is not used on Windows")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "aws-tui", "state.json")
	if path != want {
		t.Errorf("Path = %q, want %q", path, want)
	}
}

// mkdirAllWrite creates parent dirs and writes content to path.
func mkdirAllWrite(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
