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
	"github.com/Salvionied/apollo/internal/consts"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
)

type Result struct {
	Error error
}

func Run(utxoCount, iterations, parallelism int, backend string, outputFormat string, cpuProfile string) {
	// CPU profiling: check error when creating file.
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

	var wg sync.WaitGroup
	results := make(chan Result, iterations)

	// Record the overall start time.
	globalStart := time.Now()

	sem := make(chan struct{}, parallelism)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(iter int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in iteration %d: %v", iter, r)
					results <- Result{Error: fmt.Errorf("panic: %v", r)}
				}
				<-sem
				wg.Done()
			}()

			// Initialize a new instance of apollo.Apollo for each iteration.
			apolloBE := apollo.New(ctx).SetWalletFromBech32(walletAddress.String())

			apolloBE = apolloBE.
				AddLoadedUTxOs(userUtxos...).
				SetChangeAddress(walletAddress).
				AddRequiredSigner(serialization.PubKeyHash(walletAddress.PaymentPart)).
				SetTtl(int64(lastSlot) + 300)

			for j := 0; j < utxoCount; j++ {
				apolloBE = apolloBE.PayToAddress(walletAddress, 2_000_000)
			}

			_, err = apolloBE.Complete()
			if err != nil {
				results <- Result{Error: fmt.Errorf("tx build failed: %w", err)}
				return
			}
			results <- Result{}
		}(i)
	}

	wg.Wait()
	globalEnd := time.Now()
	close(results)

	// Calculate overall elapsed time.
	totalElapsed := globalEnd.Sub(globalStart)
	var failures int
	var successCount int

	for res := range results {
		if res.Error != nil {
			log.Printf("Error: %v", res.Error)
			failures++
		} else {
			successCount++
		}
	}

	if successCount == 0 {
		log.Fatal("All iterations failed! Check logs for errors.")
	}

	// Calculate transactions per second based on overall elapsed time.
	tps := float64(successCount) / totalElapsed.Seconds()

	PrintResults(tps, failures, outputFormat)
}
