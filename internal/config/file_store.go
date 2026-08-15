package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appConfigDirName = "molebar"
	configFileName   = "config.json"
	displayModeFile  = "display_mode"
)

// FileStore persists versioned preferences as JSON.
type FileStore struct {
	path string
}

// NewFileStore writes and reads the preference at path.
// An empty path uses the default user config location
// ($XDG / ~/Library/Application Support/molebar/config.json).
func NewFileStore(path string) *FileStore {
	if path == "" {
		path = DefaultConfigPath()
	}
	return &FileStore{path: path}
}

// DefaultConfigPath is the on-disk location of the saved preferences.
// It is empty when the user config directory cannot be resolved.
func DefaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, appConfigDirName, configFileName)
}

// DefaultDisplayModePath is the legacy on-disk location of the saved
// display mode. It is read during migration and is never deleted.
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

// Load returns saved preferences. Missing files are not an error: ok is
// false so the caller can apply defaults without writing. A legacy
// display_mode file (plain "sys"/"net"/"both") is migrated in memory
// and the original file is left untouched.
func (s *FileStore) Load() (Preferences, bool, error) {
	if s == nil || s.path == "" {
		return DefaultPreferences(), false, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.loadLegacySibling()
		}
		return DefaultPreferences(), false, fmt.Errorf("read config: %w", err)
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return s.loadLegacySibling()
	}
	if raw[0] != '{' {
		return parseLegacyDisplayMode(raw)
	}
	var p Preferences
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return DefaultPreferences(), false, nil
	}
	return p.Normalize(), true, nil
}

func (s *FileStore) loadLegacySibling() (Preferences, bool, error) {
	legacy := s.legacyPath()
	if legacy == "" || legacy == s.path {
		return DefaultPreferences(), false, nil
	}
	data, err := os.ReadFile(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultPreferences(), false, nil
		}
		return DefaultPreferences(), false, fmt.Errorf("read display mode: %w", err)
	}
	return parseLegacyDisplayMode(strings.TrimSpace(string(data)))
}

func (s *FileStore) legacyPath() string {
	if s == nil || s.path == "" {
		return ""
	}
	dir := filepath.Dir(s.path)
	if filepath.Base(s.path) == displayModeFile {
		return s.path
	}
	return filepath.Join(dir, displayModeFile)
}

func parseLegacyDisplayMode(raw string) (Preferences, bool, error) {
	if raw == "" {
		return DefaultPreferences(), false, nil
	}
	mode, err := ParseDisplayMode(raw)
	if err != nil {
		return DefaultPreferences(), false, nil
	}
	p := DefaultPreferences()
	p.ApplyDisplayMode(mode)
	return p.Normalize(), true, nil
}

// Save replaces the preference file. It writes to a temp file in the same
// directory and renames into place so a crash cannot leave a truncated file.
// A legacy display_mode sibling is never deleted.
func (s *FileStore) Save(prefs Preferences) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("save config: empty path")
	}
	prefs = prefs.Normalize()
	prefs.Version = CurrentVersion
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, "config-*.tmp")
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
	if _, err := tmp.Write(data); err != nil {
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
