# AI Agent Instructions

This file provides guidance for AI coding agents and automated review bots
working with this repository. It is the authoritative source for conventions,
build commands, and architectural context when opening PRs against Apollo.

## Project Overview

Apollo is a pure Go library for Cardano blockchain development. It provides
CBOR serialization of Cardano ledger types, a fluent transaction builder,
wallet/key management, multi-backend chain context support, and Plutus
(V1/V2/V3) script integration.

Module path: `github.com/Salvionied/apollo`
Go version: **1.24+** (see `go.mod`)

## Build, Format, and Test Commands

```bash
make mod-tidy    # go mod tidy — fetch deps and update go.mod/go.sum
make format      # go fmt ./... and gofmt -s -w on all .go files
make golines     # golines -w --chain-split-dots --max-len=80 --reformat-tags .
make test        # go mod tidy + go test -v -race ./...
make clean       # remove files under tmp/
```

Run a single test or package:

```bash
go test -v -race ./serialization/Address/...
go test -v -race -run TestName ./path/to/package
```

Coverage report:

```bash
./test.sh        # produces coverage.out and coverage.html
```

Linting (CI runs this via `.github/workflows/golangci-lint.yml`):

```bash
golangci-lint run
```

**Before opening a PR, always run:** `make format && make golines && make test`
plus `golangci-lint run`. Review bots should flag PRs that fail any of these.

## Coding Standards

### Formatting

- **Max line length: 80 characters.** Enforced by `golines` with
  `--chain-split-dots` and `--reformat-tags`. Long method chains must be
  split at the dots, and struct tags must be re-aligned.
- `gofmt -s` (simplified) is required — the `format` target applies it.
- Do not hand-format struct tags; let `golines --reformat-tags` do it.

### Linting

`.golangci.yml` enables these linters on top of the defaults:

- `bodyclose` — HTTP response bodies must be closed.
- `fatcontext` — no context nested in loops.
- `perfsprint` — use `strconv`/string concat instead of `fmt.Sprintf` where
  equivalent.
- `prealloc` — preallocate slices where length is known.

`noctx` is explicitly disabled. Tests are excluded from linting
(`run.tests: false`), but test code should still compile cleanly and pass
`go vet`.

### Error Handling

- Return errors; never `panic` in library code.
- Never silently swallow errors — at minimum, wrap with context:
  `fmt.Errorf("operation failed: %w", err)`.
- Prefer `errors.Is` / `errors.As` over string matching.

### Naming

- Exported identifiers use `PascalCase`; unexported use `camelCase`.
- Package names are lowercase, single word, no underscores.
- Directory names under `serialization/` use `PascalCase` to match the type
  they implement (e.g., `serialization/Address/`, `serialization/MultiAsset/`).

### Testing

- New exported behavior requires tests.
- Use the deterministic `FixedChainContext` backend for unit tests that need
  a chain context.
- Reusable fixtures live in `testUtils/testUtils.go` —
  `InitUtxos()`, `InitUtxosDifferentiated()`, `InitUtxosCongested()` generate
  mock UTxO sets.
- Run `make test` (race detector enabled) before committing.

### CBOR Serialization

- Apollo wraps `github.com/fxamacker/cbor/v2` via the internal
  `serialization/cbor` package (`apolloCbor`). Use that wrapper rather than
  importing `fxamacker/cbor` directly from new code so tag handling
  (e.g. `CborTagSet = 258`) stays consistent.
- Implement `MarshalCBOR` / `UnmarshalCBOR` for new custom types.
- Ensure deterministic encoding — sort map keys where Cardano requires it.
- Test round-trip: `marshal -> unmarshal -> deep-equal`.

## Architecture

### Entry Points

| File                | Purpose                                        |
|---------------------|------------------------------------------------|
| `ApolloBuilder.go`  | Main transaction builder (fluent API)          |
| `backends.go`       | Factory functions for chain contexts           |
| `Models.go`         | Core value types: `Unit`, `Payment`, `PaymentI`|
| `Utils.go`          | Shared helpers                                 |

Typical usage:

```go
apollo.New(chainContext).
    AddInput(utxos...).
    PayToAddressBech32("addr...", 1_000_000).
    Complete().
    Sign()
```

### Package Layout

| Package                    | Purpose                                        |
|----------------------------|------------------------------------------------|
| `serialization/`           | CBOR-serializable Cardano ledger types         |
| `serialization/cbor/`      | Internal CBOR wrapper + Cardano tag constants  |
| `txBuilding/Backend/Base/` | `ChainContext` interface definition            |
| `txBuilding/Backend/*`     | Backend implementations (see below)            |
| `txBuilding/CoinSelection/`| UTxO selection algorithms                      |
| `txBuilding/Errors/`       | Custom error types                             |
| `crypto/bech32/`           | Address encoding                               |
| `crypto/bip32/`            | HD wallets (xprv/xpub)                         |
| `crypto/ed25519/`          | EdDSA signing                                  |
| `apollotypes/`             | `Wallet` interface and Plutus-related types    |
| `plutusencoder/`           | Struct-tag-driven Plutus data marshaling       |
| `constants/`               | Network and URL constants                      |
| `testUtils/`               | Test fixtures (`InitUtxos*`)                   |

