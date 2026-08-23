<div align="center">
    <img src="./assets/logo.jpg" alt="apollo logo" width="480">
</div>

# Apollo v2

**Apollo** is a pure Go library for constructing Cardano transactions. It uses
the Blink Labs ledger packages for types, CBOR, scripts, addresses, and
transaction bodies — [gouroboros](https://github.com/blinklabs-io/gouroboros)
for the ledger, [bursa](https://github.com/blinklabs-io/bursa) for HD wallets,
and [plutigo](https://github.com/blinklabs-io/plutigo) (or Apollo's own
[`plutusencoder`](docs/plutusencoder/README.md) struct tags) for Plutus data.

Ready to learn? Start with the [documentation](docs/README.md) or jump to the
[first transaction](docs/getting_started/README.md).

---

[![Go Reference](https://pkg.go.dev/badge/github.com/Salvionied/apollo/v2.svg)](https://pkg.go.dev/github.com/Salvionied/apollo/v2)
[![GitHub tag](https://img.shields.io/github/v/tag/Salvionied/apollo?include_prereleases&sort=semver&label=tag)](https://github.com/Salvionied/apollo/tags)
[![Build Status](https://github.com/Salvionied/apollo/actions/workflows/go-test.yml/badge.svg)](https://github.com/Salvionied/apollo/actions/workflows/go-test.yml)
[![Lint Status](https://github.com/Salvionied/apollo/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/Salvionied/apollo/actions/workflows/golangci-lint.yml)
[![CodeQL](https://github.com/Salvionied/apollo/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Salvionied/apollo/actions/workflows/codeql-analysis.yml)
[![TODOs](https://img.shields.io/github/search/Salvionied/apollo/TODO)](https://github.com/search?q=repo%3ASalvionied%2Fapollo+TODO&type=code)
[![Go Report Card](https://goreportcard.com/badge/github.com/Salvionied/apollo)](https://goreportcard.com/report/github.com/Salvionied/apollo)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Salvionied/apollo)](go.mod)
[![License](https://img.shields.io/github/license/Salvionied/apollo)](LICENSE)

## Install

Apollo v2 requires Go 1.25.13 or newer — the `go` directive in `go.mod` is a
hard floor, so an older 1.25 patch release fails to build:

```bash
go get github.com/Salvionied/apollo/v2
```

Upgrading from v1? See the [migration guide](docs/v2_migration/MIGRATION.md).

## Features

- Address parsing and validation (canonical bech32; no silent base58 fallback)
- HD wallets via bursa (mnemonic, optional passphrase, watch-only)
- Fluent Conway transaction builder (`Complete` → `Sign` → `Submit`)
- Automatic coin selection (MACS by default, pluggable)
- Chain backends: Blockfrost, Ogmios/Kupo, UTxO RPC, plus `fixed` and cache
- Plutus V1–V3 scripts, datums, redeemers, and reference inputs
- Struct-tag Plutus encoding (`plutusencoder`)
- Staking certificates, withdrawals, and CIP-1694 governance
- Race-tested suite (`go test -race ./...`)

## Coin Selection

The builder balances transactions automatically using a pluggable
`CoinSelector`. A selector receives the available UTxO pool and the target
value and returns a deterministic subset that covers it.

Two selectors ship in-tree:

- **MACS (the default)**: Multi-Asset Coin Selection
  ([IEEE Blockchain 2023](https://doi.org/10.1109/Blockchain60715.2023.00029)).
  It covers each deficient asset class directly, prefers UTxOs near the pool
  average, sweeps a bounded amount of dust, and avoids change in the min-UTxO
  dead band. On multi-asset payments it is substantially cheaper than
  largest-first; on plain ADA it is within a fraction of a percent while
  leaving far less dust.
- **Largest-First**: the v1 greedy behaviour — ADA-only UTxOs first, largest
  lovelace first. Use it when you need the old input set.

```go
builder := apollo.New(chainContext) // MACS with dust sweeping

builder = builder.SetCoinSelector(&apollo.LargestFirstSelector{})

builder = builder.SetCoinSelector(&apollo.MACSSelector{
    DustThreshold: 2_000_000,
    MaxDustInputs: 4,
})
```

Details, benchmarks, and the design note: [coin selection](docs/coin_selection/README.md).

## Chain Backends and Evaluation

Apollo does not embed a Plutus CEK machine. Script execution units come from
the chain backend's `EvaluateTx`. Backends implement `ChainContext` and may
report optional operations through `CapabilityReporter` — check
`CapabilityEvaluateTxAdditionalUtxos` before relying on evaluator-supplied
UTxOs for transaction chaining.

| Backend | Package |
|---------|---------|
| Blockfrost | `backend/blockfrost` |
| Ogmios + Kupo | `backend/ogmios` |
| UTxO RPC | `backend/utxorpc` |
| In-memory tests | `backend/fixed` |
| TTL cache wrapper | `backend/cache` |

See [backends](docs/backends/README.md) and
[capabilities](docs/backends/capabilities.md).

## Evaluation Witnesses

When a script transaction lists required signers, the evaluator needs valid
signatures on the *preliminary* body. Apollo supplies those as
evaluation-only witnesses: they are not kept in the unsigned transaction
`Complete()` returns.

`BursaWallet` provides its payment and stake witnesses automatically.
Watch-only, hardware, and remote wallets register an
`EvaluationWitnessProvider` instead of changing the `Wallet` interface:

```go
type remoteEvaluationSigner struct{}

func (remoteEvaluationSigner) EvaluationWitnesses(
    bodyHash common.Blake2b256,
    required []common.Blake2b224,
) ([]common.VkeyWitness, error) {
    // Return valid witnesses for any requested hashes controlled remotely.
    return nil, nil
}

builder.AddEvaluationWitnessProvider(remoteEvaluationSigner{})
```

Full behaviour: [evaluation witnesses](docs/wallets/evaluation_witnesses.md).

## Basic Example

This sends 1 ADA to a receiver on mainnet via Blockfrost. The full walkthrough
is in [getting started](docs/getting_started/README.md).

```go
package main

import (
    "encoding/hex"
    "fmt"

    "github.com/blinklabs-io/gouroboros/ledger/common"

    apollo "github.com/Salvionied/apollo/v2"
    "github.com/Salvionied/apollo/v2/backend/blockfrost"
    "github.com/Salvionied/apollo/v2/constants"
)

func main() {
    chain := blockfrost.NewBlockFrostChainContext(
        constants.BlockfrostBaseUrlMainnet,
        1, // mainnet network ID
        "your_blockfrost_project_id",
    )

    // 1.- Build transaction
    builder, err := apollo.New(chain).SetWalletFromMnemonic("your mnemonic here")
    if err != nil {
        panic(err)
    }

    utxos, err := chain.Utxos(builder.GetWallet().Address())
    if err != nil {
        panic(err)
    }

    receiver, err := common.NewAddress("addr1...")
    if err != nil {
        panic(err)
    }

    builder, err = builder.
        AddLoadedUTxOs(utxos...).
        PayToAddress(receiver, 1_000_000).
        Complete()
    if err != nil {
        panic(err)
    }

    // 2.- Sign transaction
    builder, err = builder.Sign()
    if err != nil {
        panic(err)
    }

    // 3.- Submit transaction
    txId, err := builder.Submit()
    if err != nil {
        panic(err)
    }
    fmt.Println(hex.EncodeToString(txId.Bytes()))
}
```

## Conway Era Support

Apollo builds Conway-era transactions, including CIP-1694 governance. You can:

- Register, update, and retire DReps
- Authorize or resign constitutional committee keys
- Cast votes and submit governance action proposals
- Donate to the treasury
- Register stake, delegate to pools and DReps, and withdraw rewards

See [Conway governance](docs/conway_governance/README.md) and
[staking](docs/staking_functionalities/README.md). Plutus scripts, datums, and
reference inputs are covered under [Plutus V3](docs/plutus_v3_support/README.md)
and [data attachment](docs/data_attachment/README.md).

## Contributing

We welcome contributions. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for
guidelines. Before opening a pull request, run:

```bash
gofmt -w $(find . -name '*.go' -not -path './.worktrees/*')
go test -race ./...
```

## License

[MIT](LICENSE)

For questions and requests, join the
[Apollo Discord](https://discord.gg/MH4CmJcg49).

Created by Edoardo Salvioni (Zhaata).
