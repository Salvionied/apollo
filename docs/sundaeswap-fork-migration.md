# Migration Guide: SundaeSwap-finance/apollo Fork to Salvionied/apollo Master

This guide covers the changes needed when switching from
`github.com/SundaeSwap-finance/apollo` back to
`github.com/Salvionied/apollo`.

Both codebases share the same lineage. The upstream master has incorporated
most of the SundaeSwap fork's improvements (error returns, PlutusV3, cost
model rework, etc.) and added further refinements. The migration is mostly
mechanical: update import paths and adjust to minor API differences.

---

## 1. Module / Import Path

**Every import must change:**

```go
// Before (fork)
import "github.com/SundaeSwap-finance/apollo"
import "github.com/SundaeSwap-finance/apollo/serialization/PlutusData"

// After (upstream)
import "github.com/Salvionied/apollo"
import "github.com/Salvionied/apollo/serialization/PlutusData"
```

A global find-and-replace handles this:

```bash
find . -name '*.go' -exec sed -i \
  's|github.com/SundaeSwap-finance/apollo|github.com/Salvionied/apollo|g' {} +
go mod tidy
```

The fork depended on `github.com/SundaeSwap-finance/kugo` and
`github.com/SundaeSwap-finance/ogmigo/v6`. Upstream uses
`github.com/blinklabs-io/kugo` and `github.com/blinklabs-io/ogmigo/v6`
(or other equivalents). You do not need to worry about these transitive
dependencies unless you imported them directly.

---

## 2. APIs That Are Identical

The following SundaeSwap fork APIs exist in upstream master with the
**same signature** -- no changes needed beyond the import path swap:

| API | Notes |
|-----|-------|
| `Complete() (*Apollo, []byte, error)` | 3-return-value form |
| `MintAssetsWithRedeemer(Unit, PlutusData)` | Takes `PlutusData` not `Redeemer` |
| `ForceFee(int64) *Apollo` | |
| `GetMints() Value.Value` | |
| `GetBurns() Value.Value` | |
| `GetEstimatedFee() (int64, error)` | |
| `SetAdditionalUTxOs([]UTxO.UTxO) *Apollo` | |
| `PayToContractAsHash(...)` | |
| `AttachV3Script(PlutusV3Script)` | |
| `GetUsedUTxOs() map[string]bool` | Was `[]string` in old upstream |
| `RedeemerTagNames` | Was `RdeemerTagNames` in old upstream |
| `Address.AddressFromBytes(payment, paymentIsScript, staking, stakingIsScript, network)` | Script credential support |
| `Address.IsPublicKeyAddress() bool` | |
| `ChainContext.Utxos() ([]UTxO.UTxO, error)` | Error return |
| `ChainContext.EvaluateTx() (map[...]..., error)` | Error return |
| `ChainContext.EvaluateTxWithAdditionalUtxos(...)` | |
| `ChainContext.CostModelsV1/V2/V3()` | |
| `PlutusData.CostModel` type | |
| `Transaction.Bytes()` uses `CanonicalEncOptions` | |
| `TransactionOutput.GetScriptRef()` returns `nil` for pre-Alonzo | |

---

## 3. APIs That Differ

### 3.1 `AddReferenceScriptV1/V2/V3`

The fork combines reference input registration with script version tagging:

```go
// Fork -- takes txHash and index, implicitly calls AddReferenceInput
b.AddReferenceScriptV2(txHash, index)
```

Upstream separates the two concerns:

```go
// Upstream -- call AddReferenceInput yourself, then tag the version
b.AddReferenceInput(txHash, index).AddReferenceScriptV2()
```

**Migration:** Split the single call into two chained calls.

### 3.2 `UtxoFromRef` Return Type

The fork returns a value type:

```go
// Fork
func (b *Apollo) UtxoFromRef(txHash string, txIndex int) (UTxO.UTxO, error)
```

Upstream returns a pointer:

```go
// Upstream
func (b *Apollo) UtxoFromRef(txHash string, txIndex int) (*UTxO.UTxO, error)
```

**Migration:** Handle the `*UTxO.UTxO` return; check for `nil` in addition
to checking the error.

### 3.3 `GetUtxoFromRef` (ChainContext Interface)

Same story -- the fork returns `(UTxO.UTxO, error)`, upstream returns
`(*UTxO.UTxO, error)`:

```go
// Fork
GetUtxoFromRef(txHash string, txIndex int) (UTxO.UTxO, error)

// Upstream
GetUtxoFromRef(txHash string, txIndex int) (*UTxO.UTxO, error)
```

**Migration:** If you implement `ChainContext`, update your return type
to `*UTxO.UTxO`.

### 3.4 `ScriptRef` Type

The fork uses a struct with an `InnerScript` field (renamed from `_Script`):

```go
// Fork
type ScriptRef struct {
    Script InnerScript
}
type InnerScript struct {
    _ struct{} `cbor:",toarray"`
    Script []byte
}
```

