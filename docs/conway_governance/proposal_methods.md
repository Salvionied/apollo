# Proposal Methods

This page documents how to **submit governance action proposals**: `AddProposal`, plus the seven `GovAction` types. Implementation: [`ApolloBuilder.go`](../../ApolloBuilder.go) (`AddProposal`), [`serialization/Governance/Governance.go`](../../serialization/Governance/Governance.go) (`ProposalProcedure`, `GovAction`, all action types).

A **proposal** bundles a deposit, a return address for the deposit refund, the action being proposed, and an anchor pointing to the rationale. A single transaction can carry multiple proposals; each one independently locks its deposit until the action is ratified, expired, or dropped (at which point the deposit is returned to the reward account).

## Method signature

```go
func (b *Apollo) AddProposal(
    proposal Governance.ProposalProcedure,
) *Apollo
```

Append-only; chainable. The proposal is appended to `TransactionBody` field 20 (`ProposalProcedures`). `Complete()` adds each `proposal.Deposit` to the required input balance.

## ProposalProcedure

```go
type ProposalProcedure struct {
    Deposit       int64               // governance action deposit (per protocol params)
    RewardAccount []byte              // 29-byte stake address bytes for deposit refund
    Action        GovAction           // one of the seven concrete action types
    Anchor        Certificate.Anchor  // rationale document (URL + 32-byte hash)
}
```

The `RewardAccount` is the 29-byte raw form of a stake address (network-prefix byte + 28-byte stake credential hash). The deposit is refunded to this account when the action concludes, regardless of outcome.

## The seven GovAction types

All seven implement the `Governance.GovAction` interface and are accepted by `ProposalProcedure.Action`. They map 1:1 to `cardano-cli conway governance action create-*` commands.

| Type | Kind | Purpose |
|---|---|---|
| `InfoAction` | 6 | Non-binding informational proposal — captures community sentiment with no on-chain effect |
| `NoConfidence` | 3 | Replace the current constitutional committee with a "no confidence" state |
| `HardForkInitiation` | 1 | Move the network to a new major protocol version |
| `ParameterChange` | 0 | Update one or more protocol parameters |
| `TreasuryWithdrawals` | 2 | Withdraw funds from the treasury to one or more reward accounts |
| `UpdateCommittee` | 4 | Add and/or remove constitutional committee members and adjust quorum |
| `NewConstitution` | 5 | Replace the on-chain constitution (anchor + optional guardrails script hash) |

Many actions take an optional `PrevActionId *GovActionId` linking to the most recent ratified action of the same type (or `nil` if none). This forms a linear "lineage" enforced by the ledger to prevent stale proposals from overwriting newer ones.

### `InfoAction`

```go
type InfoAction struct{}
```

No fields. Use it to put a question or position to the governance bodies for a non-binding vote.

### `NoConfidence`

```go
type NoConfidence struct {
    PrevActionId *GovActionId
}
```

### `HardForkInitiation`

```go
type ProtocolVersion struct {
    Major uint64
    Minor uint64
}

type HardForkInitiation struct {
    PrevActionId    *GovActionId
    ProtocolVersion ProtocolVersion
}
```

### `ParameterChange`

```go
type ParameterChange struct {
    PrevActionId *GovActionId
    ParamUpdate  map[int]any   // protocol-parameter index → new value
}
```

`ParamUpdate` keys are the integer indices defined by the Cardano ledger spec (e.g. `0` = `minfeeA`, `1` = `minfeeB`). Apollo encodes the map with canonical CBOR ordering.

### `TreasuryWithdrawals`

```go
type Withdrawal struct {
    RewardAccount []byte  // 29-byte stake address bytes
    Coin          int64
}

type TreasuryWithdrawals struct {
    Withdrawals  []Withdrawal
    PrevActionId *GovActionId
}
```

### `UpdateCommittee`

```go
type AddedCommitteeMember struct {
    Credential Certificate.Credential
    Epoch      uint64   // expiry epoch for this member
}

type UpdateCommittee struct {
    PrevActionId *GovActionId
    Removed      []Certificate.Credential   // members to remove
    Added        []AddedCommitteeMember     // members to add (with expiry)
    Quorum       Certificate.UnitInterval   // new quorum fraction
}
```

