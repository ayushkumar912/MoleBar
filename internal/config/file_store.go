package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appConfigDirName = "molebar"
	displayModeFile  = "display_mode"
)

// FileStore persists the display-mode preference as a single file.
type FileStore struct {
	path string
}

// NewFileStore writes and reads the preference at path.
// An empty path uses the default user config location
// ($XDG / ~/Library/Application Support/molebar/display_mode).
func NewFileStore(path string) *FileStore {
	if path == "" {
		path = DefaultDisplayModePath()
	}
	return &FileStore{path: path}
}

// DefaultDisplayModePath is the on-disk location of the saved display mode.
// It is empty when the user config directory cannot be resolved.
func DefaultDisplayModePath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, appConfigDirName, displayModeFile)
}

func (s *FileStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Load returns the saved mode. Missing or invalid files are not an error:
// ok is false so the caller can apply the default without writing.
func (s *FileStore) Load() (DisplayMode, bool, error) {
	if s == nil || s.path == "" {
		return DefaultDisplayMode, false, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultDisplayMode, false, nil
		}
		return DefaultDisplayMode, false, fmt.Errorf("read display mode: %w", err)
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return DefaultDisplayMode, false, nil
	}
	mode, err := ParseDisplayMode(raw)
	if err != nil {
		return DefaultDisplayMode, false, nil
	}
	return mode, true, nil
}

// Save replaces the preference file. It writes to a temp file in the same
// directory and renames into place so a crash cannot leave a truncated file.
func (s *FileStore) Save(mode DisplayMode) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("save display mode: empty path")
	}
	mode = NormalizeDisplayMode(string(mode))
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "display_mode-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.WriteString(mode.String()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	cleanup = false
	return nil
}