### Chain Backends

All implement `txBuilding/Backend/Base/Base.go` `ChainContext`:

- `BlockFrostChainContext/` — BlockFrost REST API
- `OgmiosChainContext/`     — Ogmios JSON-RPC
- `MaestroChainContext/`    — Maestro REST API
- `UtxorpcChainContext/`    — UTxO RPC (gRPC)
- `FixedChainContext/`      — Deterministic in-memory backend for tests
- `Cache/`                  — Caching layer shared across backends

### Key Interfaces

**`ChainContext`** (`txBuilding/Backend/Base/Base.go:224`):

- `GetProtocolParams()` — protocol parameters
- `Utxos(address)`      — query UTxOs
- `SubmitTx(tx)`         — submit a transaction
- `EvaluateTx(tx)`       — evaluate Plutus scripts / execution units

**`Wallet`** (`apollotypes/types.go:18`):

- `GetAddress()`, `SignTx()`, `PkeyHash()`, `SkeyHash()`

### Network Constants

Defined in `constants/constants.go`:

- `MAINNET`, `TESTNET`, `PREVIEW`, `PREPROD` (enum type `Network`)
- BlockFrost base URLs for each network

## Plutus

### Plutus V3

End-to-end documentation lives under `docs/plutus_v3_support/`:

- `transaction_building.md`
- `script_management.md`
- `reference_inputs.md`
- `cost_models.md`
- `data_structures.md`

### Plutus Encoder Struct Tags

See `plutusencoder/readme.md` for the full tag reference.

```go
type Datum struct {
    _      struct{} `plutusType:"IndefList" plutusConstr:"1"`
    Pkh    []byte   `plutusType:"Bytes"`
    Amount int64    `plutusType:"Int"`
}
```

Tag options: `Bytes`, `Int`, `Map`, `IndefList`, `DefList`, `StringBytes`,
`Address`. `plutusConstr:"N"` sets the constructor index.

## Common Tasks

### Adding a `ChainContext` Method

1. Add the method to the interface in
   `txBuilding/Backend/Base/Base.go`.
2. Implement it in every backend under `txBuilding/Backend/`:
   `BlockFrostChainContext`, `MaestroChainContext`, `OgmiosChainContext`,
   `UtxorpcChainContext`, and `FixedChainContext`.
3. Add (or extend) tests for each implementation. `FixedChainContext` is the
   place to exercise the happy path deterministically.

### Adding a Transaction Builder Method

1. Add the method to the `Apollo` struct in `ApolloBuilder.go`.
2. Follow the fluent API pattern: return `*Apollo` (or
   `(*Apollo, error)` when the step can fail).
3. Extend `ApolloBuilder_test.go` with coverage.

### Adding a Serialization Type

1. Create `serialization/TypeName/` with the type and constructors.
2. Implement CBOR marshal/unmarshal using the `serialization/cbor` wrapper.
3. Add round-trip tests and, where applicable, known-vector tests against
   CBOR produced by cardano-cli or another reference implementation.

## PR and Commit Guidance

- Keep PRs focused — one logical change per PR. Mechanical refactors
  (e.g. `golines` reformatting) belong in their own commit or PR.
- Commit messages follow conventional-style prefixes used in recent history:
  `feat:`, `fix:`, `chore:`, `build(deps):`, `refactor:`, `test:`, `docs:`.
- Do not include generated artifacts (`coverage.out`, `coverage.html`,
  files under `tmp/`) in commits.
- CI runs `go-test`, `golangci-lint`, and `codeql-analysis` workflows under
  `.github/workflows/`. PRs must be green on all three before merge.

## Things to Flag in Code Review

- `fmt.Sprintf` used only for string concatenation → `perfsprint`.
- Slice appends in a loop without `make([]T, 0, n)` when `n` is known →
  `prealloc`.
- Unclosed `http.Response.Body` → `bodyclose`.
- Lines over 80 columns, unformatted struct tags, or unsplit long method
  chains → `golines` was not run.
- `panic` in library (non-test) code.
- Direct imports of `github.com/fxamacker/cbor/v2` outside
  `serialization/cbor/` — should use the internal wrapper.
- New exported APIs without tests or without doc comments on exported
  identifiers.
- Non-deterministic CBOR output (map iteration order, missing key sort).