Upstream uses a flat `[]byte` alias with proper CBOR tag-24 handling and
constructor helpers:

```go
// Upstream
type ScriptRef []byte

func NewV1ScriptRef(script PlutusV1Script) (ScriptRef, error)
func NewV2ScriptRef(script PlutusV2Script) (ScriptRef, error)
func NewV3ScriptRef(script PlutusV3Script) (ScriptRef, error)
```

**Migration:** Replace direct `ScriptRef{Script: InnerScript{...}}`
construction with the `NewV1ScriptRef` / `NewV2ScriptRef` /
`NewV3ScriptRef` constructors. Access to the raw script bytes changes
from `scriptRef.Script.Script` to `[]byte(scriptRef)`.

### 3.5 `WalletAddressFromBytes`

The fork renamed this to `AddressFromBytes` and removed the old name.
Upstream has **both**: the new `AddressFromBytes` (with script credential
flags) and a compatibility wrapper `WalletAddressFromBytes` that delegates
to it with `paymentIsScript=false, stakingIsScript=false`.

**Migration:** No action needed. If you were using `AddressFromBytes`,
it exists. If you were using `WalletAddressFromBytes`, it also still
exists upstream.

### 3.6 `SetWalletFromKeypair` Signing Key Handling

The fork passes the raw bytes directly to `Key.SigningKey`:

```go
// Fork
signingKey := Key.SigningKey{Payload: signingKey_bytes}
```

Upstream uses `ed25519.NewKeyFromSeed()` to derive the full 64-byte
private key from the 32-byte seed, matching the cardano-cli convention:

```go
// Upstream
signingKey := Key.SigningKey{
    Payload: ed25519.NewKeyFromSeed(signingKey_bytes),
}
```

**Migration:** If you were passing raw 32-byte seeds from cardano-cli
key files, upstream handles the conversion automatically. If you were
relying on the fork's raw-passthrough behavior with pre-expanded 64-byte
keys, you may need to adjust.

### 3.7 Additional Error Returns on ChainContext Methods

Upstream has expanded error returns on additional `ChainContext` methods
that the fork may not have changed:

```go
// Upstream adds error returns to:
GetProtocolParams() (ProtocolParameters, error) // fork may not return error
GetGenesisParams() (GenesisParameters, error)
Epoch() (int, error)
MaxTxFee() (int, error)
LastBlockSlot() (int, error)
GetContractCbor(string) (string, error)
```

**Migration:** If you implement `ChainContext`, add error returns to
these methods.

### 3.8 `ProtocolParameters.MinFeeReferenceScripts`

Both have this field, but upstream uses it in the `Fee()` calculation
identically. No migration needed.

### 3.9 Ogmios Backend

The fork upgraded to `ogmigo/v6` v6.1.0 with major rewrites to the
Ogmios chain context. Upstream also supports `ogmigo/v6` but through
its own backend paths. If you were using `OgmiosChainContext` directly,
verify your constructor calls match the upstream API.

---

## 4. Deleted Code

### `txBuilding/TxBuilder/TxBuilder.go`

The fork deleted this file (the old transaction builder). Upstream also
does not ship this file on master. If you were importing from
`txBuilding/TxBuilder`, that package no longer exists in either codebase
-- use `apollo.New(cc)` with the builder pattern instead.

---

## 5. Dependency Changes

| Fork | Upstream |
|------|----------|
| `go 1.23.0` | `go 1.25` |
| `github.com/SundaeSwap-finance/kugo v1.3.0` | Different kugo dependency |
| `github.com/SundaeSwap-finance/ogmigo/v6 v6.1.0` | Different ogmigo dependency |
| `golang.org/x/exp` (removed) | Also removed |
| `github.com/tyler-smith/go-bip39` | `github.com/blinklabs-io/go-bip39` |

Run `go mod tidy` after switching imports.

---

## 6. Migration Checklist

1. **Find-and-replace** all `github.com/SundaeSwap-finance/apollo`
   imports to `github.com/Salvionied/apollo`
2. **Split** `AddReferenceScriptV1/V2/V3(txHash, index)` calls into
   `AddReferenceInput(txHash, index).AddReferenceScriptV1/V2/V3()`
3. **Update** `UtxoFromRef` / `GetUtxoFromRef` callers to expect
   `*UTxO.UTxO` instead of `UTxO.UTxO`
4. **Update** any custom `ChainContext` implementations to match the
   expanded interface (error returns on `GetProtocolParams`,
   `GetGenesisParams`, `Epoch`, `MaxTxFee`, `LastBlockSlot`,
   `GetContractCbor`)
5. **Replace** `ScriptRef` struct usage with the new `[]byte`-based
   `ScriptRef` and its `NewV1/V2/V3ScriptRef` constructors
6. **Run** `go mod tidy` to resolve dependency changes
7. **Run** `make format && make golines` to match upstream style
   (80-char lines)
8. **Run** `go test -race ./...` to verify
