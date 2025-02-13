# Apollo-Bench: Cardano Transaction Benchmark Tool

Apollo-Bench is a benchmarking tool for the Apollo Cardano transaction–building library written in Golang. It is designed to stress–test key parts of the library, namely:

- **UTXO Selection:** Simulates the process of selecting UTXOs from a wallet.
- **CBOR Serialization/Deserialization:** Measures the performance of building and serializing transactions.

The tool runs multiple iterations concurrently and computes performance metrics such as transactions per second (TPS), average latency per transaction, and theoretical throughput based on latency. Results can be output as a pretty table or in JSON format.

---

## Features

- **Throughput Metrics:**  
  - **Wall-clock Tx/s:** Actual transactions built per second (calculated as the number of successful iterations divided by the total elapsed time).
  - **Latency-based Tx/s:** Theoretical maximum based on average transaction latency.
  - **Average Latency:** Mean time to build and serialize a transaction.
  
- **Failure Analysis:** Reports any failed transaction builds.
- **Configurable Benchmarking:**  
  - Specify number of iterations, UTXO count, and parallel workers.
  - Choose among different backend chain indexers (e.g., Maestro, Blockfrost, Ogmios).
- **System Information:** Displays CPU model, total and available memory, Go version, and OS/Arch.
- **Optional CPU Profiling:** Write a CPU profile to a file for further performance analysis.

---

## Requirements

