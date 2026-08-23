# Apollo Repository Working Guide

This is the shared working agreement for coding agents, reviewers, and human
contributors to Apollo v2. Use it together with the task description and the
code itself. If guidance conflicts, the more specific instruction wins; the
current source, tests, `go.mod`, and CI workflows are authoritative for facts
that may have changed.

## Start Here

Before editing or reviewing:

1. Run `git status --short` and preserve unrelated tracked and untracked work.
2. Read the relevant implementation, its tests, and any package documentation.
3. Identify the contract being changed: public Go API, transaction behavior,
   CBOR or Plutus encoding, backend-provider behavior, wallet/signing behavior,
   or documentation.
4. Search for every caller and implementation of an affected type or method.
   Prefer `rg` and `rg --files` for repository discovery.
5. Choose the smallest focused test that can demonstrate the intended behavior
   before changing code.

Keep changes scoped to the request. Do not discard, rewrite, or format unrelated
user work. Do not commit scratch plans, agent state, credentials, coverage
artifacts, or temporary files.

## Project Overview

Apollo v2 is a pure Go Cardano transaction-building library with module path:

```text
github.com/Salvionied/apollo/v2
```

The repository has one Go module. Its current hard toolchain floor is Go
1.25.13, as declared by `go.mod`; an older patch release will refuse to build
the module. Keep the version in `go.mod`, `README.md`, `CONTRIBUTING.md`, CI,
and this guide synchronized. Raise it only when the source or dependency graph
requires the newer toolchain.

Apollo uses these Blink Labs packages as the source of core behavior:

- `github.com/blinklabs-io/gouroboros` for ledger types, CBOR, scripts,
  addresses, transactions, certificates, and governance types
- `github.com/blinklabs-io/bursa` for HD wallet key derivation
- `github.com/blinklabs-io/plutigo` for Plutus data encoding and decoding

Use the canonical upstream module for every dependency. Do not introduce local
replacements or substitute forks merely to make a build pass.

## Repository Map

### Core entry points

| Path | Responsibility |
| --- | --- |
| `apollo.go` | `Apollo` transaction builder, fluent API, balancing, fees, evaluation, signing, and submission |
| `models.go` | `Unit`, `Payment`, `PaymentI`, and value conversion |
| `helpers.go` | value, script, CBOR, and witness helpers |
| `coinselection.go`, `macs.go` | coin-selection interface and implementations |
| `evaluation_*.go` | execution-unit evaluation and temporary evaluation witnesses |
| `wallet.go` | wallet interfaces and Bursa, key-pair, and external-wallet adapters |
| `convenience.go` | Bech32 and script-specific convenience wrappers |
| `metadata_json.go` | transaction metadata JSON conversion |

### Packages

| Path | Responsibility |
| --- | --- |
| `backend/` | `ChainContext`, capability reporting, and common backend types |
| `backend/blockfrost/` | Blockfrost adapter |
| `backend/ogmios/` | Ogmios and Kupo adapter |
| `backend/utxorpc/` | UTxO RPC adapter |
| `backend/fixed/` | deterministic in-memory backend for tests |
| `backend/cache/` | cached `ChainContext` wrapper |
| `internal/backendutil/` | internal provider-response parsing helpers |
| `plutusencoder/` | struct-tag-driven Plutus data marshaling |
| `constants/` | network constants |
| `docs/` | version-specific feature and migration documentation |

Apollo v2 does not contain the v1 `serialization/`, `txBuilding/`, or `crypto/`
package trees. Do not recreate those packages or cite them in new guidance.

## Important Contracts

### Chain contexts

`backend.ChainContext` supplies protocol and genesis parameters, network and
chain position, UTxO lookup, transaction evaluation and submission, and
reference-script lookup. It is a public interface implemented outside this
repository, so adding a method is source-breaking.

