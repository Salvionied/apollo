# Apollo v1 to v2 Migration Guide

Apollo v2 replaces all custom serialization code with [gouroboros](https://github.com/blinklabs-io/gouroboros) types, reducing the codebase from ~13,000+ lines to ~3,500. The API has been simplified with unified methods, while convenience wrappers preserve familiar calling patterns.

## Import Path

```go
// v1
import "github.com/Salvionied/apollo"

// v2
import "github.com/Salvionied/apollo/v2"
```

## Quick Reference: Method Changes

| v1 Method | v2 Replacement |
|-----------|---------------|
| `AttachV1Script(script)` | `AttachScript(script)` |
| `AttachV2Script(script)` | `AttachScript(script)` |
| `AttachV3Script(script)` | `AttachScript(script)` |
| `AttachDatum(datum)` | `AttachDatum(datum)` (same) or `AddDatum(datum)` |
| `AddPlutusV1Script(script)` | `AttachScript(script)` |
| `AddPlutusV2Script(script)` | `AttachScript(script)` |
| `AddPlutusV3Script(script)` | `AttachScript(script)` |
| `AddNativeScript(script)` | `AttachScript(script)` |
| `AddMint(units...)` | `Mint(unit, nil, nil)` |
| `MintAssetsWithRedeemer(unit, redeemer, exUnits)` | `Mint(unit, &redeemer, &exUnits)` |
| `PayToAddressBech32(addr, ...)` | `PayToAddressBech32(addr, ...)` (same) or parse + `PayToAddress(addr, ...)` |
| `AddInputAddressFromBech32(addr)` | `AddInputAddressFromBech32(addr)` (same) or parse + `AddInputAddress(addr)` |
| `SetWalletFromBech32(addr)` | Parse address, then `SetWallet(NewExternalWallet(addr))` |
| `SetWalletFromKeypair(vkey, skey, net)` | Build address manually, then `SetWallet(NewExternalWallet(addr))` |
| `SetChangeAddressBech32(addr)` | `SetChangeAddressBech32(addr)` (same) or parse + `SetChangeAddress(addr)` |
| `SetWalletAsChangeAddress()` | Not needed — wallet address is the default change address |
| `AddRequiredSignerFromBech32(addr)` | Parse address, then use `AddRequiredSignerPaymentKey(addr)` or `AddRequiredSignerStakeKey(addr)` |
| `AddRequiredSignerFromAddress(addr, payment, staking)` | Use `AddRequiredSignerPaymentKey(addr)` and/or `AddRequiredSignerStakeKey(addr)` |
| `AddReferenceInputV3(hash, idx)` | `AddReferenceInput(hash, idx)` |
| `PayToAddressWithV1ReferenceScript(...)` | `PayToAddressWithV1ReferenceScript(...)` (same) or `PayToAddressWithReferenceScript(addr, lovelace, script, units...)` |
| `PayToAddressWithV2ReferenceScript(...)` | `PayToAddressWithV2ReferenceScript(...)` (same) or `PayToAddressWithReferenceScript(addr, lovelace, script, units...)` |
| `PayToAddressWithV3ReferenceScript(...)` | `PayToAddressWithV3ReferenceScript(...)` (same) or `PayToAddressWithReferenceScript(addr, lovelace, script, units...)` |
| `PayToContractWithV1ReferenceScript(...)` | `PayToContractWithV1ReferenceScript(...)` or `PayToContractWithReferenceScript(addr, datum, lovelace, script, units...)` |
| `PayToContractWithV2ReferenceScript(...)` | `PayToContractWithV2ReferenceScript(...)` or `PayToContractWithReferenceScript(addr, datum, lovelace, script, units...)` |
| `PayToContractWithV3ReferenceScript(...)` | `PayToContractWithV3ReferenceScript(...)` or `PayToContractWithReferenceScript(addr, datum, lovelace, script, units...)` |
| `CompleteExact(fee)` | `SetFee(fee)` then `Complete()` |
| `SetEstimateRequired()` | Automatic — set by `CollectFrom` and `Mint` with redeemers |
| `ConsumeAssetsFromUtxo(utxo, payments...)` | `AddInput(utxo)` then `AddPayment(payments...)` |
| `GetPaymentsLength()` | Removed — internal detail |
| `GetRedeemers()` | Removed — leaked private type |
| `UpdateRedeemers(r)` | Removed — leaked private type |
| `GetSortedInputs()` | Removed — internal detail |
| `RegisterStakeFromAddress(addr)` | `RegisterStakeFromAddress(addr)` (same) or `RegisterStake(addr)` |
| `RegisterStakeFromBech32(bech32)` | `RegisterStakeFromBech32(bech32)` (same) or `RegisterStake(bech32)` |
| `DeregisterStakeFromAddress(addr)` | `DeregisterStakeFromAddress(addr)` (same) or `DeregisterStake(addr)` |
| `DeregisterStakeFromBech32(bech32)` | `DeregisterStakeFromBech32(bech32)` (same) or `DeregisterStake(bech32)` |
| `DelegateStakeFromAddress(addr, pool)` | `DelegateStakeFromAddress(addr, pool)` (same) or `DelegateStake(addr, pool)` |
| `DelegateStakeFromBech32(bech32, pool)` | `DelegateStakeFromBech32(bech32, pool)` (same) or `DelegateStake(bech32, pool)` |
| `RegisterAndDelegateStakeFromAddress(...)` | `RegisterAndDelegateStakeFromAddress(...)` (same) or `RegisterAndDelegateStake(addr, pool, coin)` |
| `RegisterAndDelegateStakeFromBech32(...)` | `RegisterAndDelegateStakeFromBech32(...)` (same) or `RegisterAndDelegateStake(bech32, pool, coin)` |
| `DelegateVoteFromAddress(addr, drep)` | `DelegateVoteFromAddress(addr, drep)` (same) or `DelegateVote(addr, drep)` |
| `DelegateVoteFromBech32(bech32, drep)` | `DelegateVoteFromBech32(bech32, drep)` (same) or `DelegateVote(bech32, drep)` |
| `DelegateStakeAndVoteFromAddress(...)` | `DelegateStakeAndVoteFromAddress(...)` (same) or `DelegateStakeAndVote(addr, pool, drep)` |
| `DelegateStakeAndVoteFromBech32(...)` | `DelegateStakeAndVoteFromBech32(...)` (same) or `DelegateStakeAndVote(bech32, pool, drep)` |
| `RegisterAndDelegateVoteFromAddress(...)` | `RegisterAndDelegateVoteFromAddress(...)` (same) or `RegisterAndDelegateVote(addr, drep, coin)` |
| `RegisterAndDelegateVoteFromBech32(...)` | `RegisterAndDelegateVoteFromBech32(...)` (same) or `RegisterAndDelegateVote(bech32, drep, coin)` |
| `RegisterAndDelegateStakeAndVoteFromAddress(...)` | `RegisterAndDelegateStakeAndVoteFromAddress(...)` (same) or `RegisterAndDelegateStakeAndVote(addr, pool, drep, coin)` |
| `RegisterAndDelegateStakeAndVoteFromBech32(...)` | `RegisterAndDelegateStakeAndVoteFromBech32(...)` (same) or `RegisterAndDelegateStakeAndVote(bech32, pool, drep, coin)` |
| `NewV1ScriptRef(script)` | `NewScriptRef(script)` |
| `NewV2ScriptRef(script)` | `NewScriptRef(script)` |
| `NewV3ScriptRef(script)` | `NewScriptRef(script)` |

## Dependencies

Apollo v2 uses [bursa](https://github.com/blinklabs-io/bursa) v0.15.0 for HD wallet key derivation and transaction signing. Bursa provides:

- **BIP32-Ed25519 key derivation** following CIP-1852 (payment, stake, DRep, committee, and calidus keys)
- **Native `XPrv.Sign()`** for correct CIP-1852 extended Ed25519 signatures
- **Mnemonic generation** via `GenerateMnemonic()`

Wallet creation uses bursa internally — see `SetWalletFromMnemonic` and `NewBursaWallet`.

## Detailed Changes

### 1. Script Attachment (12+ methods -> 1 unified + convenience)

v2 uses a single `AttachScript` method that handles all script types — PlutusV1, PlutusV2, PlutusV3, and NativeScript. Duplicate scripts are ignored automatically.

```go
// v1
a.AttachV1Script(v1Script)
a.AttachV2Script(v2Script)
a.AttachV3Script(v3Script)
a.AddNativeScript(nativeScript)

// v2 - single method, auto-detects type
a.AttachScript(v1Script)
a.AttachScript(v2Script)
a.AttachScript(v3Script)
a.AttachScript(nativeScript)
```

`AttachDatum` is still available as an alias for `AddDatum`:

```go
// Both work in v2
a.AttachDatum(datum)
a.AddDatum(datum)
```

### 2. Script References (6 constructors -> 1)

```go
// v1
ref := apollo.NewV1ScriptRef(v1Script)
ref := apollo.NewV2ScriptRef(v2Script)
ref := apollo.NewV3ScriptRef(v3Script)

// v2 - auto-detects type
ref := apollo.NewScriptRef(script)
```

### 3. Minting (2 methods -> 1)

`AddMint` and `MintAssetsWithRedeemer` are replaced by a single `Mint` method. Pass nil for redeemer/exUnits for native minting:

```go
// v1 - native mint
a.AddMint(unit1, unit2)

// v2 - native mint (one unit at a time, nil redeemer)
a.Mint(unit1, nil, nil)
a.Mint(unit2, nil, nil)

// v1 - script mint
a.MintAssetsWithRedeemer(unit, redeemer, exUnits)

// v2 - script mint (pass pointers)
a.Mint(unit, &redeemer, &exUnits)
```

### 4. PayToContract Signature Change

The `isInline` boolean parameter is removed. Use the method name to choose behavior:

```go
// v1 - inline datum
a.PayToContract(addr, datum, lovelace, true, units...)

// v2 - inline datum (default)
a.PayToContract(addr, datum, lovelace, units...)

// v1 - datum hash
a.PayToContract(addr, datum, lovelace, false, units...)

// v2 - datum hash
a.PayToContractWithDatumHash(addr, datum, lovelace, units...)

// v1 - pre-computed hash
a.PayToContractAsHash(addr, hashBytes, lovelace, false)

// v2 - pre-computed hash (still available)
a.PayToContractAsHash(addr, hashBytes, lovelace)
```

### 5. Reference Script Payments (unified + convenience)

v2 provides a unified method with auto-detection, plus version-specific convenience wrappers:

```go
// v1
a.PayToAddressWithV1ReferenceScript(addr, lovelace, v1Script, units...)
a.PayToAddressWithV2ReferenceScript(addr, lovelace, v2Script, units...)
a.PayToAddressWithV3ReferenceScript(addr, lovelace, v3Script, units...)

// v2 - unified (auto-detects type)
a.PayToAddressWithReferenceScript(addr, lovelace, script, units...)

// v2 - version-specific (still available as convenience)
a.PayToAddressWithV1ReferenceScript(addr, lovelace, v1Script, units...)
a.PayToAddressWithV2ReferenceScript(addr, lovelace, v2Script, units...)
a.PayToAddressWithV3ReferenceScript(addr, lovelace, v3Script, units...)

// v2 - contract + reference script (new unified method)
a.PayToContractWithReferenceScript(addr, datum, lovelace, script, units...)

// v2 - version-specific contract + reference script (convenience)
a.PayToContractWithV1ReferenceScript(addr, datum, lovelace, v1Script, units...)
a.PayToContractWithV2ReferenceScript(addr, datum, lovelace, v2Script, units...)
a.PayToContractWithV3ReferenceScript(addr, datum, lovelace, v3Script, units...)
```

### 6. Bech32 Convenience Methods

Bech32 convenience methods are available. You can also parse the address yourself and use the core methods:

```go
// v2 - convenience methods (parse bech32 for you)
a, err = a.PayToAddressBech32("addr1...", 2_000_000)
a, err = a.AddInputAddressFromBech32("addr1...")
a, err = a.SetChangeAddressBech32("addr1...")

// v2 - explicit parsing (recommended for reuse)
addr, err := common.NewAddress("addr1...")
a.PayToAddress(addr, 2_000_000)
a.AddInputAddress(addr)
a.SetChangeAddress(addr)
```

**Note**: The `SetWalletFromBech32` method is not available in v2. Parse the address and use `NewExternalWallet`:
```go
addr, err := common.NewAddress("addr1...")
a.SetWallet(apollo.NewExternalWallet(addr))
```

### 7. Required Signers (Boolean flags -> Named methods)

```go
// v1 - boolean flags for payment/staking
a.AddRequiredSignerFromAddress(addr, true, false)   // payment only
a.AddRequiredSignerFromAddress(addr, false, true)    // staking only
a.AddRequiredSignerFromAddress(addr, true, true)     // both

// v2 - explicit named methods
a.AddRequiredSignerPaymentKey(addr)  // payment only
a.AddRequiredSignerStakeKey(addr)    // staking only
a.AddRequiredSignerPaymentKey(addr).AddRequiredSignerStakeKey(addr) // both
```

### 8. Reference Inputs (Unified)

`AddReferenceInputV3` is removed. All reference inputs use the same method:

```go
// v1
a.AddReferenceInput(txHash, idx)    // for V1/V2
a.AddReferenceInputV3(txHash, idx)  // for V3

// v2
a, err = a.AddReferenceInput(txHash, idx)    // for all script versions
```

### 9. Staking & Delegation (unified + convenience)

v2 provides unified methods that accept flexible input types via `any`, plus `FromAddress`/`FromBech32` convenience wrappers for type safety:

```go
// v2 - unified method, multiple input types
a.RegisterStake(&cred)       // credential pointer
a.RegisterStake(addr)        // common.Address
a.RegisterStake("addr1...")  // bech32 string
a.RegisterStake(nil)         // wallet fallback

// v2 - type-safe convenience wrappers (still available)
a.RegisterStakeFromAddress(addr)
a.RegisterStakeFromBech32("stake1u...")
```

This pattern applies to all 8 staking/delegation operations:
- `RegisterStake` / `RegisterStakeFromAddress` / `RegisterStakeFromBech32`
- `DeregisterStake` / `DeregisterStakeFromAddress` / `DeregisterStakeFromBech32`
- `DelegateStake` / `DelegateStakeFromAddress` / `DelegateStakeFromBech32`
- `RegisterAndDelegateStake` / `RegisterAndDelegateStakeFromAddress` / `RegisterAndDelegateStakeFromBech32`
- `DelegateVote` / `DelegateVoteFromAddress` / `DelegateVoteFromBech32`
- `DelegateStakeAndVote` / `DelegateStakeAndVoteFromAddress` / `DelegateStakeAndVoteFromBech32`
- `RegisterAndDelegateVote` / `RegisterAndDelegateVoteFromAddress` / `RegisterAndDelegateVoteFromBech32`
- `RegisterAndDelegateStakeAndVote` / `RegisterAndDelegateStakeAndVoteFromAddress` / `RegisterAndDelegateStakeAndVoteFromBech32`

**Note**: All staking/delegation methods now return `(*Apollo, error)` instead of `*Apollo`.

### 10. Removed Methods (No Direct Replacement)

| Method | Reason |
|--------|--------|
| `CompleteExact(fee)` | Use `SetFee(fee)` then `Complete()` |
| `SetWalletAsChangeAddress()` | Default behavior — wallet is always the change address |
| `SetWalletFromKeypair(...)` | Incomplete implementation — build address manually |
| `SetEstimateRequired()` | Internal — automatically set by `CollectFrom`/`Mint` |
| `ConsumeAssetsFromUtxo(...)` | Use `AddInput(utxo)` then `AddPayment(...)` |
| `GetPaymentsLength()` | Internal detail |
| `GetRedeemers()` | Leaked private type |
| `UpdateRedeemers(r)` | Leaked private type |
| `GetSortedInputs()` | Internal detail |

### 11. Type Changes

| v1 Type | v2 Type |
|---------|---------|
| `serialization.TransactionBody` | `conway.ConwayTransactionBody` |
| `serialization.Transaction` | `conway.ConwayTransaction` |
| `serialization.TransactionOutput` | `babbage.BabbageTransactionOutput` |
| `serialization.MultiAsset` | `common.MultiAsset[T]` |
| `serialization.PlutusData` | `common.Datum` |
| `serialization.Redeemer` | `common.RedeemerKey` + `common.RedeemerValue` |
| `serialization.NativeScript` | `common.NativeScript` |
| `PlutusData` (custom) | `common.PlutusData` / `common.Datum` |

### 12. Backend / Chain Context

The `ChainContext` interface is now in `backend` package with gouroboros types:

```go
// v1
import "github.com/Salvionied/apollo/txBuilding/Backend/Base"

// v2
import "github.com/Salvionied/apollo/v2/backend"
```

Supported backends: `blockfrost`, `ogmios`, `maestro`, `utxorpc`, `fixed` (testing).

### 13. Value Type

v2 introduces an explicit `Value` type replacing various ad-hoc representations:

```go
v := apollo.NewSimpleValue(2_000_000)           // ADA only
v := apollo.NewValue(2_000_000, multiAsset)     // ADA + assets
result, err := v.Add(other)                     // returns error on overflow
v, err = v.Sub(other)
ok := v.GreaterOrEqual(other)
```

**Note**: `Value.Add` returns `(Value, error)` to detect uint64 overflow.

### 14. Type Safety Improvements

**`int64` for monetary amounts**: `Unit.Quantity` and `Payment.Lovelace` are now `int64` (previously `int`) to ensure consistent 64-bit precision across all platforms.

**Error-returning interfaces**: `PaymentI.ToValue()` and `PaymentI.ToTxOut()` now return errors instead of silently swallowing them:

```go
// v1
type PaymentI interface {
    ToValue() Value
    ToTxOut() *babbage.BabbageTransactionOutput
}

// v2
type PaymentI interface {
    ToValue() (Value, error)
    ToTxOut() (*babbage.BabbageTransactionOutput, error)
}
```

**`ParseFraction` returns errors**: `backend.ParseFraction` now returns `(float64, error)` instead of silently returning 0 on invalid input.

### 15. Wallet Passphrase Support

`NewBursaWallet` keeps the simple signature for common use. Use `NewBursaWalletWithPassphrase` for BIP39 passphrase:

```go
// No passphrase (most common)
w, err := apollo.NewBursaWallet(mnemonic)

// With passphrase
w, err := apollo.NewBursaWalletWithPassphrase(mnemonic, "my-secret")

// Via Apollo builder
a, err = a.SetWalletFromMnemonic(mnemonic)                        // no passphrase
a, err = a.SetWalletFromMnemonicWithPassphrase(mnemonic, "pass")  // with passphrase
```

## Complete v2 Public API

### Construction & Wallet
- `New(cc) *Apollo`
- `SetWallet(w) *Apollo`
- `SetWalletFromMnemonic(mnemonic) (*Apollo, error)`
- `SetWalletFromMnemonicWithPassphrase(mnemonic, passphrase) (*Apollo, error)`
- `GetWallet() Wallet`

### Payments
- `AddPayment(payment) *Apollo`
- `PayToAddress(addr, lovelace, units...) *Apollo`
- `PayToAddressBech32(bech32, lovelace, units...) (*Apollo, error)`
- `PayToContract(addr, datum, lovelace, units...) *Apollo`
- `PayToContractWithDatumHash(addr, datum, lovelace, units...) (*Apollo, error)`
- `PayToContractAsHash(addr, datumHash, lovelace, units...) *Apollo`
- `PayToAddressWithReferenceScript(addr, lovelace, script, units...) (*Apollo, error)`
- `PayToAddressWithV1ReferenceScript(addr, lovelace, script, units...) (*Apollo, error)`
- `PayToAddressWithV2ReferenceScript(addr, lovelace, script, units...) (*Apollo, error)`
- `PayToAddressWithV3ReferenceScript(addr, lovelace, script, units...) (*Apollo, error)`
- `PayToContractWithReferenceScript(addr, datum, lovelace, script, units...) (*Apollo, error)`
- `PayToContractWithV1ReferenceScript(addr, datum, lovelace, script, units...) (*Apollo, error)`
- `PayToContractWithV2ReferenceScript(addr, datum, lovelace, script, units...) (*Apollo, error)`
- `PayToContractWithV3ReferenceScript(addr, datum, lovelace, script, units...) (*Apollo, error)`

### Inputs & UTxOs
- `AddInput(utxo) *Apollo`
- `AddLoadedUTxOs(utxos...) *Apollo`
- `AddInputAddress(addr) *Apollo`
- `AddInputAddressFromBech32(bech32) (*Apollo, error)`
- `CollectFrom(utxo, redeemer, exUnits) *Apollo`
- `ConsumeUTxO(utxo, payments...) (*Apollo, error)`
- `UtxoFromRef(hash, index) (*Utxo, error)`
- `GetUsedUTxOs() []string`

### Scripts & Minting
- `AttachScript(script) *Apollo`
- `AddDatum(datum) *Apollo`
- `AttachDatum(datum) *Apollo`
- `Mint(unit, redeemer, exUnits) *Apollo`
- `GetBurns() (Value, error)`

### Reference Inputs
- `AddReferenceInput(txHash, index) (*Apollo, error)`

### Required Signers
- `AddRequiredSigner(pkh) *Apollo`
- `AddRequiredSignerPaymentKey(addr) *Apollo`
- `AddRequiredSignerStakeKey(addr) *Apollo`

### Transaction Parameters
- `SetTtl(ttl) *Apollo`
- `SetValidityStart(start) *Apollo`
- `SetFee(fee) *Apollo`
- `SetFeePadding(padding) *Apollo`
- `SetChangeAddress(addr) *Apollo`
- `SetChangeAddressBech32(bech32) (*Apollo, error)`
- `SetCollateralAmount(amount) *Apollo`
- `AddCollateral(utxo) *Apollo`
- `DisableExecutionUnitsEstimation() *Apollo`

### Staking & Delegation (unified + convenience)
- `RegisterStake(credOrAddr) (*Apollo, error)`
- `RegisterStakeFromAddress(addr) (*Apollo, error)`
- `RegisterStakeFromBech32(bech32) (*Apollo, error)`
- `DeregisterStake(credOrAddr) (*Apollo, error)`
- `DeregisterStakeFromAddress(addr) (*Apollo, error)`
- `DeregisterStakeFromBech32(bech32) (*Apollo, error)`
- `DelegateStake(credOrAddr, poolHash) (*Apollo, error)`
- `DelegateStakeFromAddress(addr, poolHash) (*Apollo, error)`
- `DelegateStakeFromBech32(bech32, poolHash) (*Apollo, error)`
- `RegisterAndDelegateStake(credOrAddr, poolHash, coin) (*Apollo, error)`
- `RegisterAndDelegateStakeFromAddress(addr, poolHash, coin) (*Apollo, error)`
- `RegisterAndDelegateStakeFromBech32(bech32, poolHash, coin) (*Apollo, error)`
- `DelegateVote(credOrAddr, drep) (*Apollo, error)`
- `DelegateVoteFromAddress(addr, drep) (*Apollo, error)`
- `DelegateVoteFromBech32(bech32, drep) (*Apollo, error)`
- `DelegateStakeAndVote(credOrAddr, poolHash, drep) (*Apollo, error)`
- `DelegateStakeAndVoteFromAddress(addr, poolHash, drep) (*Apollo, error)`
- `DelegateStakeAndVoteFromBech32(bech32, poolHash, drep) (*Apollo, error)`
- `RegisterAndDelegateVote(credOrAddr, drep, coin) (*Apollo, error)`
- `RegisterAndDelegateVoteFromAddress(addr, drep, coin) (*Apollo, error)`
- `RegisterAndDelegateVoteFromBech32(bech32, drep, coin) (*Apollo, error)`
- `RegisterAndDelegateStakeAndVote(credOrAddr, poolHash, drep, coin) (*Apollo, error)`
- `RegisterAndDelegateStakeAndVoteFromAddress(addr, poolHash, drep, coin) (*Apollo, error)`
- `RegisterAndDelegateStakeAndVoteFromBech32(bech32, poolHash, drep, coin) (*Apollo, error)`
- `RegisterPool(params) *Apollo`
- `DeregisterPool(poolHash, epoch) *Apollo`
- `SetCertificates(certs) *Apollo`
- `GetStakeCredentialFromWallet() (Credential, error)`

### Withdrawals & Metadata
- `AddWithdrawal(address, amount, redeemer, exUnits) *Apollo`
- `SetShelleyMetadata(metadata) *Apollo`

### Building & Signing
- `Complete() (*Apollo, error)`
- `Sign() (*Apollo, error)`
- `SignWithSkey(skey) (*Apollo, error)`
- `AddVerificationKeyWitness(witness) (*Apollo, error)`
- `Submit() (Blake2b256, error)`
- `GetTx() *ConwayTransaction`
- `GetTxCbor() ([]byte, error)`
- `LoadTxCbor(hex) (*Apollo, error)`
- `Clone() *Apollo`
