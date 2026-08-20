# AI Agent Instructions

This file provides guidance for AI coding agents working with Apollo v2.

## Project Overview

Apollo v2 is a pure Go Cardano transaction-building library. It uses
Blink Labs packages directly for core ledger behavior:

- `github.com/blinklabs-io/gouroboros` for ledger types, CBOR, scripts,
  addresses, transactions, certificates, and governance types
- `github.com/blinklabs-io/bursa` for HD wallet key derivation
- `github.com/blinklabs-io/plutigo` for Plutus data encoding/decoding

Module path: `github.com/Salvionied/apollo/v2`

## Build and Test Commands

```bash
make mod-tidy    # go mod tidy
make format      # go fmt / gofmt
make test        # go test -v -race ./...
make clean       # remove temporary files
```

Single test:

```bash
go test -v -race -run TestName ./path/to/package
```

Linting:

```bash
golangci-lint run
```

## Architecture

### Entry Points

| File | Purpose |
|------|---------|
| `apollo.go` | Main transaction builder and fluent API |
| `models.go` | `Unit`, `Payment`, `PaymentI`, value conversion |
| `helpers.go` | Value, script, CBOR, and witness helpers |
| `wallet.go` | Wallet interfaces and Bursa wallet adapter |
| `convenience.go` | Bech32/script convenience wrappers |

### Package Structure

| Package | Purpose |
|---------|---------|
| `backend/` | ChainContext interface and shared backend helpers |
| `backend/blockfrost/` | Blockfrost backend |
| `backend/maestro/` | Maestro backend |
| `backend/ogmios/` | Ogmios/Kupo backend |
| `backend/utxorpc/` | UTxO RPC backend |
| `backend/fixed/` | Deterministic in-memory test backend |
| `plutusencoder/` | Struct-tag-driven Plutus data marshaling |
| `constants/` | Network constants |
| `internal/backendutil/` | Provider response parsing helpers (not public API) |

### Key Interfaces

**ChainContext** (`backend/base.go`):

- `ProtocolParams()` / `GenesisParams()` - protocol and genesis parameters
- `NetworkId()` / `CurrentEpoch()` / `Tip()` - network and chain position
- `MaxTxFee()` - maximum fee derived from protocol parameters
- `Utxos(address)` / `UtxoByRef(txHash, index)` - query UTxOs
- `SubmitTx(txCbor)` - submit a transaction
- `EvaluateTx(txCbor, additionalUtxos)` - evaluate Plutus scripts
- `ScriptCbor(scriptHash)` - resolve reference-script bytes

Backends may implement `CapabilityReporter`; callers must check
`CapabilityEvaluateTxAdditionalUtxos` before relying on evaluator-supplied
UTxOs. `backend/cache` provides the cache wrapper.

**Wallet** (`wallet.go`):

- `Address()`
- `SignTxBody(txHash)`
- `PubKeyHash()`
- `StakePubKeyHash()`

## Coding Standards

### Error Handling

- Return errors, never panic in library code.
- Never silently ignore errors.
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`.
- Fluent builder methods that cannot return an error should call
  `setErrOnce`; `Complete()` returns the stored error.

### Naming

- Use camelCase for variables.
- Exported types use PascalCase.
- Package names are lowercase and single word where possible.

### Testing

- New exported behavior requires tests.
- Use `backend/fixed.FixedChainContext` for deterministic chain-context tests.
- Run `make test` before committing when feasible.

### CBOR and Ledger Types

- Prefer gouroboros ledger/common/conway types over local duplicates.
- Implement local CBOR only when gouroboros does not already provide the type.
- Ensure deterministic encoding for maps and sets where Cardano requires it.
- Test roundtrip behavior for custom encoders.

## Plutus Data Struct Tags

```go
type Datum struct {
    _      struct{} `plutusType:"IndefList" plutusConstr:"1"`
    Pkh    []byte   `plutusType:"Bytes"`
    Amount int64    `plutusType:"Int"`
}
```

Options include `Bytes`, `Int`, `BigInt`, `Map`, `IndefList`, `DefList`,
`StringBytes`, `HexString`, `Bool`, and `IndefBool`.

## Common Tasks

### Adding a Backend Method

1. Add it to `backend.ChainContext` in `backend/base.go`.
2. Implement it in every backend package.
3. Add deterministic tests using `backend/fixed`.

### Adding a Transaction Builder Method

1. Add the method to `Apollo` in `apollo.go`.
2. Follow the fluent API pattern.
3. Add tests in the relevant `*_test.go` file.

### Adding Metadata or Governance Support

Prefer `common.TransactionMetadatum`, `common.VotingProcedures`, and
`conway.ConwayProposalProcedure` from gouroboros. Do not recreate the removed
v1 `serialization/*` package tree.

## Go Version

Requires Go 1.25.13 or newer, matching the `go` directive in `go.mod`. That
directive is a hard floor at patch granularity: a toolchain older than it
refuses to build the module rather than warning. Keep the directive, this
line, and `README.md` in step with each other. Raising the directive locks
out every user on an older patch, so raise it only when something in the
tree actually needs the newer toolchain.