Optional behavior belongs behind a focused extension interface or the existing
`CapabilityReporter` mechanism where possible. Code that depends on evaluator-
supplied UTxOs must check
`CapabilityEvaluateTxAdditionalUtxos`. An unsupported local operation should
return `backend.NewUnsupportedError`; callers can classify it with
`errors.Is(err, backend.ErrUnsupported)`.

When a change really must alter `ChainContext`:

- account for every backend, the cache wrapper, and `backend/fixed`;
- search for third-party-facing constructors and compile-time assertions;
- document the compatibility impact; and
- add deterministic tests for supported and unsupported behavior.

Backend tests must not depend on live provider availability. Use fixtures and
`backend/fixed.FixedChainContext`; cover malformed, missing, null, out-of-range,
and provider-error responses as applicable. Never silently treat malformed
provider data as a zero value or successful empty response.

### Wallets and signing

The `Wallet` interface exposes `Address`, `SignTxBody`, `PubKeyHash`, and
`StakePubKeyHash`. `EvaluationWitnessProvider` is a separate optional contract
for witnesses used only during execution-unit evaluation.

Treat mnemonic phrases, passphrases, signing keys, raw witnesses, provider
tokens, and transaction secrets as sensitive. Never log or commit them. Signing
and derivation changes must fail closed, verify that keys control the expected
credentials, and test both payment and stake behavior. Evaluation-only
witnesses must not leak into the final unsigned transaction.

### Fluent builder errors

Methods that can return an error should return it with useful context. Fluent
methods whose signature cannot return an error must call `setErrOnce`; the
first stored error is returned by `Complete`. Never panic in library code,
silently ignore an error, or overwrite an earlier builder error.

### CBOR, ledger, and Plutus data

- Prefer gouroboros `ledger/common` and era-specific types over local copies.
- Preserve Cardano wire semantics, including map ordering and required
  definite or indefinite encodings.
- Use preserved decoded CBOR bytes when a hash or exact wire round trip depends
  on them. Clear preserved bytes before marshaling an intentionally mutated
  decoded object.
- Keep map, set, witness, mint-policy, withdrawal, and redeemer ordering
  deterministic.
- Cover new encoders and decoders with round-trip, known-vector, malformed-
  input, boundary, and determinism tests.
- Treat receiver changes on serialization methods as API changes: both value
  and pointer forms can be observed through `json` or `cbor` interfaces.
- Guard typed nils held in interfaces before dereferencing them.

Plutus struct tags include `Bytes`, `Int`, `BigInt`, `Map`, `IndefList`,
`DefList`, `StringBytes`, `HexString`, `Bool`, and `IndefBool`. A constructor
can be declared with a blank field:

```go
type Datum struct {
    _      struct{} `plutusType:"IndefList" plutusConstr:"1"`
    Pkh    []byte   `plutusType:"Bytes"`
    Amount int64    `plutusType:"Int"`
}
```

### Values, fees, and portability

Cardano amounts, fees, deposits, execution units, and indexes cross signed,
unsigned, and architecture-sized integer boundaries. Reject negatives where
the ledger requires non-negative values, check narrowing conversions and
overflow, and avoid results that differ between 64-bit and 32-bit builds. CI
vets and tests `linux/386` specifically to catch these issues.

## Implementation Workflow

For fixes and new behavior:

1. Reproduce the behavior or turn the acceptance criterion into a focused
   test. For a regression, confirm that the test fails for the expected reason
   without the fix, not because it fails to compile or set up.
2. Trace the full boundary. A response shape, public method, stored value,
   CBOR representation, or interface change must include all producers,
   consumers, wrappers, fixtures, examples, and documentation.
3. Implement the smallest coherent change. Reuse existing types and helpers
   before introducing new abstractions.
4. Test the negative path as well as the happy path. New exported behavior
   requires tests and a doc comment.
5. Run focused validation first, then the repository-wide checks appropriate
   to the change.
6. Inspect `git diff --check`, `git diff --stat`, and the complete diff. Confirm
   that generated or module files changed only when their source inputs did.

