package platform

import "testing"

func TestKeepMenuOpenOnTogglesDoesNotPanic(t *testing.T) {
	KeepMenuOpenOnToggles()
	if MenuIsTracking() {
		t.Fatal("menu should not be tracking before the tray is shown")
	}
}