### `NewConstitution`

```go
type NewConstitution struct {
    PrevActionId *GovActionId
    Anchor       Certificate.Anchor  // points to the constitution document
    ScriptHash   []byte              // optional Plutus guardrails script hash; nil for none
}
```

## Inputs and constraints

- `Deposit` must equal the protocol-parameter `govActionDeposit` (typically 100,000 ADA on mainnet).
- `RewardAccount` must be exactly 29 bytes and correspond to a registered stake key — refunds are credited there.
- `Anchor.DataHash` must be 32 bytes.
- For action types with a `PrevActionId`: pass `nil` for the first action of that type, or the `GovActionId` of the most recently ratified action of the same type otherwise.

## Cardano CLI equivalence (10.14.0.0)

| CLI | Apollo |
|-----|--------|
| `conway governance action create-info` | `AddProposal(...)` with `InfoAction{}` |
| `conway governance action create-no-confidence` | `AddProposal(...)` with `NoConfidence{...}` |
| `conway governance action create-hardfork` | `AddProposal(...)` with `HardForkInitiation{...}` |
| `conway governance action create-protocol-parameters-update` | `AddProposal(...)` with `ParameterChange{...}` |
| `conway governance action create-treasury-withdrawal` | `AddProposal(...)` with `TreasuryWithdrawals{...}` |
| `conway governance action update-committee` | `AddProposal(...)` with `UpdateCommittee{...}` |
| `conway governance action create-constitution` | `AddProposal(...)` with `NewConstitution{...}` |

## Examples

### Info proposal

**Apollo:**

```go
import (
    "github.com/Salvionied/apollo/serialization/Certificate"
    "github.com/Salvionied/apollo/serialization/Governance"
)

proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,  // 29 bytes
    Action:        Governance.InfoAction{},
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/info.json",
        DataHash: docHash,
    },
}

apollob, err = apollob.
    AddProposal(proposal).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli conway governance action create-info \
  --mainnet \
  --governance-action-deposit 100000000000 \
  --deposit-return-stake-verification-key-file stake.vkey \
  --anchor-url https://example.com/info.json \
  --anchor-data-hash <hex-hash> \
  --out-file info.action

cardano-cli conway transaction build --proposal-file info.action ...
```

### Hard fork initiation

```go
proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,
    Action: Governance.HardForkInitiation{
        PrevActionId: prevHardForkId,  // or nil for the first one
        ProtocolVersion: Governance.ProtocolVersion{
            Major: 11,
            Minor: 0,
        },
    },
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/hardfork.json",
        DataHash: docHash,
    },
}

apollob, err = apollob.AddProposal(proposal). /* ... */ Complete()
```

### Protocol parameter change

```go
proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,
    Action: Governance.ParameterChange{
        PrevActionId: nil,
        ParamUpdate: map[int]any{
            0: uint64(50),     // minfeeA
            1: uint64(180_000), // minfeeB
        },
    },
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/pparams.json",
        DataHash: docHash,
    },
}

apollob, err = apollob.AddProposal(proposal). /* ... */ Complete()
```

### Treasury withdrawal

```go
proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,
    Action: Governance.TreasuryWithdrawals{
        Withdrawals: []Governance.Withdrawal{
            {
                RewardAccount: recipient1Bytes,
                Coin:          50_000_000_000,
            },
            {
                RewardAccount: recipient2Bytes,
                Coin:          25_000_000_000,
            },
        },
        PrevActionId: nil,
    },
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/treasury-wd.json",
        DataHash: docHash,
    },
}

apollob, err = apollob.AddProposal(proposal). /* ... */ Complete()
```

### Update committee

```go
newMember := Certificate.Credential{
    Code: 0,
    Hash: serialization.ConstrainedBytes{Payload: ccColdKeyHash},
}

proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,
    Action: Governance.UpdateCommittee{
        PrevActionId: prevCommitteeUpdateId,  // or nil for the first one
        Removed: []Certificate.Credential{
            // existing member to remove
        },
        Added: []Governance.AddedCommitteeMember{
            {Credential: newMember, Epoch: 600},
        },
        Quorum: Certificate.UnitInterval{Num: 2, Den: 3},
    },
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/committee.json",
        DataHash: docHash,
    },
}

apollob, err = apollob.AddProposal(proposal). /* ... */ Complete()
```

