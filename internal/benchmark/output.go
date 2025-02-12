package benchmark

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func PrintResults(tps float64, failures int, format string) {
	sysInfo := GetSystemInfo()
	result := struct {
		TxPerSecond float64    `json:"tx_per_second"`
		Failures    int        `json:"failures"`
		SystemInfo  SystemInfo `json:"system_info"`
	}{
		TxPerSecond: tps,
		Failures:    failures,
		SystemInfo:  sysInfo,
	}

	switch format {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
	default:
		fmt.Printf("Transactions per second: %.2f\n", tps)
		fmt.Printf("Failures: %d\n", failures)
		fmt.Printf("CPU: %s\n", sysInfo.CPUModel)
		fmt.Printf("Memory: %d MB\n", sysInfo.TotalMemory/1024/1024)
	}
}
