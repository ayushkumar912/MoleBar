//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
void install_stay_open_menus(void);
int molebar_menu_is_tracking(void);
void molebar_sync_stay_open_menu(void);
*/
import "C"

import "sync"

var (
	menuClosedMu      sync.Mutex
	menuClosedHandler func()
)

func keepMenuOpenOnToggles() {
	C.install_stay_open_menus()
}

// SyncStayOpenMenu copies current check state onto the stay-open checkbox views.
func SyncStayOpenMenu() {
	C.molebar_sync_stay_open_menu()
}

// MenuIsTracking reports whether a macOS menu is currently open.
func MenuIsTracking() bool {
	return C.molebar_menu_is_tracking() != 0
}

// SetMenuClosedHandler is called on the AppKit thread after the tray menu
// finishes tracking. Used to apply a deferred status-item title.
func SetMenuClosedHandler(fn func()) {
	menuClosedMu.Lock()
	menuClosedHandler = fn
	menuClosedMu.Unlock()
}

//export molebarMenuTrackingEnded
func molebarMenuTrackingEnded() {
	menuClosedMu.Lock()
	h := menuClosedHandler
	menuClosedMu.Unlock()
	if h != nil {
		h()
	}
}
