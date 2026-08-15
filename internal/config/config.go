package config

import "strings"

// Store is the persisted user-preference backend.
// A small interface exists so tests can inject a path or memory store;
// production uses FileStore.
type Store interface {
	Load() (mode DisplayMode, ok bool, err error)
	Save(mode DisplayMode) error
}

// ResolveDisplayMode returns the effective runtime mode.
// An explicit CLI value overrides the saved preference for this process only
// and is never written back. An empty CLI value uses the store, then the default.
func ResolveDisplayMode(store Store, cliTitle string) DisplayMode {
	if strings.TrimSpace(cliTitle) != "" {
		mode, err := ParseDisplayMode(cliTitle)
		if err != nil {
			return DefaultDisplayMode
		}
		return mode
	}
	if store == nil {
		return DefaultDisplayMode
	}
	mode, ok, err := store.Load()
	if err != nil || !ok {
		return DefaultDisplayMode
	}
	return mode
}
