# Credential Helpers

This page documents how to obtain **stake credentials** and use **addresses with a staking part** in Apollo: `GetStakeCredentialFromWallet`, `GetStakeCredentialFromAddress`, and related address/wallet assumptions. Implementation: [`ApolloBuilder.go`](../../ApolloBuilder.go), [`serialization/Address`](../../serialization/Address/), [`apollotypes/types.go`](../../apollotypes/types.go).

## Purpose and method signatures

### GetStakeCredentialFromWallet

```go
func (b *Apollo) GetStakeCredentialFromWallet() (*Certificate.Credential, error)
```

Returns the stake credential derived from the wallet’s stake verification key. Requires a `GenericWallet` with non-empty `StakeVerificationKey` (e.g. from `SetWalletFromMnemonic`).

### GetStakeCredentialFromAddress

```go
func GetStakeCredentialFromAddress(address Address.Address) (*Certificate.Credential, error)
```

Uses `address.StakingPart` (must be 28 bytes). Builds a key-hash credential (Code 0). Returns an error if the address has no staking part or length is not 28.

## Inputs and constraints

- **Stake credential**: `Certificate.Credential` with `Code` (0 = key hash, 1 = script hash) and `Hash` (e.g. 28-byte key hash). Used in certificates and when building stake/reward addresses.
- **GetStakeCredentialFromWallet**: Only works with a wallet that has stake keys (e.g. `GenericWallet` from mnemonic). Custom wallets may not have stake keys.
- **GetStakeCredentialFromAddress**: Address must have a valid 28-byte `StakingPart` (base address or stake address). Withdrawal addresses must have this for `AddWithdrawal`.

## Behavior details

- **Address**: Apollo’s `Address` has `PaymentPart`, `StakingPart` (28 bytes for key hash), `HeaderByte`, `Network`, `AddressType`, `Hrp` (e.g. `stake1` / `stake_test1` for stake addresses). Base address = both parts set; stake-only address = staking part and correct header/hrp.
- **SetWalletFromMnemonic**: Derives payment and staking keys (staking path e.g. `m/1852'/1815'/0'/2/0`). `GenericWallet.StakeVerificationKey` and `StakeSigningKey` are set. `SignTx` signs with the stake key when the transaction uses it (certificates or withdrawals). There is no single `GetStakeAddress()` on the wallet interface; use `GetStakeCredentialFromWallet()` and build an address or pass the credential into certificate methods.

## Cardano CLI equivalence (10.14.0.0)

| CLI | Apollo |
|-----|--------|
| `stake-address build` (from verification key) | Build credential with `GetStakeCredentialFromWallet()` or `GetStakeCredentialFromAddress()`; build `Address` with staking part only and correct header/hrp. |
| `address build --stake-verification-key-file` | Build `Address` with `StakingPart` set to the 28-byte stake key hash. |
| Staking key from mnemonic | `SetWalletFromMnemonic`; wallet has `StakeVerificationKey` / `StakeSigningKey`. |

## Example

**Apollo (credential from wallet, then use in certificate):**

```go
cred, err := apollob.GetStakeCredentialFromWallet()
if err != nil {
    // wallet has no stake keys
}
apollob, err = apollob.RegisterStake(cred)
```

**Apollo (credential from Bech32 address):**

```go
addr, _ := Address.DecodeAddress("stake1u...")
cred, err := GetStakeCredentialFromAddress(addr)
```

**Cardano CLI:** `cardano-cli stake-address build --stake-verification-key-file stake.vkey` produces a stake address; in Apollo you build the credential and optionally an `Address` with that staking part.

## Evidence

- **Implementation-only** for these builder helpers. Wallet stake key derivation: `serialization/HDWallet/HDWallet_test.go` (e.g. `TestPaymentAddress12Reward`). Address encoding/decoding with staking part: `serialization/Address/Address_test.go`.

## Caveats and validation

- `GetStakeCredentialFromWallet()` only works with a wallet that implements stake keys (e.g. `GenericWallet` from mnemonic).
- Withdrawal addresses must have a valid 28-byte `StakingPart`; see [withdrawal_methods.md](withdrawal_methods.md).
- Validate on preprod/mainnet with small amounts when relying on implementation-only paths.
