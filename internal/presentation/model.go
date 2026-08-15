package presentation

// ViewModel is everything the tray needs to paint one frame.
type ViewModel struct {
	Title   string
	Tooltip string

	CPU         string
	Memory      string
	Swap        string
	Disk        string
	Battery     string
	Temperature string
	Health      string

	Down            string
	Up              string
	Session         string
	SessionRX       string
	SessionTX       string
	PeakRX          string
	PeakTX          string
	AvgRX           string
	AvgTX           string
	SessionDuration string

	Updated string

	DisplayLabel string
	ModeSys      bool
	ModeNet      bool
	ModeBoth     bool

	ProcessRows []ProcessRow
	AlertRows   []AlertRow
	ProfileRows []ProfileRow
	MetricRows  []MetricRow

	AlertsEnabled          bool
	LaunchAtLogin          bool
	LaunchAtLoginSupported bool

	SystemSummary string
	Diagnostics   string
}

// ProcessRow is one read-only top-process line.
type ProcessRow struct {
	Title string
}

// AlertRow is one active alert line.
type AlertRow struct {
	Title string
}

// ProfileRow is one profile menu entry.
type ProfileRow struct {
	ID      string
	Label   string
	Checked bool
}

// MetricRow is one tray-metric menu entry.
type MetricRow struct {
	ID      string
	Label   string
	Checked bool
}
