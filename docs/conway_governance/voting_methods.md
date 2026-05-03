# Voting Methods

This page documents how to **cast votes** on governance actions: `AddVote`. Implementation: [`ApolloBuilder.go`](../../ApolloBuilder.go) (`AddVote`), [`serialization/Governance/Governance.go`](../../serialization/Governance/Governance.go) (`Voter`, `GovActionId`, `VotingProcedure`, `VotingProcedures`).

A vote is a tuple of **(voter, action ID, procedure)**. A single transaction can carry many votes from many voters on many actions; Apollo groups votes by voter and deduplicates per (voter, action) pair automatically.

## Types

### `Governance.Voter`

```go
type VoterRole int
const (
    ConstitutionalCommitteeKeyHash VoterRole = 0
    ConstitutionalCommitteeScript  VoterRole = 1
    DRepKeyHash                    VoterRole = 2
    DRepScript                     VoterRole = 3
    StakePoolOperator              VoterRole = 4
)

type Voter struct {
    Role VoterRole
    Hash serialization.ConstrainedBytes  // 28-byte credential hash
}
```

The role identifies *who* is voting; the hash is the credential. Constitutional committee members vote with their authorized **hot** credential (see [committee_methods.md](committee_methods.md)).

### `Governance.GovActionId`

```go
type GovActionId struct {
    TransactionHash []byte  // 32-byte hash of the transaction that proposed the action
    GovActionIndex  uint32  // index into that transaction's ProposalProcedures
}
```

A governance action is identified by the transaction that proposed it plus the index of the proposal within that transaction's proposals array.

### `Governance.VotingProcedure`

```go
type Vote int
const (
    VoteNo      Vote = 0
    VoteYes     Vote = 1
    VoteAbstain Vote = 2
)

type VotingProcedure struct {
    Vote   Vote
    Anchor *Certificate.Anchor  // optional rationale document
}
```

The optional anchor lets a voter publish a written rationale for their vote. Pass `nil` to vote without a rationale.

## Method signature

```go
func (b *Apollo) AddVote(
    voter Governance.Voter,
    actionId Governance.GovActionId,
    procedure Governance.VotingProcedure,
) *Apollo
```

Append-only; chainable. Internally calls `VotingProcedures.Add()` which:

- **Groups by voter**: votes from the same `Voter` (same role + same hash) are collected under one map entry.
- **Deduplicates per action**: calling `AddVote` again with the same voter and the same action **replaces** the previous procedure rather than adding a duplicate. This makes the call idempotent and lets you change your mind before sending the transaction.

## Behavior details

- The CBOR field on `TransactionBody` is map 19 (`{voter => {action_id => procedure}}`); see CIP-1694.
- Voting carries no deposit — only the standard transaction fee.
- The voter's credential must witness the transaction (key witness for `*KeyHash` roles, script witness for `*Script` roles, pool key for `StakePoolOperator`).

## Inputs and constraints

- `voter.Hash.Payload` must be exactly 28 bytes.
- `actionId.TransactionHash` must be exactly 32 bytes.
- `procedure.Anchor.DataHash` (when present) must be 32 bytes.

## Cardano CLI equivalence (10.14.0.0)

| CLI | Apollo |
|-----|--------|
| `conway governance vote create` | `AddVote(voter, actionId, procedure)` |

Note: cardano-cli emits a single vote file; Apollo equivalently allows multiple `AddVote` calls before `Complete()` to bundle many votes into one transaction.

## Examples

### Single DRep vote

**Apollo:**

```go
import (
    "github.com/Salvionied/apollo/serialization"
    "github.com/Salvionied/apollo/serialization/Governance"
)

voter := Governance.Voter{
    Role: Governance.DRepKeyHash,
    Hash: serialization.ConstrainedBytes{Payload: drepKeyHash},
}

actionId := Governance.GovActionId{
    TransactionHash: proposalTxHash,
    GovActionIndex:  0,
}

procedure := Governance.VotingProcedure{
    Vote:   Governance.VoteYes,
    Anchor: nil,
}

apollob, err = apollob.
    AddVote(voter, actionId, procedure).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli conway governance vote create \
  --yes \
  --drep-verification-key-file drep.vkey \
  --governance-action-tx-id <hex> \
  --governance-action-index 0 \
  --out-file vote.cert

cardano-cli conway transaction build --vote-file vote.cert ...
```

