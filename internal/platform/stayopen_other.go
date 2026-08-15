//go:build !darwin

package platform

func keepMenuOpenOnToggles() {}

func MenuIsTracking() bool { return false }

func SetMenuClosedHandler(func()) {}

func SyncStayOpenMenu() {}
