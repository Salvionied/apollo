# Delegate Stake Methods

This page documents **stake pool delegation** and **stake+vote delegation**: `DelegateStake`*, `DelegateStakeAndVote*`. Implementation: `[ApolloBuilder.go](../../ApolloBuilder.go)`, `[serialization/Certificate/Certificate.go](../../serialization/Certificate/Certificate.go)`.

## Purpose and method signatures

### DelegateStake / DelegateStakeFromAddress / DelegateStakeFromBech32

```go
func (b *Apollo) DelegateStake(stakeCredential *Certificate.Credential, poolKeyHash serialization.PubKeyHash) (*Apollo, error)
func (b *Apollo) DelegateStakeFromAddress(address Address.Address, poolKeyHash serialization.PubKeyHash) (*Apollo, error)
func (b *Apollo) DelegateStakeFromBech32(address string, poolKeyHash serialization.PubKeyHash) (*Apollo, error)
```

If `stakeCredential` is `nil`, it is taken from the wallet. `poolKeyHash` is the pool’s cold key hash (28 bytes). Appends a `StakeDelegation` certificate.

### DelegateStakeAndVote / DelegateStakeAndVoteFromAddress / DelegateStakeAndVoteFromBech32

```go
func (b *Apollo) DelegateStakeAndVote(stakeCredential *Certificate.Credential, poolKeyHash serialization.PubKeyHash, drep *Certificate.Drep) (*Apollo, error)
// ... FromAddress(address, poolKeyHash, drep), FromBech32(bech32, poolKeyHash, drep)
```

Appends a `StakeVoteDelegCert` (kind 10): delegates both stake (to pool) and vote (to DRep) in one certificate.

## Inputs and constraints

- `poolKeyHash`: 28-byte cold key hash of the target stake pool.
- `drep`: `Certificate.Drep` (Code + optional Credential); build it for the target DRep.

## Cardano CLI equivalence (10.14.0.0)

| CLI                                    | Apollo                                               |
| -------------------------------------- | ---------------------------------------------------- |
| `stake-address delegation-certificate` | `DelegateStake(cred, poolKeyHash)` etc.              |
| Stake + vote delegation (one cert)     | `DelegateStakeAndVote(cred, poolKeyHash, drep)` etc. |

## Examples

### Stake pool delegation

**Apollo:**

```go
var poolKeyHash serialization.PubKeyHash
apollob, err = apollob.
    DelegateStake(nil, poolKeyHash).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli stake-address delegation-certificate --stake-verification-key-file stake.vkey --cold-verification-key-file pool.vkey --out-file deleg.cert
cardano-cli transaction build --certificate-file deleg.cert ...
```

### Stake and vote (one cert)

**Apollo:**

```go
drep := &Certificate.Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: drepKeyHash}}
apollob, err = apollob.DelegateStakeAndVote(nil, poolKeyHash, drep)
```

## Evidence

- **Implementation-only**. Certificate round-trips: `TestStakeDelegationRoundTrip`, `TestStakeVoteDelegCertRoundTrip` in `Certificate_test.go`.

## Caveats and validation

- Pool key hash must be the correct cold key hash. DRep must match node expectations. Validate on preprod/mainnet with small amounts.
