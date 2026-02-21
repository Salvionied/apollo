# Deregister Methods

This page documents **stake deregistration**: `DeregisterStake`, `DeregisterStakeFromAddress`, `DeregisterStakeFromBech32`. Deregistering returns the 2 ADA stake key deposit; `Complete()` accounts for the refund. Implementation: `[ApolloBuilder.go](../../ApolloBuilder.go)`, `[serialization/Certificate/Certificate.go](../../serialization/Certificate/Certificate.go)`.

## Purpose and method signatures

```go
func (b *Apollo) DeregisterStake(stakeCredential *Certificate.Credential) (*Apollo, error)
func (b *Apollo) DeregisterStakeFromAddress(address Address.Address) (*Apollo, error)
func (b *Apollo) DeregisterStakeFromBech32(address string) (*Apollo, error)
```

If `stakeCredential` is `nil` in `DeregisterStake`, the credential is taken from the wallet. Each appends a `StakeDeregistration` certificate.

## Inputs and constraints

- Credential from wallet (nil), from address, or from Bech32 string. Same pattern as registration helpers.

## Behavior details

- The transaction’s requested balance is **reduced** by the deposit refund in `Complete()` when deregistration certificates are present (see `addChangeAndFee` and balance logic in `ApolloBuilder.go`).

## Cardano CLI equivalence (10.14.0.0)

| CLI                                        | Apollo                                                                                             |
| ------------------------------------------ | -------------------------------------------------------------------------------------------------- |
| `stake-address deregistration-certificate` | `DeregisterStake(cred)` / `DeregisterStakeFromAddress(addr)` / `DeregisterStakeFromBech32(bech32)` |

## Example

**Apollo:**

```go
apollob, err = apollob.
    DeregisterStake(nil).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli stake-address deregistration-certificate --stake-verification-key-file stake.vkey --out-file dereg.cert
cardano-cli transaction build --certificate-file dereg.cert ...
```

## Evidence

- **Implementation-only** Certificate: `TestStakeDeregistrationRoundTrip` in `Certificate_test.go`.

## Caveats and validation

- Validate on preprod/mainnet with small amounts. Ensure the stake key is actually registered before deregistering.
