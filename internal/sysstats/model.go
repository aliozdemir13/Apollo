package sysstats

// Stats is the snapshot returned to the frontend.
type Stats struct {
	CPUPercent   float64 `json:"cpuPercent"`   // 0-100
	MemPercent   float64 `json:"memPercent"`   // 0-100
	MemUsedGB    float64 `json:"memUsedGB"`    // GiB
	MemTotalGB   float64 `json:"memTotalGB"`   // GiB
	BatteryPct   float64 `json:"batteryPct"`   // 0-100, -1 if no battery
	BatteryState string  `json:"batteryState"` // "charging" | "discharging" | "full" | "n/a"
	UptimeHours  float64 `json:"uptimeHours"`
	Hostname     string  `json:"hostname"`
}

// Process is a single running process for the "top consumers" list.
type Process struct {
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"` // %CPU
	Mem  float64 `json:"mem"` // %memory
}
