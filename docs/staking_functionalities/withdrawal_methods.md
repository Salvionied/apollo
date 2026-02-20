# Withdrawal Methods

This page documents **reward withdrawals**: `AddWithdrawal`. Implementation: `[ApolloBuilder.go](../../ApolloBuilder.go)`. Withdrawal and redeemer handling are exercised in backend tests.

## Purpose and method signature

```go
func (b *Apollo) AddWithdrawal(
    address Address.Address,
    amount int,
    redeemerData PlutusData.PlutusData,
) *Apollo
```

- **address**: Must have a valid **staking part** (28 bytes), e.g. stake/reward address or base address. Converted internally to 29-byte form (header + 28-byte staking credential).
- **amount**: Withdrawal amount in lovelace.
- **redeemerData**: Optional. If provided (e.g. not empty `PlutusData{}`), a redeemer with `Tag = REWARD` is created and associated with this withdrawal; execution units are filled during evaluation.

## Inputs and constraints

- Address must have 28-byte `StakingPart`. Enterprise addresses are not valid for withdrawals.
- Redeemer from file/JSON: read and parse to `PlutusData.PlutusData` in your application; Apollo does not accept file paths.

## Behavior details

- Withdrawals are stored in the builder’s withdrawal map and included in the transaction body by `buildTxBody()`. Fee estimation in `Complete()` accounts for withdrawal amounts.
- If `withdrawals` is nil, it is created on first `AddWithdrawal`. When redeemer data is provided, a `Redeemer` with `Tag = Redeemer.REWARD` and the corresponding index is stored in `stakeRedeemers` and merged into the witness set; execution units are filled when estimation is enabled.

## Cardano CLI equivalence (10.14.0.0)


| CLI                                                        | Apollo                                                                                  |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `transaction build --withdrawal STAKE_ADDRESS+AMOUNT`      | `AddWithdrawal(decodedStakeAddress, amount, PlutusData.PlutusData{})`                   |
| `--withdrawal` + `--withdrawal-reference-tx-in-redeemer-`* | Read/decode redeemer to `PlutusData`, then `AddWithdrawal(addr, amount, redeemerData)`. |


**Parity:** Full (manual file/value for redeemer).

## Examples

### Withdrawal (no redeemer)

**Apollo:**

```go
apollob := apollo.New(cc)
apollob, err = apollob.
    AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
    AddWithdrawal(decoded_addr_for_fixtures, 1_000_000, PlutusData.PlutusData{}).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli transaction build --withdrawal "stake1u...+1000000" ...
```

### Withdrawal with redeemer

**Apollo:**

```go
var redeemerData PlutusData.PlutusData
apollob = apollob.AddWithdrawal(stakeAddr, 1_000_000, redeemerData)
```

**Cardano CLI:**

```bash
cardano-cli transaction build --withdrawal "stake1u...+1000000" --withdrawal-reference-tx-in-redeemer-value '...' ...
```

## Evidence

- **Verified by tests:** `TestUTXORPC_TransactionWithWithdrawals` in `UtxorpcChainContext_test.go`, `TestOGMIOS_TransactionWithWithdrawals` in `OgmiosChainContext_test.go`. Withdrawal with redeemer: implementation (stake redeemers merged and evaluated; file/JSON loading is app responsibility).

## Caveats and validation

- Withdrawals with redeemers require evaluation so execution units can be filled; the builder’s evaluation flow handles this when estimation is enabled.
