package platform

import (
	"errors"
	"testing"
)

func TestLoginItemUnsupportedWithoutPath(t *testing.T) {
	m := NewDarwinLoginItem("")
	if m.Path != "" {
		// currentAppPath may resolve an executable in this test process.
		m.Path = ""
	}
	if _, err := m.Enabled(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Enabled err = %v", err)
	}
	if err := m.SetEnabled(true); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetEnabled err = %v", err)
	}
}

func TestQuoteAppleScriptEscapes(t *testing.T) {
	got := quoteAppleScript(`say "hi"\`)
	if got != `"say \"hi\"\\"` {
		t.Fatalf("got %s", got)
	}
}
