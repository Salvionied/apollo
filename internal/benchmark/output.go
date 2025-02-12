package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type BenchmarkResult struct {
	WallClockTPS  float64       `json:"wall_clock_tps"`
	LatencyTPS    float64       `json:"latency_tps"`
	AvgLatency    time.Duration `json:"avg_latency"`
	Failures      int           `json:"failures"`
	Iterations    int           `json:"iterations"`
	Parallelism   int           `json:"parallelism"`
	UTXOCount     int           `json:"utxo_count"`
	SystemInfo    SystemInfo    `json:"system_info"`
	BenchDuration time.Duration `json:"bench_duration"`
}

func PrintResults(wallClockTPS, latencyTPS float64, avgLatency time.Duration,
	failures, iterations, parallelism, utxoCount int, benchDuration time.Duration,
	format string) {

	result := BenchmarkResult{
		WallClockTPS:  wallClockTPS,
		LatencyTPS:    latencyTPS,
		AvgLatency:    avgLatency,
		Failures:      failures,
		Iterations:    iterations,
		Parallelism:   parallelism,
		UTXOCount:     utxoCount,
		SystemInfo:    GetSystemInfo(),
		BenchDuration: benchDuration,
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			color.Red("Failed to encode JSON: %v", err)
			os.Exit(1)
		}
	default:
		printColorfulTable(result)
	}
}

func printColorfulTable(result BenchmarkResult) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Metric", "Value", "Description"})
	table.SetBorder(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor},
	)
	table.SetHeaderAlignment(10)
	table.SetColumnSeparator("│")
	table.SetRowSeparator("─")

	// Disable text wrapping
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetColWidth(1000) // Effectively disable column wrapping

	// Custom function for section headers
	addSectionHeader := func(title string) {
		// Add top border for the section
		table.Rich([]string{"", "", ""}, []tablewriter.Colors{
			{tablewriter.FgHiBlackColor, tablewriter.Bold},
			{tablewriter.FgHiBlackColor, tablewriter.Bold},
			{tablewriter.FgHiBlackColor, tablewriter.Bold},
		})
		// Add colored section title
		table.Rich([]string{
			color.HiMagentaString(title),
			"",
			"",
		}, []tablewriter.Colors{
			{tablewriter.FgHiMagentaColor, tablewriter.Bold},
			{},
			{},
		})
		// Add bottom border for the section header
		table.Rich([]string{"", "", ""}, []tablewriter.Colors{
			{tablewriter.FgHiBlackColor, tablewriter.Bold},
			{tablewriter.FgHiBlackColor, tablewriter.Bold},
			{tablewriter.FgHiBlackColor, tablewriter.Bold},
		})
	}

	// Throughput Metrics Section
	addSectionHeader("THROUGHPUT METRICS")
	addRow(table, "Wall-clock Tx/s", fmt.Sprintf("%.2f", result.WallClockTPS),
		color.HiGreenString("Actual transactions processed per second"))
	addRow(table, "Latency-based Tx/s", fmt.Sprintf("%.2f", result.LatencyTPS),
		color.HiYellowString("Theoretical maximum based on average latency"))
	addRow(table, "Avg Latency/Transaction", result.AvgLatency.Round(time.Microsecond).String(),
		color.HiWhiteString("Mean time to build and validate one transaction"))

	// Failure Analysis Section
	addSectionHeader("FAILURE ANALYSIS")
	failureColor := color.HiGreenString
	if result.Failures > 0 {
		failureColor = color.HiRedString
	}
	addRow(table, "Failed Transactions",
		fmt.Sprintf("%s (%d/%d)", failureColor(fmt.Sprintf("%d", result.Failures)), result.Failures, result.Iterations),
		color.HiWhiteString("Total failed transaction constructions"))

	// Configuration Section
	addSectionHeader("BENCHMARK CONFIGURATION")
	addRow(table, "Iterations", fmt.Sprintf("%d", result.Iterations), "")
	addRow(table, "Parallel Workers", fmt.Sprintf("%d", result.Parallelism), "")
	addRow(table, "Outputs per TX", fmt.Sprintf("%d", result.UTXOCount), "")
	addRow(table, "Total Duration", result.BenchDuration.Round(time.Millisecond).String(), "")

	// System Info Section
	addSectionHeader("SYSTEM INFORMATION")
	addRow(table, "CPU Model", result.SystemInfo.CPUModel, "")
	addRow(table, "Total Memory", fmt.Sprintf("%d GB", result.SystemInfo.TotalMemory/1e9), "")
	addRow(table, "Available Memory", fmt.Sprintf("%d GB", result.SystemInfo.AvailableMem/1e9), "")
	addRow(table, "Go Version", result.SystemInfo.GoVersion, "")
	addRow(table, "OS/Arch", fmt.Sprintf("%s/%s", result.SystemInfo.OS, runtime.GOARCH), "")

	// Efficiency Section
	addSectionHeader("EFFICIENCY ANALYSIS")
	addRow(table, "Throughput Efficiency",
		fmt.Sprintf("%s%.1f%%", color.HiCyanString(""), (result.WallClockTPS/result.LatencyTPS)*100),
		color.HiWhiteString("Ratio of actual vs theoretical maximum throughput"))

	table.Render()
}

func addRow(table *tablewriter.Table, metric, value, description string) {
	table.Rich([]string{metric, value, description}, []tablewriter.Colors{
		{tablewriter.FgHiCyanColor},
		{tablewriter.FgHiGreenColor},
		{tablewriter.FgHiWhiteColor},
	})
}