Do not change behavior during a code-review-only task unless the requester also
asks for implementation. Do not use a dependency bump, formatter sweep, or
unrelated cleanup to hide the actual change.

## Build and Validation

Repository targets:

```bash
make mod-tidy    # run go mod tidy; review go.mod and go.sum afterward
make format      # go fmt plus gofmt -s across Go files
make golines     # optional 80-column mechanical formatting
make test        # go test -v -race ./...
make clean       # remove files under tmp/
```

Useful focused and CI-parity checks:

```bash
go test -v -race -run '^TestName$' ./path/to/package
go test ./...
go test -race ./...
go vet ./...
go mod tidy -diff
test -z "$(gofmt -l .)"
GOOS=linux GOARCH=386 go vet ./...
GOOS=linux GOARCH=386 go test -count=1 ./...
golangci-lint run
govulncheck ./...
```

Select checks in proportion to the change:

| Change | Minimum useful validation |
| --- | --- |
| Documentation only | `git diff --check`; verify referenced paths, symbols, commands, and links |
| Local implementation | focused package test, then `make test` |
| Exported API or builder behavior | focused positive and negative tests, examples/doc comments, then `make test` |
| Backend adapter | provider fixtures and backend package tests, then `make test` |
| CBOR or Plutus encoding | round-trip, malformed, known-vector, and deterministic tests, then `make test` |
| Dependency metadata | `go mod tidy -diff`, relevant tests, and a `go.mod`/`go.sum` review |
| Portability-sensitive integers | native tests plus the `linux/386` vet and test commands |

Formatting commands mutate files. Run them only when appropriate, then inspect
the diff for unrelated churn. Do not claim a check passed unless you ran it and
read its exit status. List every skipped expected check and the concrete reason.

CI currently tests Go 1.25.13 and the floating Go 1.26 release, runs race
detection, checks formatting and module tidiness, vets and tests `linux/386`,
runs `golangci-lint`, scans with `govulncheck`, and performs CodeQL analysis.
Read `.github/workflows/` before changing CI assumptions.

## Code Review Guide

Review the requested diff against its base and stated intent. Follow data and
control flow beyond the changed lines. Treat automated-review comments as
hypotheses: reproduce or disprove each one against the current checkout.

Prioritize findings in this order:

1. security, key-handling, signing, transaction-validity, and behavioral bugs;
2. public API, serialized-data, CBOR, and provider-contract compatibility;
3. missing regression, malformed-input, determinism, race, or portability
   coverage;
4. architecture and dependency-boundary violations;
5. documentation or generated-artifact drift;
6. maintainability and style issues that tools do not already settle.

Every finding should identify the path and current line or symbol, the input or
state that triggers it, the observable impact, and the evidence used to confirm
it. Distinguish blockers from non-blocking recommendations, and distinguish
new defects from pre-existing problems. If there are no findings, say so and
name the checks and areas reviewed; never infer correctness from bot silence.

For pull requests, keep summaries and review comments short and factual. Use
Conventional Commits and DCO sign-off (`git commit -s`). Human review remains
required even when an automated reviewer approves the change.

## Documentation Expectations

Repository-local commands, APIs, and architecture belong here, in
`README.md`, package documentation, or `docs/`. Prefer links to the owning
source or test over copying explanations that can drift.

Update user documentation and examples when behavior, supported APIs,
configuration, or toolchain requirements change. Update `CHANGELOG.md` when the
project's release-facing convention calls for it. Keep examples reproducible
and compile-tested where practical. Do not present historical cardano-cli parity
notes as current without revalidation.

## Handoff

When handing work back, report:

- what changed and why;
- the exact validation commands run and their exit codes;
- expected checks not run, with concrete reasons;
- remaining risks, compatibility concerns, or follow-up work; and
- for a review, findings ordered by severity or an explicit statement that no
  findings were identified.

Do not claim live-provider, Cardano network, GitHub, or release validation that
was not actually performed.
