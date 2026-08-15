package molestatus

// Status is a partial mirror of `mo status --json` / watch NDJSON.
// Only fields MoleBar consumes are modeled; unknown keys are ignored.
type Status struct {
	CollectedAt string `json:"collected_at"`
	HealthScore *int   `json:"health_score"`
	HealthMsg   string `json:"health_score_msg"`

	CPU struct {
		Usage float64   `json:"usage"`
		Load1 float64   `json:"load1"`
		Cores []float64 `json:"per_core"`
	} `json:"cpu"`

	Memory struct {
		UsedPercent float64 `json:"used_percent"`
		Total       int64   `json:"total"`
		Used        int64   `json:"used"`
		Available   int64   `json:"available"`
		SwapUsed    int64   `json:"swap_used"`
		SwapTotal   int64   `json:"swap_total"`
	} `json:"memory"`

	Disks        []Disk        `json:"disks"`
	Batteries    []Battery     `json:"batteries"`
	Network      []Network     `json:"network"`
	Thermal      Thermal       `json:"thermal"`
	TopProcesses []ProcessInfo `json:"top_processes"`
	Procs        int           `json:"procs"`
}

// Disk is one volume from Mole's disks array.
type Disk struct {
	Mount       string  `json:"mount"`
	UsedPercent float64 `json:"used_percent"`
	Total       int64   `json:"total"`
	Used        int64   `json:"used"`
	Purgeable   int64   `json:"purgeable"`
}

// Battery matches Mole's BatteryStatus. Percent is float64 in the upstream schema.
// Optional fields use zero/empty as "Mole omitted or sent empty"; callers
// should treat presence via PrimaryBattery / PrimaryBatteryInfo.
type Battery struct {
	Percent    float64 `json:"percent"`
	Status     string  `json:"status"`
	Health     string  `json:"health"`
	CycleCount int     `json:"cycle_count"`
	Capacity   int     `json:"capacity"`
	TimeLeft   string  `json:"time_left"`
}

// BatteryInfo is the domain view of the primary battery. Pointers are nil
// when Mole did not supply a meaningful value.
type BatteryInfo struct {
	Percent    float64
	Status     string
	Charging   *bool
	Health     *string
	CycleCount *int
	Capacity   *int
}

// Thermal matches Mole's ThermalStatus. Temperatures are Celsius.
type Thermal struct {
	CPUTemp     float64 `json:"cpu_temp"`
	GPUTemp     float64 `json:"gpu_temp"`
	BatteryTemp float64 `json:"battery_temp"`
	FanSpeed    int     `json:"fan_speed"`
}

// ProcessInfo is one Mole top-process row. Command is ignored on purpose
// so MoleBar never displays or exports arbitrary command lines.
type ProcessInfo struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	MemoryBytes uint64  `json:"memory_bytes"`
}

// ProcessStat is the read-only process row MoleBar exposes.
type ProcessStat struct {
	PID        int
	Name       string
	CPUPercent float64
	Memory     uint64
}

// Network is one interface record Mole chose to include.
type Network struct {
	Name      string  `json:"name"`
	RxRateMBs float64 `json:"rx_rate_mbs"`
	TxRateMBs float64 `json:"tx_rate_mbs"`
	IP        string  `json:"ip"`
}
