#!/bin/bash
set -e # Exit on error

VERSIONS=("cebeb95a1c3ebe9d590d87202128efe47565ddc3" "v1.3.0")

for version in "${VERSIONS[@]}"; do
    if ! git checkout "$version" &>/dev/null; then
        echo "Error: Failed to checkout $version"
        exit 1
    fi

    go build -o "bin/apollo-bench-$version" ./cmd/benchmark || {
        echo "Error: Build failed for $version"
        exit 1
    }

    echo "Benchmarking $version..."
    "./bin/apollo-bench-$version" --utxo-count 100 --iterations 1000 --output json >"results/$version.json" || {
        echo "Error: Benchmark failed for $version"
    }
done
