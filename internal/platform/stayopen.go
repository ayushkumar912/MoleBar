package platform

// KeepMenuOpenOnToggles prevents the tray menu from dismissing when
// the user clicks a checkbox-style item (tray metrics, profiles,
// alerts, launch at login). Action items such as Quit still close it.
func KeepMenuOpenOnToggles() {
	keepMenuOpenOnToggles()
}
