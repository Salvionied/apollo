# Register Methods

This page documents **stake registration** and **register+delegate** certificate methods: `RegisterStake`*, `RegisterAndDelegateStake`*, `RegisterAndDelegateVote*`, `RegisterAndDelegateStakeAndVote*`. Implementation: `[ApolloBuilder.go](../../ApolloBuilder.go)`, `[serialization/Certificate/Certificate.go](../../serialization/Certificate/Certificate.go)`. Certificate CBOR: `[serialization/Certificate/Certificate_test.go](../../serialization/Certificate/Certificate_test.go)`.

## Constants

- **Stake key deposit**: 2 ADA (`2_000_000` lovelace), `STAKE_DEPOSIT` in `ApolloBuilder.go`. `Complete()` adds it to the required balance when registration certificates are present.

## Purpose and method signatures

### RegisterStake / RegisterStakeFromAddress / RegisterStakeFromBech32

```go
func (b *Apollo) RegisterStake(stakeCredential *Certificate.Credential) (*Apollo, error)
func (b *Apollo) RegisterStakeFromAddress(address Address.Address) (*Apollo, error)
func (b *Apollo) RegisterStakeFromBech32(address string) (*Apollo, error)
```

If `stakeCredential` is `nil` in `RegisterStake`, the credential is taken from the wallet. Each appends a `StakeRegistration` certificate.

### RegisterAndDelegateStake / RegisterAndDelegateStakeFromAddress / RegisterAndDelegateStakeFromBech32

```go
func (b *Apollo) RegisterAndDelegateStake(stakeCredential *Certificate.Credential, poolKeyHash serialization.PubKeyHash, coin int64) (*Apollo, error)
// ... FromAddress(address, poolKeyHash, coin), FromBech32(bech32, poolKeyHash, coin)
```

`coin` is the deposit (typically `STAKE_DEPOSIT`). Appends a `StakeRegDelegCert` certificate.

### RegisterAndDelegateVote / RegisterAndDelegateVoteFromAddress / RegisterAndDelegateVoteFromBech32

```go
func (b *Apollo) RegisterAndDelegateVote(stakeCredential *Certificate.Credential, drep *Certificate.Drep, coin int64) (*Apollo, error)
// ... FromAddress(address, drep, coin), FromBech32(bech32, drep, coin)
```

Appends a `VoteRegDelegCert` (kind 12). Deposit required.

### RegisterAndDelegateStakeAndVote / …FromAddress / …FromBech32

```go
func (b *Apollo) RegisterAndDelegateStakeAndVote(stakeCredential *Certificate.Credential, poolKeyHash serialization.PubKeyHash, drep *Certificate.Drep, coin int64) (*Apollo, error)
```

Appends a `StakeVoteRegDelegCert` (kind 13). Deposit required.

## Inputs and constraints

- Ensure inputs cover at least 2 ADA deposit when registering (or register+delegate). `Complete()` adds the deposit automatically.
- Pool key hash must be the correct 28-byte cold key hash for the target pool. DRep must be built as `Certificate.Drep` (code + credential).

## Behavior details

- Apollo’s staking methods **append** certificates to the builder’s certificate list. The low-level `SetCertificates(c *Certificate.Certificates)` is available to set the full list at once if needed.

## Cardano CLI equivalence (10.14.0.0)

| CLI                                      | Apollo                                                                   |
| ---------------------------------------- | ------------------------------------------------------------------------ |
| `stake-address registration-certificate` | `RegisterStake` / `RegisterStakeFromAddress` / `RegisterStakeFromBech32` |
| Reg + deleg (combined cert)              | `RegisterAndDelegateStake(cred, poolKeyHash, STAKE_DEPOSIT)` etc.        |
| Reg + vote delegation                    | `RegisterAndDelegateVote(cred, drep, coin)` etc.                         |
| Reg + stake + vote                       | `RegisterAndDelegateStakeAndVote(cred, poolKeyHash, drep, coin)` etc.    |

## Examples

### Stake registration

**Apollo:**

```go
apollob, err = apollob.
    RegisterStake(nil).  // nil => use wallet credential
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli stake-address registration-certificate --stake-verification-key-file stake.vkey --out-file reg.cert
cardano-cli transaction build --certificate-file reg.cert ...
```

### Register and delegate in one transaction

**Apollo:**

```go
cred, _ := apollob.GetStakeCredentialFromWallet()
apollob, err = apollob.
    RegisterAndDelegateStake(cred, poolKeyHash, apollo.STAKE_DEPOSIT).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 12_000_000).
    Complete()
```

**Cardano CLI:** Combine registration cert and delegation cert (or use combined reg+deleg where supported).

## Evidence

- **Implementation-only** Certificate round-trips: `TestStakeRegistrationRoundTrip`, `TestStakeRegDelegCertRoundTrip`, `TestStakeVoteRegDelegCertRoundTrip` in `Certificate_test.go`; vote reg cert shapes in `Certificate.go`.

## Caveats and validation

- Ensure enough inputs for the 2 ADA deposit. Pool key hash and DRep must match node expectations.
- Validate on preprod/mainnet with small amounts before production use.
