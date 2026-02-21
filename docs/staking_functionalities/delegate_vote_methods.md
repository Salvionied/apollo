# Delegate Vote Methods

This page documents **vote (DRep) delegation**: `DelegateVote*`. Certificate types: [`serialization/Certificate/Certificate.go`](../../serialization/Certificate/Certificate.go) (`Drep`, `VoteDelegCert`); builder: [`ApolloBuilder.go`](../../ApolloBuilder.go).

## DRep type

`Certificate.Drep`:

- `Code`: 0 = key hash, 1 = script hash, or special values for “always abstain” / “no confidence.”
- `Credential`: optional key or script hash (nil for abstain/no-confidence).

Build a `Drep` and pass it into the delegation methods.

## Purpose and method signatures

```go
func (b *Apollo) DelegateVote(stakeCredential *Certificate.Credential, drep *Certificate.Drep) (*Apollo, error)
func (b *Apollo) DelegateVoteFromAddress(address Address.Address, drep *Certificate.Drep) (*Apollo, error)
func (b *Apollo) DelegateVoteFromBech32(address string, drep *Certificate.Drep) (*Apollo, error)
```

If `stakeCredential` is `nil`, it is taken from the wallet. Appends a `VoteDelegCert` certificate (vote-only; stake key may already be registered).

## Inputs and constraints

- DRep key/script hash and special codes must match what the node expects.

## Cardano CLI equivalence (10.14.0.0)

| CLI | Apollo |
|-----|--------|
| Vote delegation cert | `DelegateVote(cred, drep)` / `DelegateVoteFromAddress(addr, drep)` / `DelegateVoteFromBech32(bech32, drep)` |

## Example

**Apollo:**

```go
drep := &Certificate.Drep{
    Code:       0,
    Credential: &serialization.ConstrainedBytes{Payload: drepKeyHash},
}
apollob, err = apollob.
    DelegateVote(nil, drep).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

**Cardano CLI:** Vote delegation certificate in `transaction build`.

## Evidence

- **Implementation-only**. Vote-only cert shape in `Certificate.go`; related round-trips for combined certs in `Certificate_test.go`.

## Caveats and validation

- No ApolloBuilder-level integration tests for vote delegation. Validate on preprod/mainnet with small amounts.
