#!/bin/bash
set -e # Exit on error

# Determine directories (assuming script is in apollo/scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)" # Root is 'apollo'
RESULTS_DIR="$SCRIPT_DIR/results"
BIN_DIR="$ROOT_DIR/bin"

# Ensure the results and bin directories exist
mkdir -p "$RESULTS_DIR" "$BIN_DIR"

# Check if at least one version is passed as parameter
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 version1 [version2 version3 ...]"
    exit 1
fi

# The versions array is created from all command-line arguments
VERSIONS=("$@")

# Color variables
CYAN="\033[1;36m"
GREEN="\033[1;32m"
RED="\033[1;31m"
RESET="\033[0m"

for version in "${VERSIONS[@]}"; do
    # Create a temporary worktree directory for the specific version
    WORKTREE_DIR=$(mktemp -d)

    # Add the worktree for the specific version (detached checkout)
    if ! git worktree add "$WORKTREE_DIR" "$version" &>/dev/null; then
        echo -e "${RED}Error: Failed to check out $version${RESET}"
        exit 1
    fi

    printf "${CYAN}[%s] Building binary for version: %s...${RESET}\n" "$(date '+%Y-%m-%d %H:%M:%S')" "$version"
    pushd "$WORKTREE_DIR" >/dev/null
    if ! go build -o "$BIN_DIR/apollo-bench-$version" ./cmd/benchmark; then
        echo -e "${RED}Error: Build failed for $version${RESET}"
        popd >/dev/null
        exit 1
    fi
    popd >/dev/null

    printf "${CYAN}[%s] Benchmarking version: %s...${RESET}\n" "$(date '+%Y-%m-%d %H:%M:%S')" "$version"
    if ! "$BIN_DIR/apollo-bench-$version" --utxo-count 100 --iterations 10000 --parallelism 10 --backend maestro --output json >"$RESULTS_DIR/$version.json"; then
        echo -e "${RED}Error: Benchmark failed for $version${RESET}"
    else
        echo -e "${GREEN}Benchmark successful for $version! Output stored in $RESULTS_DIR/$version.json${RESET}"
    fi

    # Cleanup the temporary worktree
    rm -rf "$WORKTREE_DIR"
done