### New constitution

```go
proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,
    Action: Governance.NewConstitution{
        PrevActionId: prevConstitutionId,  // or nil for the first one
        Anchor: Certificate.Anchor{
            Url:      "https://example.com/constitution.txt",
            DataHash: constitutionHash,
        },
        ScriptHash: guardrailsScriptHash,  // or nil for no guardrails
    },
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/proposal.json",
        DataHash: proposalDocHash,
    },
}

apollob, err = apollob.AddProposal(proposal). /* ... */ Complete()
```

### No confidence

```go
proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000,
    RewardAccount: rewardAccountBytes,
    Action: Governance.NoConfidence{
        PrevActionId: prevNoConfidenceId,  // or nil for the first one
    },
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/no-confidence.json",
        DataHash: docHash,
    },
}

apollob, err = apollob.AddProposal(proposal). /* ... */ Complete()
```

### Multiple proposals in one transaction

```go
apollob, err = apollob.
    AddProposal(infoProposal).
    AddProposal(parameterChangeProposal).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

`Complete()` adds the **sum** of both deposits to the required input balance.

## Evidence

- **Builder behavior verified by tests**:
  - `TestAddProposal` ([`governance_test.go`](../../governance_test.go)) — single info proposal; `ProposalProcedures` populated, deposit preserved.
  - `TestAddMultipleProposals` — two proposals (info + no-confidence) in one transaction; both deposits preserved in order.
  - `TestVotingAndProposalFieldsNilByDefault` — `ProposalProcedures` is `nil` until `AddProposal` is called.
- **GovAction CBOR round-trips** ([`serialization/Governance/Governance_test.go`](../../serialization/Governance/Governance_test.go)):
  - `TestInfoActionRoundTrip` (kind 6)
  - `TestNoConfidenceRoundTrip` (kind 3) — with and without `PrevActionId`
  - `TestHardForkInitiationRoundTrip` (kind 1)
  - `TestParameterChangeRoundTrip` (kind 0) — confirms canonical CBOR map ordering for `ParamUpdate`
  - `TestTreasuryWithdrawalsRoundTrip` (kind 2)
  - `TestUpdateCommitteeRoundTrip` (kind 4) and `TestUpdateCommitteeRoundTrip_EmptyMembers`
  - `TestNewConstitutionRoundTrip` (kind 5) — with and without script hash
  - `TestUnmarshalGovAction` — full dispatch table; `TestUnmarshalGovAction_Errors` covers malformed input.
- **ProposalProcedure CBOR round-trips**:
  - `TestProposalProcedureRoundTrip` — single proposal with each of the seven actions.
  - `TestProposalProceduresRoundTrip` — array form.
  - `TestProposalProcedureWithUpdateCommittee` — focused round-trip including credential keys in maps.
  - `TestProposalProcedureMarshalCBORNilAction` — confirms `nil` action returns an error rather than panicking.
  - `TestProposalProcedures_Empty` — empty proposals collection encodes/decodes correctly.

## Caveats and validation

- **Deposit must match `govActionDeposit`** at submission time. Apollo does not query the protocol parameters — pass the value applicable to your target network.
- **Anchor URL is *not* dereferenced** by Apollo or the node — only the on-chain hash is verified.
- `RewardAccount` must be a registered stake address; the refund will be credited there even years later when the action concludes.
- For `ParameterChange`, the keys in `ParamUpdate` must be valid protocol-parameter indices for the current era. The node rejects unknown indices.
- For `UpdateCommittee` and `TreasuryWithdrawals`, the totals (added/removed members, withdrawn coin) must satisfy ledger constraints — e.g. cannot withdraw more than the current treasury value, cannot reduce committee below threshold.
- **`PrevActionId` discipline**: most action types require linking to the most recently ratified action of the same kind. An out-of-date `PrevActionId` causes the node to reject the proposal. Query the chain for the current head before constructing a proposal.
- Validate on preview/preprod first — these networks have lower deposits and faster epochs to make iteration practical.
