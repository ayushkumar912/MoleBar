package config

import "strings"

// Store is the persisted user-preference backend.
// A small interface exists so tests can inject a path or memory store;
// production uses FileStore.
type Store interface {
	Load() (Preferences, bool, error)
	Save(Preferences) error
}

// ResolveDisplayMode returns the effective runtime mode.
// An explicit CLI value overrides the saved preference for this process only
// and is never written back. An empty CLI value uses the store, then the default.
func ResolveDisplayMode(store Store, cliTitle string) DisplayMode {
	return ResolvePreferences(store, cliTitle).DisplayMode()
}

// ResolvePreferences returns persisted preferences with an optional
// runtime-only CLI title override applied to the layout.
func ResolvePreferences(store Store, cliTitle string) Preferences {
	prefs := DefaultPreferences()
	if store != nil {
		if loaded, ok, err := store.Load(); err == nil && ok {
			prefs = loaded.Normalize()
		}
	}
	if strings.TrimSpace(cliTitle) != "" {
		mode, err := ParseDisplayMode(cliTitle)
		if err != nil {
			return DefaultPreferences()
		}
		prefs.ApplyDisplayMode(mode)
	}
	return prefs.Normalize()
}
