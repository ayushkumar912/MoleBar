package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreMissingPreference(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "config.json"))
	prefs, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if ok {
		t.Fatalf("Load missing ok=true, prefs=%+v", prefs)
	}
	if prefs.DisplayMode() != DefaultDisplayMode {
		t.Fatalf("Load missing mode=%q", prefs.DisplayMode())
	}
}

func TestFileStoreValidLegacyPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display_mode")
	if err := os.WriteFile(path, []byte("net\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	prefs, ok, err := store.Load()
	if err != nil || !ok {
		t.Fatalf("Load valid: ok=%v err=%v", ok, err)
	}
	if prefs.DisplayMode() != DisplayModeNet {
		t.Fatalf("got %q", prefs.DisplayMode())
	}
	if !prefs.Layout.Equal(LayoutFromDisplayMode(DisplayModeNet)) {
		t.Fatalf("layout = %+v", prefs.Layout)
	}
}

func TestFileStoreMigratesSiblingDisplayMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "display_mode"), []byte("both"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(filepath.Join(dir, "config.json"))
	prefs, ok, err := store.Load()
	if err != nil || !ok {
		t.Fatalf("sibling migrate: ok=%v err=%v", ok, err)
	}
	if prefs.DisplayMode() != DisplayModeBoth {
		t.Fatalf("mode = %q", prefs.DisplayMode())
	}
	if _, err := os.Stat(filepath.Join(dir, "display_mode")); err != nil {
		t.Fatalf("legacy file was removed: %v", err)
	}
}

func TestFileStoreInvalidStoredValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display_mode")
	if err := os.WriteFile(path, []byte("bogus"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	prefs, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load invalid: %v", err)
	}
	if ok {
		t.Fatalf("invalid stored value should not be ok, got %+v", prefs)
	}
}

func TestFileStoreSaveReadRoundTrip(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "molebar", "config.json"))
	prefs := DefaultPreferences()
	prefs.ApplyDisplayMode(DisplayModeBoth)
	if err := store.Save(prefs); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Load()
	if err != nil || !ok || got.DisplayMode() != DisplayModeBoth {
		t.Fatalf("round trip: mode=%q ok=%v err=%v", got.DisplayMode(), ok, err)
	}
	if got.Version != CurrentVersion {
		t.Fatalf("version = %d", got.Version)
	}
}

func TestFileStoreSaveDoesNotDeleteLegacy(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "display_mode")
	if err := os.WriteFile(legacy, []byte("sys"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(filepath.Join(dir, "config.json"))
	prefs := DefaultPreferences()
	prefs.ApplyProfile("developer")
	if err := store.Save(prefs); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sys" {
		t.Fatalf("legacy mutated to %q", data)
	}
}

func TestCLIOverrideNotPersistedImplicitly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "display_mode")
	if err := os.WriteFile(path, []byte("sys"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)

	got := ResolveDisplayMode(store, "net")
	if got != DisplayModeNet {
		t.Fatalf("ResolveDisplayMode CLI = %q", got)
	}

	prefs, ok, err := store.Load()
	if err != nil || !ok || prefs.DisplayMode() != DisplayModeSys {
		t.Fatalf("store mutated: mode=%q ok=%v err=%v", prefs.DisplayMode(), ok, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sys" {
		t.Fatalf("file contents = %q, want sys (CLI override must not persist)", data)
	}
}

func TestResolveDisplayModeUsesStoreWhenCLIEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewFileStore(path)
	prefs := DefaultPreferences()
	prefs.ApplyDisplayMode(DisplayModeNet)
	if err := store.Save(prefs); err != nil {
		t.Fatal(err)
	}
	if got := ResolveDisplayMode(store, ""); got != DisplayModeNet {
		t.Fatalf("got %q", got)
	}
	if got := ResolveDisplayMode(store, "   "); got != DisplayModeNet {
		t.Fatalf("whitespace CLI = %q", got)
	}
}
