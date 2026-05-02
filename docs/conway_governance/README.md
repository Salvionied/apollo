# Conway Governance in Apollo

This section documents how to build **Conway-era governance transactions** with the Apollo library: DRep registration/retirement/update, constitutional committee key authorization and resignation, casting votes on governance actions, submitting governance action proposals, and treasury donations. The APIs align with **cardano-cli** (10.14.0.0) `conway governance` commands and CIP-1694.

For **vote delegation** (delegating an existing stake key's voting power to a DRep), see [Staking Functionalities → Delegate Vote Methods](../staking_functionalities/delegate_vote_methods.md). This section covers the *other* side: becoming a DRep, voting as one, and creating proposals.

## Table of Contents

- [DRep Methods](drep_methods.md) — `RegisterDRep`, `RetireDRep`, `UpdateDRep`; deposit and anchor handling
- [Committee Methods](committee_methods.md) — `AuthorizeCommitteeHotKey`, `ResignCommitteeColdKey`; cold/hot key separation
- [Voting Methods](voting_methods.md) — `AddVote`; voter roles, votes (Yes/No/Abstain), per-vote anchors
- [Proposal Methods](proposal_methods.md) — `AddProposal`; the seven `GovAction` types (Info, NoConfidence, HardForkInitiation, ParameterChange, TreasuryWithdrawals, UpdateCommittee, NewConstitution)
- [Treasury Methods](treasury_methods.md) — `SetCurrentTreasuryValue`, `AddTreasuryDonation`

Each page includes method signatures, behavior, side-by-side Apollo + CLI examples, evidence labels (test functions verifying the behavior), and caveats.

## Capability Matrix

| Area | Apollo support | Notes |
|------|----------------|-------|
| DRep registration | Yes | `RegisterDRep(cred, coin, anchor)` (kind 16); deposit and refund handled in `Complete()` |
| DRep retirement | Yes | `RetireDRep(cred, coin)` (kind 17); refunds the original deposit |
| DRep update | Yes | `UpdateDRep(cred, anchor)` (kind 18); replaces the on-chain anchor |
| Committee hot-key authorization | Yes | `AuthorizeCommitteeHotKey(cold, hot)` (kind 14) |
| Committee cold-key resignation | Yes | `ResignCommitteeColdKey(cold, anchor)` (kind 15) |
| Casting votes | Yes | `AddVote(voter, actionId, procedure)`; auto-grouped per voter and deduped per action |
| Submitting proposals | Yes | `AddProposal(proposalProcedure)` covering all 7 `GovAction` kinds |
| Treasury donation | Yes | `AddTreasuryDonation(coin)` (TransactionBody field 22); accumulates across calls |
| Current treasury value pin | Yes | `SetCurrentTreasuryValue(coin)` (TransactionBody field 21) |
| Negative / overflow validation | Yes | Treasury setters reject negative values; donations detect `int64` overflow |

## CLI-to-Apollo Mapping Index

| Cardano CLI (10.14.0.0) | Apollo method / pattern |
|-------------------------|--------------------------|
| `conway governance drep registration-certificate` | `RegisterDRep(cred, coin, anchor)` |
| `conway governance drep retirement-certificate` | `RetireDRep(cred, coin)` |
| `conway governance drep update-certificate` | `UpdateDRep(cred, anchor)` |
| `conway governance committee hot-key-authorization-certificate` | `AuthorizeCommitteeHotKey(cold, hot)` |
| `conway governance committee cold-key-resignation-certificate` | `ResignCommitteeColdKey(cold, anchor)` |
| `conway governance vote create` | `AddVote(voter, actionId, procedure)` |
| `conway governance action create-info` | `AddProposal(...)` with `Governance.InfoAction{}` |
| `conway governance action create-no-confidence` | `AddProposal(...)` with `Governance.NoConfidence{...}` |
| `conway governance action create-hardfork` | `AddProposal(...)` with `Governance.HardForkInitiation{...}` |
| `conway governance action create-protocol-parameters-update` | `AddProposal(...)` with `Governance.ParameterChange{...}` |
| `conway governance action create-treasury-withdrawal` | `AddProposal(...)` with `Governance.TreasuryWithdrawals{...}` |
| `conway governance action update-committee` | `AddProposal(...)` with `Governance.UpdateCommittee{...}` |
| `conway governance action create-constitution` | `AddProposal(...)` with `Governance.NewConstitution{...}` |
| `transaction build --treasury-donation` | `AddTreasuryDonation(coin)` |
| `transaction build --current-treasury-value` | `SetCurrentTreasuryValue(coin)` |

## End-to-end example

A single transaction can combine multiple governance operations. The example below registers a DRep, votes on an existing action, submits a new info proposal, and donates to the treasury — all in one call to `Complete()`:

```go
import (
    "github.com/Salvionied/apollo"
    "github.com/Salvionied/apollo/serialization"
    "github.com/Salvionied/apollo/serialization/Certificate"
    "github.com/Salvionied/apollo/serialization/Governance"
)

drepCred := Certificate.Credential{
    Code: 0,
    Hash: serialization.ConstrainedBytes{Payload: drepKeyHash},
}

voter := Governance.Voter{
    Role: Governance.DRepKeyHash,
    Hash: serialization.ConstrainedBytes{Payload: drepKeyHash},
}

actionId := Governance.GovActionId{
    TransactionHash: existingActionTxHash,
    GovActionIndex:  0,
}

procedure := Governance.VotingProcedure{
    Vote:   Governance.VoteYes,
    Anchor: nil,
}

proposal := Governance.ProposalProcedure{
    Deposit:       100_000_000_000, // 100k ADA per Conway protocol params
    RewardAccount: rewardAccountBytes,
    Action:        Governance.InfoAction{},
    Anchor: Certificate.Anchor{
        Url:      "https://example.com/proposal.json",
        DataHash: proposalDocHash,
    },
}

apollob, err := apollob.
    RegisterDRep(drepCred, 500_000_000, &Certificate.Anchor{
        Url:      "https://example.com/drep.json",
        DataHash: drepDocHash,
    }).
    AddVote(voter, actionId, procedure).
    AddProposal(proposal).
    AddTreasuryDonation(5_000_000).
    AddInputAddressFromBech32(myAddr).
    AddLoadedUTxOs(utxos...).
    PayToAddressBech32(myAddr, 10_000_000).
    Complete()
```

`Complete()` accounts for the DRep deposit, the proposal deposit, and the treasury donation when balancing inputs and outputs.

## See also

- [Staking Functionalities](../staking_functionalities/README.md) — Vote delegation (`DelegateVote*`), stake registration, and combined certificates
- [Plutus V3 Support](../plutus_v3_support/README.md) — Plutus V3 scripts and reference inputs (relevant if using script-based DRep credentials)

## Reference

- [CIP-1694](https://cips.cardano.org/cip/CIP-1694) — On-chain governance for Cardano
- [Cardano CLI repository](https://github.com/IntersectMBO/cardano-cli) — CLI command reference (10.14.0.0)
- PR [#202](https://github.com/Salvionied/apollo/pull/202) — Conway-era governance support implementation
- Issue [#192](https://github.com/Salvionied/apollo/issues/192) — Governance functionality tracking