### Voting Yes/No/Abstain on multiple actions in one transaction

The same voter can vote on several different actions:

```go
apollob, err = apollob.
    AddVote(voter, action1, Governance.VotingProcedure{Vote: Governance.VoteYes}).
    AddVote(voter, action2, Governance.VotingProcedure{Vote: Governance.VoteNo}).
    AddVote(voter, action3, Governance.VotingProcedure{Vote: Governance.VoteAbstain}).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

All three votes are stored under the single voter entry in the transaction's voting procedures map.

### Vote with a rationale anchor

```go
import "github.com/Salvionied/apollo/serialization/Certificate"

procedure := Governance.VotingProcedure{
    Vote: Governance.VoteNo,
    Anchor: &Certificate.Anchor{
        Url:      "https://example.com/vote-rationale.json",
        DataHash: rationaleDocHash,
    },
}

apollob, err = apollob.
    AddVote(voter, actionId, procedure).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

### Constitutional committee member voting with hot key

```go
voter := Governance.Voter{
    Role: Governance.ConstitutionalCommitteeKeyHash,
    Hash: serialization.ConstrainedBytes{Payload: ccHotKeyHash},
}

apollob, err = apollob.
    AddVote(voter, actionId, Governance.VotingProcedure{
        Vote: Governance.VoteYes,
    }).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

Use the **hot** key hash here — the cold key authorized this hot key via `AuthorizeCommitteeHotKey`.

### Stake pool operator voting

SPOs vote on no-confidence, hard fork, and a few other action types. The hash is the pool's cold key hash:

```go
voter := Governance.Voter{
    Role: Governance.StakePoolOperator,
    Hash: serialization.ConstrainedBytes{Payload: poolColdKeyHash},
}

apollob, err = apollob.
    AddVote(voter, actionId, Governance.VotingProcedure{
        Vote: Governance.VoteYes,
    }).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

### Changing your vote before sending

`AddVote` deduplicates per (voter, action). The last call wins for any given pair:

```go
apollob = apollob.
    AddVote(voter, actionId, Governance.VotingProcedure{Vote: Governance.VoteYes}).
    AddVote(voter, actionId, Governance.VotingProcedure{Vote: Governance.VoteNo})
// The transaction sends only one vote: VoteNo.
```

## Evidence

- **Builder behavior verified by tests**:
  - `TestAddVote` ([`governance_test.go`](../../governance_test.go)) — single vote with empty hashes; voting procedures map populated, vote value preserved.
  - `TestAddMultipleVotes` ([`governance_test.go`](../../governance_test.go)) — same voter on two different actions grouped under one voter entry.
  - `TestVotingAndProposalFieldsNilByDefault` ([`governance_test.go`](../../governance_test.go)) — voting procedures are `nil` until `AddVote` is called.
- **Voting procedure data structures verified by tests**:
  - `TestVoterRoundTrip`, `TestGovActionIdRoundTrip`, `TestVotingProcedureRoundTrip`, `TestVotingProceduresRoundTrip` ([`serialization/Governance/Governance_test.go`](../../serialization/Governance/Governance_test.go)).
  - `TestVotingProceduresAdd` — append a new voter and a second vote for an existing voter.
  - `TestVotingProceduresAddReplacesDuplicateAction` — confirms (voter, action) deduplication.
  - `TestVotingProceduresMarshalCBORCanonicalOrder` — outer and inner map keys emitted in canonical CBOR order, required for deterministic transaction hashing.
  - `TestVotingProcedures_Empty` — empty voting procedures encodes/decodes correctly.

## Caveats and validation

- **Witness requirement**: every distinct `Voter` adds a witness requirement to the transaction. Forgetting to sign with the matching key (or include the matching script witness) will cause the node to reject the transaction.
- **Role/hash type must match**: a `DRepKeyHash` voter with a script-hash payload, or a `ConstitutionalCommitteeScript` with a key-hash payload, will be rejected on submission.
- **Per-action role restrictions** apply at the ledger level (e.g. SPOs cannot vote on parameter changes affecting only DReps). Apollo does not check these — see CIP-1694 for the role/action matrix.
- **Action must exist** at submission time. If the referenced governance action has already been ratified, expired, or never existed, the node will reject the vote.
- **Canonical CBOR ordering** of the outer voter map and inner action map is enforced by Apollo's marshaler — required so two parties building the same vote bundle produce the same transaction hash.
- Validate on preview/preprod first.
