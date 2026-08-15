package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteDiagnostics writes report to path using a temp file + rename.
func WriteDiagnostics(path, report string) error {
	if path == "" {
		return fmt.Errorf("export: empty path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("export: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "molebar-diag-*.tmp")
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.WriteString(report); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("export: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("export: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("export: %w", err)
	}
	cleanup = false
	return nil
}

// DefaultDiagnosticsPath is a non-interactive fallback location.
func DefaultDiagnosticsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	name := fmt.Sprintf("molebar-diagnostics-%s.txt", time.Now().UTC().Format("20060102-150405"))
	return filepath.Join(dir, "molebar", name)
}

// SavePathChooser asks the user where to write a diagnostics file.
type SavePathChooser interface {
	Choose(ctx context.Context, defaultName string) (string, error)
}
