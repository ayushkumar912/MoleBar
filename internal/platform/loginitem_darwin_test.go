//go:build darwin

package platform

import (
	"context"
	"strings"
	"testing"
)

func TestDarwinLoginItemEnabledAndToggle(t *testing.T) {
	var scripts []string
	listed := ""
	m := &DarwinLoginItem{
		Path: "/Applications/MoleBar.app",
		runner: func(_ context.Context, stdin string) (string, error) {
			scripts = append(scripts, stdin)
			if strings.Contains(stdin, "get the path") {
				return listed, nil
			}
			if strings.Contains(stdin, "make login item") {
				listed = "/Applications/MoleBar.app"
				return "", nil
			}
			if strings.Contains(stdin, "delete") {
				listed = ""
				return "", nil
			}
			return "", nil
		},
	}
	on, err := m.Enabled()
	if err != nil || on {
		t.Fatalf("initial enabled=%v err=%v", on, err)
	}
	if err := m.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	on, err = m.Enabled()
	if err != nil || !on {
		t.Fatalf("after enable %v %v", on, err)
	}
	if err := m.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	on, err = m.Enabled()
	if err != nil || on {
		t.Fatalf("after disable %v %v", on, err)
	}
	if len(scripts) < 4 {
		t.Fatalf("scripts = %#v", scripts)
	}
}
