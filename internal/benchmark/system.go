package benchmark

import (
	"log"
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type SystemInfo struct {
	GoVersion    string
	OS           string
	CPUModel     string
	TotalMemory  uint64
	AvailableMem uint64
}

func GetSystemInfo() SystemInfo {
	info := SystemInfo{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
	}

	// Handle memory metrics
	memStat, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Failed to get memory stats: %v", err)
	} else {
		info.TotalMemory = memStat.Total
		info.AvailableMem = memStat.Available
	}

	// Handle CPU metrics
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		log.Printf("Failed to get CPU info: %v", err)
	} else {
		info.CPUModel = cpuInfo[0].ModelName
	}

	return info
}