- **Go:** Version 1.18 or higher is recommended.
- **Dependencies:**  
  - [Cobra](https://github.com/spf13/cobra) – For CLI flag parsing.
  - [Tablewriter](https://github.com/olekukonko/tablewriter) – For formatted table output.
  - [Fatih/color](https://github.com/fatih/color) – For colored terminal output.
  - [gopsutil](https://github.com/shirou/gopsutil) – For gathering system metrics.
- **Apollo Library:** Clone the [Apollo repository](https://github.com/Salvionied/apollo) as this tool is part of that project.

---

## Installation

Clone the Apollo repository and build the benchmark binary:

```bash
git clone https://github.com/Salvionied/apollo.git
go build -o ./bin/apollo-bench ./cmd/benchmark
```

---

## Configuration

Apollo-Bench relies on several environment constants. You can change these values to test other networks, wallet addresses, or smart contract addresses:

```go
NETWORK             string
BFC_NETWORK_ID      int
MAESTRO_NETWORK_ID  int
BFC_API_URL         string
BFC_API_KEY         string
MAESTRO_API_KEY     string
OGMIGO_ENDPOINT     string
KUGO_ENDPOINT       string
TEST_WALLET_ADDRESS string 
```

---

## Usage

Run the benchmark binary:

```bash
./apollo-bench [flags]
```

### Available Flags

- `--utxo-count`, `-u` (default: **10**)  
  *Number of UTXOs to test.* This simulates the number of UTXO outputs to include in each transaction.

- `--iterations`, `-i` (default: **1000**)  
  *Number of transactions to build.* This defines the total number of iterations for the benchmark run.

- `--parallelism`, `-p` (default: **4**)  
  *Number of parallel goroutines.* Controls the concurrency level during the benchmark.

- `--output`, `-o` (default: **"table"**)  
  *Output format for results.* Options:
  - `table`: Displays a formatted, colorful table.
  - `json`: Outputs results as formatted JSON.

- `--backend`, `-b` (default: **"maestro"**)  
  *Selects the backend chain indexer.* Supported options include:
  - `maestro`
  - `blockfrost`
  - `ogmios`

- `--cpu-profile`, `-c` (default: **""**)  
  *Writes CPU profiling data to the specified file.*  
  Example:

  ```bash
  ./apollo-bench -c cpu.prof
  ```

  You can then analyze the profile with:

  ```bash
  go tool pprof cpu.prof
  ```

---

## Examples

### Basic Test Run

```bash
./apollo-bench --utxo-count 100 --iterations 5000 --parallelism 8
```

### Blockfrost Backend Test

```bash
./apollo-bench -b blockfrost -u 50 -i 10000 -p 12
```

### JSON Output with Profiling

```bash
./apollo-bench -o json -c profile.out
```

---

## Benchmark Metrics & How It Works

### Key Metrics

1. **Wall-clock Tx/s**  
   - **Formula:**  

    ```plaintext
    TPS = Number of Successful Iterations / Total Elapsed Time (seconds)
    ```

   - Represents the actual throughput achieved during the benchmark.

2. **Latency-based Tx/s**  
   - **Formula:**  

     ```plaintext
     Latency-based TPS = 1 second / Average Latency per Transaction
     ```

   - Represents the theoretical maximum throughput based on average transaction latency.

3. **Average Latency**  
   - **Formula:**  

     ```plaintext
     Average Latency = Total Latency / Number of Successful Iterations
     ```

   - Measures the mean time to build and serialize a transaction.

4. **Efficiency Ratio**  
   - **Formula:**  

     ```plaintext
     Efficiency = (Wall-clock TPS / Latency-based TPS) * 100
     ```

   - Indicates how close the actual throughput is to the theoretical maximum.

### Benchmark Workflow

1. **Setup:**
   - Initialize the chain context based on the selected backend.
   - Decode the test wallet address and fetch UTXOs.
   - Run a warm-up phase (GC + 2-second sleep).

2. **Transaction Building:**
   - For each iteration:
     - Clone UTXOs for thread safety.
     - Build and serialize the transaction.
     - Record latency and track failures.

3. **Results Calculation:**
   - Compute wall-clock TPS, latency-based TPS, and average latency.
   - Generate system diagnostics and efficiency analysis.

4. **Output:**
   - Print results as a colorful table or JSON.

---

## Example Output

### Table Format

```plaintext
+─────────────────────────+─────────────────────────────────────────+───────────────────────────────────────────────────+
│         Metric          │                  Value                  │                    Description                    │
+─────────────────────────+─────────────────────────────────────────+───────────────────────────────────────────────────+
│                         │                                         │                                                   │
│ THROUGHPUT METRICS      │                                         │                                                   │
│                         │                                         │                                                   │
│ Wall-clock Tx/s         │ 2695.40                                 │ Actual transactions processed per second          │
│ Latency-based Tx/s      │ 272.75                                  │ Theoretical maximum based on average latency      │
│ Avg Latency/Transaction │ 3.668ms                                 │ Mean time to build and validate one transaction   │
│                         │                                         │                                                   │
│ FAILURE ANALYSIS        │                                         │                                                   │
│                         │                                         │                                                   │
│ Failed Transactions     │ 0 (0/1000000)                           │ Total failed transaction constructions            │
│                         │                                         │                                                   │
│ BENCHMARK CONFIGURATION │                                         │                                                   │
│                         │                                         │                                                   │
│ Iterations              │ 1000000                                 │                                                   │
│ Parallel Workers        │ 10                                      │                                                   │
│ Outputs per TX          │ 100                                     │                                                   │
│ Total Duration          │ 6m11.024s                               │                                                   │
│                         │                                         │                                                   │
│ SYSTEM INFORMATION      │                                         │                                                   │
│                         │                                         │                                                   │
│ CPU Model               │ Intel(R) Core(TM) i7-6700 CPU @ 3.40GHz │                                                   │
│ Total Memory            │ 33 GB                                   │                                                   │
│ Available Memory        │ 28 GB                                   │                                                   │
│ Go Version              │ go1.23.3                                │                                                   │
│ OS/Arch                 │ linux/amd64                             │                                                   │
│                         │                                         │                                                   │
│ EFFICIENCY ANALYSIS     │                                         │                                                   │
│                         │                                         │                                                   │
│ Throughput Efficiency   │ 988.6%                                  │ Ratio of actual vs theoretical maximum throughput │
+─────────────────────────+─────────────────────────────────────────+───────────────────────────────────────────────────+

```

### JSON Format

```json
{
  "wall_clock_tps": 2695.3914485839823,
  "latency_tps": 272.749861374883,
  "avg_latency": 3666363,
  "failures": 0,
  "iterations": 1000000,
  "parallelism": 10,
  "utxo_count": 100,
  "system_info": {
    "GoVersion": "go1.23.3",
    "OS": "linux",
    "CPUModel": "Intel(R) Core(TM) i7-6700 CPU @ 3.40GHz",
    "TotalMemory": 33419149312,
    "AvailableMem": 28558069760
  },
  "bench_duration": 370866033018
}
```

---

## Benchmarking Script

The `apollo/scripts/compare_versions.sh` script is used to benchmark two different versions/tag/commit-hash of the Apollo library and store the results in JSON format in the `results/` directory. The script:

1. Defines two specific versions to compare.
2. Uses Git worktrees to check out each version in a temporary directory.
3. Builds the benchmark binary for each version and stores it in `bin/`.
4. Runs the benchmark and stores the output as JSON files in `results/`.
5. Cleans up temporary files after execution.

Users can modify the script to benchmark different versions, adjust benchmark parameters, or integrate additional analysis tools.

---

Happy benchmarking!
