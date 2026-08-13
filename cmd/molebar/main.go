// Command molebar is a macOS menu-bar widget that shells out to Mole
// (`mo status --json`, https://github.com/tw93/Mole) on an interval and
// shows CPU, memory, swap, disk, battery, and health-score in the menu bar.
//
// Requires the `mo` CLI to be installed and on $PATH:
//
//	brew install tw93/tap/mole
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/getlantern/systray"

	"github.com/ayush-kumar912/molebar/internal/molestatus"
)

var (
	interval = flag.Duration("interval", 5*time.Second, "refresh interval")
	binPath  = flag.String("mo-bin", "", `path to the "mo" executable (default: resolve "mo" from $PATH)`)
)

func main() {
	flag.Parse()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("mo …")
	systray.SetTooltip("Mole system status")

	mCPU := systray.AddMenuItem("CPU: —", "")
	mMem := systray.AddMenuItem("Memory: —", "")
	mSwap := systray.AddMenuItem("Swap: —", "")
	mDisk := systray.AddMenuItem("Disk: —", "")
	mBatt := systray.AddMenuItem("Battery: —", "")
	systray.AddSeparator()
	mHealth := systray.AddMenuItem("Health: —", "")
	mUpdated := systray.AddMenuItem("Updated: —", "")
	mUpdated.Disable()
	systray.AddSeparator()
	mRefresh := systray.AddMenuItem("Refresh now", "Fetch mo status immediately")
	mQuit := systray.AddMenuItem("Quit", "Quit molebar")

	fetcher := &molestatus.Fetcher{BinPath: *binPath}

	refresh := func() {
		s, err := fetcher.Fetch()
		if err != nil {
			// Don't crash the tray on a transient failure — surface it in
			// the title/log and keep the last-good values in the dropdown.
			systray.SetTitle("mo: err")
			log.Printf("molebar: refresh failed: %v", err)
			return
		}

		systray.SetTitle(fmt.Sprintf("CPU %.0f%% MEM %.0f%%", s.CPU.Usage, s.Memory.UsedPercent))
		systray.SetTooltip(fmt.Sprintf("Health %d (%s)", s.HealthScore, s.HealthMsg))

		mCPU.SetTitle(fmt.Sprintf("CPU: %.1f%%  (load1 %.2f)", s.CPU.Usage, s.CPU.Load1))
		mMem.SetTitle(fmt.Sprintf("Memory: %.1f%%", s.Memory.UsedPercent))
		mSwap.SetTitle(fmt.Sprintf("Swap: %.1f%%", s.SwapPercent()))

		if pct := s.PrimaryDiskPercent(); pct >= 0 {
			mDisk.SetTitle(fmt.Sprintf("Disk: %.1f%%", pct))
		} else {
			mDisk.SetTitle("Disk: n/a")
		}

		if pct, status, ok := s.PrimaryBattery(); ok {
			mBatt.SetTitle(fmt.Sprintf("Battery: %d%% (%s)", pct, status))
		} else {
			mBatt.SetTitle("Battery: n/a")
		}

		mHealth.SetTitle(fmt.Sprintf("Health: %d (%s)", s.HealthScore, s.HealthMsg))
		mUpdated.SetTitle("Updated: " + time.Now().Format("15:04:05"))
	}

	refresh()
	ticker := time.NewTicker(*interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				refresh()
			case <-mRefresh.ClickedCh:
				refresh()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// No persistent state to clean up.
}
