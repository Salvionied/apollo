# AI Agent Instructions

This file provides guidance for AI coding agents working with this repository.

## Project Overview

Apollo is a pure Golang library for Cardano blockchain development. It provides CBOR serialization, transaction building, wallet management, and smart contract (Plutus) integration.

## Build and Test Commands

```bash
make mod-tidy    # Fetch dependencies
make format      # Run go fmt
make test        # Run tests with race detection
make clean       # Remove temporary files
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
| `ApolloBuilder.go` | Main transaction builder (fluent API) |
| `backends.go` | Factory functions for chain contexts |
| `Models.go` | Core types: Unit, Payment, PaymentI |

### Package Structure

| Package | Purpose |
|---------|---------|
| `serialization/` | CBOR serialization for all Cardano types |
| `txBuilding/Backend/` | Blockchain backend implementations |
| `txBuilding/Backend/Base/` | ChainContext interface definition |
| `crypto/` | Cryptographic primitives |
| `apollotypes/` | Wallet interfaces |
| `plutusencoder/` | Plutus data marshaling |

### Key Interfaces

**ChainContext** (`txBuilding/Backend/Base/Base.go`):
- `GetProtocolParams()` - Protocol parameters
- `Utxos(address)` - Query UTxOs
- `SubmitTx(tx)` - Submit transaction
- `EvaluateTx(tx)` - Evaluate Plutus scripts

**Wallet** (`apollotypes/Wallet.go`):
- `GetAddress()`, `SignTx()`, `PkeyHash()`, `SkeyHash()`

## Coding Standards

### Error Handling

- Return errors, never panic
- Never silently ignore errors
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`

### Naming

- Use camelCase for variables (not snake_case)
- Exported types use PascalCase
- Package names are lowercase, single word

### Testing

- All new code requires tests
- Use `FixedChainContext` for deterministic tests
- Run `make test` before committing

### CBOR Serialization

- Implement `MarshalCBOR`/`UnmarshalCBOR` for custom types
- Ensure deterministic encoding (sort map keys)
- Test roundtrip: marshal -> unmarshal -> compare

## Plutus Data Struct Tags

```go
type Datum struct {
    _      struct{} `plutusType:"IndefList" plutusConstr:"1"`
    Pkh    []byte   `plutusType:"Bytes"`
    Amount int64    `plutusType:"Int"`
}
```

Options: `Bytes`, `Int`, `Map`, `IndefList`, `DefList`, `StringBytes`

## Common Tasks

### Adding a Backend Method

1. Add to interface in `txBuilding/Backend/Base/Base.go`
2. Implement in each backend:
   - `BlockFrostChainContext/`
   - `MaestroChainContext/`
   - `OgmiosChainContext/`
   - `UtxorpcChainContext/`
   - `FixedChainContext/`
3. Add tests for each implementation

### Adding Transaction Builder Method

1. Add method to `Apollo` struct in `ApolloBuilder.go`
2. Follow fluent API pattern (return `*Apollo`)
3. Add test in `ApolloBuilder_test.go`

### Adding Serialization Type

1. Create package in `serialization/TypeName/`
2. Implement CBOR marshal/unmarshal
3. Add roundtrip tests
4. Export from package

## Go Version

Requires Go 1.24+
