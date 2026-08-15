package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreMissingPreference(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "display_mode"))
	mode, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if ok {
		t.Fatalf("Load missing ok=true, mode=%q", mode)
	}
	if mode != DefaultDisplayMode {
		t.Fatalf("Load missing mode=%q", mode)
	}
}

func TestFileStoreValidPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display_mode")
	if err := os.WriteFile(path, []byte("net\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	mode, ok, err := store.Load()
	if err != nil || !ok {
		t.Fatalf("Load valid: mode=%q ok=%v err=%v", mode, ok, err)
	}
	if mode != DisplayModeNet {
		t.Fatalf("got %q", mode)
	}
}

func TestFileStoreInvalidStoredValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display_mode")
	if err := os.WriteFile(path, []byte("bogus"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	mode, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load invalid: %v", err)
	}
	if ok {
		t.Fatalf("invalid stored value should not be ok, got %q", mode)
	}
}

func TestFileStoreSaveReadRoundTrip(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "molebar", "display_mode"))
	if err := store.Save(DisplayModeBoth); err != nil {
		t.Fatal(err)
	}
	mode, ok, err := store.Load()
	if err != nil || !ok || mode != DisplayModeBoth {
		t.Fatalf("round trip: mode=%q ok=%v err=%v", mode, ok, err)
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

	mode, ok, err := store.Load()
	if err != nil || !ok || mode != DisplayModeSys {
		t.Fatalf("store mutated: mode=%q ok=%v err=%v", mode, ok, err)
	}
	if _, err := os.ReadFile(path); err != nil {
		t.Fatal(err)
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
	path := filepath.Join(t.TempDir(), "display_mode")
	store := NewFileStore(path)
	if err := store.Save(DisplayModeNet); err != nil {
		t.Fatal(err)
	}
	if got := ResolveDisplayMode(store, ""); got != DisplayModeNet {
		t.Fatalf("got %q", got)
	}
	if got := ResolveDisplayMode(store, "   "); got != DisplayModeNet {
		t.Fatalf("whitespace CLI = %q", got)
	}
}
