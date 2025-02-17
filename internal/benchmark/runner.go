package benchmark

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
)

type Result struct {
	Duration time.Duration
	Error    error
}

func Run(utxoCount, iterations, parallelism int, backend string, outputFormat string, cpuProfile string) {

	ctx, err := GetChainContext(backend)
	if err != nil {
		log.Fatalf("Critical error getting backend chain context: %v", err)
	}

	walletAddress, err := Address.DecodeAddress(TEST_WALLET_ADDRESS)
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

	// Warm-up phase before any measurements
	runtime.GC()
	time.Sleep(2 * time.Second)

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatalf("failed to create cpu profile file: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	var (
		wg           sync.WaitGroup
		results      = make(chan Result, iterations)
		totalLatency time.Duration
		mu           sync.Mutex
	)

	sem := make(chan struct{}, parallelism)

	// Actual benchmark start time
	benchStart := time.Now()

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(iter int) {
			defer func() {
				<-sem
				wg.Done()
				if r := recover(); r != nil {
					mu.Lock()
					results <- Result{
						Error: fmt.Errorf("panic in iteration %d: %v", iter, r),
					}
					mu.Unlock()
				}
			}()

			// For thread safety
			clonedUTxOs := make([]UTxO.UTxO, len(userUtxos))
			copy(clonedUTxOs, userUtxos)

			start := time.Now()
			err := buildTransaction(clonedUTxOs, &walletAddress, ctx, lastSlot, utxoCount)
			elapsed := time.Since(start)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results <- Result{Error: fmt.Errorf("iteration %d: %w", iter, err)}
			} else {
				results <- Result{Duration: elapsed}
				totalLatency += elapsed
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Calculate metrics
	benchDuration := time.Since(benchStart)
	var failures int
	successes := 0

	for res := range results {
		if res.Error != nil {
			log.Printf("Error: %v", res.Error)
			failures++
		} else {
			successes++
		}
	}

	if successes == 0 {
		log.Fatal("All iterations failed! Check logs for errors.")
	}

	// Calculate accurate Tx/s metrics
	actualTxPerSec := float64(successes) / benchDuration.Seconds()
	latencyPerTx := totalLatency / time.Duration(successes)

	// For comparison: latency-based Tx/s
	latencyTxPerSec := float64(time.Second) / float64(latencyPerTx)
	PrintResults(
		actualTxPerSec,
		latencyTxPerSec,
		latencyPerTx,
		failures,
		iterations,
		parallelism,
		utxoCount,
		benchDuration,
		outputFormat,
	)
}

func buildTransaction(utxos []UTxO.UTxO, addr *Address.Address, ctx Base.ChainContext, lastSlot int, utxoCount int) error {
	apolloBE := apollo.New(ctx).
		SetWalletFromBech32(addr.String()).
		AddLoadedUTxOs(utxos...).
		SetChangeAddress(*addr).
		AddRequiredSigner(serialization.PubKeyHash(addr.PaymentPart)).
		SetTtl(int64(lastSlot) + 300)

	// Add multiple outputs
	for j := 0; j < utxoCount; j++ {
		apolloBE = apolloBE.PayToAddress(*addr, 2_000_000)
	}

	_, err := apolloBE.Complete()
	return err
}
