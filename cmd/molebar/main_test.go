package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ayush-kumar912/molebar/internal/config"
)

func TestValidateInterval(t *testing.T) {
	if err := validateInterval(0); err == nil {
		t.Fatal("zero interval must be rejected")
	}
	if err := validateInterval(-time.Second); err == nil {
		t.Fatal("negative interval must be rejected")
	}
	if err := validateInterval(time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestParseRuntimeRejectsInvalidInterval(t *testing.T) {
	fs := flag.NewFlagSet("molebar", flag.ContinueOnError)
	fs.SetOutput(&bytes.Buffer{})
	_, err := parseRuntime(fs, []string{"-interval=0"}, config.NewFileStore(filepath.Join(t.TempDir(), "display_mode")))
	if err == nil {
		t.Fatal("expected interval error")
	}
}

func TestParseRuntimeCLITitleDoesNotPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display_mode")
	if err := os.WriteFile(path, []byte("sys"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := config.NewFileStore(path)
	fs := flag.NewFlagSet("molebar", flag.ContinueOnError)
	cfg, err := parseRuntime(fs, []string{"-title=net"}, store)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DisplayMode != config.DisplayModeNet {
		t.Fatalf("mode = %q", cfg.DisplayMode)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sys" {
		t.Fatalf("saved preference mutated to %q", data)
	}
}

func TestParseRuntimeUsesSavedPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display_mode")
	store := config.NewFileStore(path)
	if err := store.Save(config.DisplayModeBoth); err != nil {
		t.Fatal(err)
	}
	fs := flag.NewFlagSet("molebar", flag.ContinueOnError)
	cfg, err := parseRuntime(fs, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DisplayMode != config.DisplayModeBoth {
		t.Fatalf("mode = %q", cfg.DisplayMode)
	}
}
