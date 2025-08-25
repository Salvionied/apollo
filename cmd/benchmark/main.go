package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Salvionied/apollo/internal/benchmark"
	"github.com/spf13/cobra"
)

func main() {
	var (
		utxoCount    int
		iterations   int
		parallelism  int
		backend      string
		outputFormat string
		cpuProfile   string
	)

	cmd := &cobra.Command{
		Use: "apollo-bench",
		Run: func(cmd *cobra.Command, args []string) {
			benchmark.Run(utxoCount, iterations, parallelism, backend, outputFormat, cpuProfile)
		},
	}

	cmd.Flags().IntVarP(&utxoCount, "utxo-count", "u", 10, "Number of UTXOs to test")
	cmd.Flags().IntVarP(&iterations, "iterations", "i", 1000, "Number of transactions to build")
	cmd.Flags().IntVarP(&parallelism, "parallelism", "p", 4, "Number of parallel goroutines")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table/json)")
	cmd.Flags().StringVarP(&backend, "backend", "b", "maestro", "Backend Chain Indexer")
	cmd.Flags().StringVarP(&cpuProfile, "cpu-profile", "c", "", "Write CPU profile to file")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if utxoCount <= 0 {
			return errors.New("--utxo-count must be > 0")
		}
		if iterations <= 0 {
			return errors.New("--iterations must be > 0")
		}
		return nil
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
