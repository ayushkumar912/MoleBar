//go:build !darwin

package platform

// DarwinLoginItem is unavailable off macOS.
type DarwinLoginItem struct {
	Path string
}

func NewDarwinLoginItem(string) *DarwinLoginItem { return &DarwinLoginItem{} }

func (m *DarwinLoginItem) Enabled() (bool, error) { return false, ErrUnsupported }

func (m *DarwinLoginItem) SetEnabled(bool) error { return ErrUnsupported }
