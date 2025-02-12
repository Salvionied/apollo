package benchmark

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/internal/consts"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
)

type Result struct {
	Duration time.Duration
	Error    error
}

func Run(utxoCount, iterations, parallelism int, backend string, outputFormat string, cpuProfile string) {
	if cpuProfile != "" {
		f, _ := os.Create(cpuProfile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ctx, err := GetChainContext(backend)
	if err != nil {
		log.Fatalf("Critical error getting backend chain context: %v", err)
	}

	walletAddress, err := Address.DecodeAddress(consts.TEST_WALLET_ADDRESS)
	if err != nil {
		log.Fatalf("Error decoding wallet address: %v", err)
	}

	userUtxos, err := ctx.Utxos(walletAddress)
	if err != nil {
		log.Fatalf("failed to get UTXOs: %v", err)
	}

	lastSlot, err := ctx.LastBlockSlot()
	if err != nil {
		log.Fatalf("failed to get last block slot: %v", err)
	}

	// Warm-up phase.
	runtime.GC()
	time.Sleep(2 * time.Second)

	// Run benchmark with parallelism.
	start := time.Now()
	results := make(chan Result, iterations)
	sem := make(chan struct{}, parallelism)

	for i := 0; i < iterations; i++ {
		sem <- struct{}{}
		go func(iter int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in iteration %d: %v", iter, r)
					results <- Result{Error: fmt.Errorf("panic: %v", r)}
				}
				<-sem
			}()

			// Initialize a new instance of apollo.Apollo for each iteration
			apolloBE := apollo.New(ctx).SetWalletFromBech32(walletAddress.String())

			apolloBE = apolloBE.
				AddLoadedUTxOs(userUtxos...).
				SetChangeAddress(walletAddress).
				AddRequiredSigner(serialization.PubKeyHash(walletAddress.PaymentPart)).
				SetTtl(int64(lastSlot) + 300)

			for i := 0; i < utxoCount; i++ {
				apolloBE = apolloBE.PayToAddress(walletAddress, 2_000_000)
			}

			_, err = buildAndSerialize(apolloBE)
			if err != nil {
				results <- Result{Error: fmt.Errorf("tx build failed: %w", err)}
				return
			}

			results <- Result{Duration: time.Since(start)}
		}(i)
	}

	// Collect results.
	var (
		totalDuration time.Duration
		failures      int
		successCount  int
	)

	for i := 0; i < iterations; i++ {
		res := <-results
		if res.Error != nil {
			log.Printf("Error: %v", res.Error)
			failures++
		} else {
			totalDuration += res.Duration
			successCount++
		}
	}

	if successCount == 0 {
		log.Fatal("All iterations failed! Check logs for errors.")
	}

	// Calculate Tx/s and print results.
	avgDuration := totalDuration / time.Duration(successCount)
	tps := float64(successCount) / avgDuration.Seconds()
	PrintResults(tps, failures, outputFormat)
}

func buildAndSerialize(apolloBE *apollo.Apollo) (*apollo.Apollo, error) {
	apolloBE, err := apolloBE.Complete()
	if err != nil {
		return nil, err
	}

	return apolloBE, nil
}
