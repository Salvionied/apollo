# Changelog

All notable changes to Apollo are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - Unreleased

Apollo v2 replaces the hand-written `serialization/*` tree with the Blink Labs
ledger packages — [gouroboros](https://github.com/blinklabs-io/gouroboros) for
types, CBOR, scripts, addresses and transactions,
[bursa](https://github.com/blinklabs-io/bursa) for HD wallet derivation and
signing, and [plutigo](https://github.com/blinklabs-io/plutigo) for Plutus
data. The module path is now `github.com/Salvionied/apollo/v2` and the import
path must be updated even where the call sites do not change.

The [v1 to v2 migration guide](docs/v2_migration/MIGRATION.md) carries the
method-by-method mapping; this file records what changed and why. Building
against Apollo v2 requires Go 1.25.13 or newer.

Everything in this release is breaking with respect to 1.8.x — the import path
changes, so individual entries below are not marked. Because the path changes,
the compiler walks you through every renamed type, moved package, changed
signature and removed method: fix the build errors and you have covered the
whole `Changed` and `Removed` lists.

What the compiler cannot tell you is where v2 still compiles and behaves
differently. Read these eight before upgrading:

- **Coin selection changed.** MACS is the default selector, so a given wallet
  will generally choose different inputs and pay a different fee than under v1
  — comparable on plain ADA payments, substantially lower when the payment
  carries native assets. `SetCoinSelector(&LargestFirstSelector{})` restores
  the v1 greedy behavior.
- **Fees are higher for transactions carrying reference scripts.** The Conway
  tiered reference-script fee is now part of the minimum fee. A reference
  script whose per-byte price is absent from the protocol parameters is now an
  error instead of an underpriced transaction.
- **`GetBurns()` returns different numbers.** It now reports the absolute
  quantities burned; previously it was byte-identical to `GetMints()` and
  handed back the signed mint value. `GetMints()` still returns that value.
- **Some addresses your code paid in v1 are now rejected.** An address whose
  payload carries trailing extra-data bytes, or whose bech32 is not the
  canonical encoding of what it decodes to, fails instead of being paid.
- **Mixed-network transactions now fail at `Complete()`** instead of building
  and signing. A context network id outside `{0, 1}` is also rejected.
- **A datum whose stored CBOR does not survive its own re-encoding is now
  rejected**, rather than producing a hash no other tool can match. Accept the
  canonical form with `SetCbor(nil)`, or reference a foreign hash without the
  datum through `PayToContractAsHash`.
- **A single wallet UTxO may now serve as both a spending input and
  collateral**, so a one-UTxO wallet builds script transactions that v1
  refused.
- **The UTxO RPC backend now speaks `utxorpc.v1beta`**, falling back to v1alpha
  on `UNIMPLEMENTED`. Constructing the context is unchanged.

### Added

- Multi-Asset Coin Selection (MACS) as a pluggable `CoinSelector`, and it is
  now the default. `SetCoinSelector(&LargestFirstSelector{})` restores the v1
  greedy behavior. Measured on fees rather than input counts, over transactions
  built through `Complete()` with mainnet parameters: on a 150-transaction
  wallet lifetime of plain ADA payments MACS costs within 0.56% of
  largest-first (25.42 against 25.28 ADA) while ending with one dust UTxO
  rather than ninety, and on a multi-asset payment it is roughly three times
  cheaper (200,437 lovelace over 20 inputs against 623,629 over 286). See
  `docs/design/2026-06-11-macs-coin-selection-design.md`.
- Backend capability reporting: `backend.CapabilityReporter`,
  `backend.CapabilitiesOf`, `backend.Supports` and
  `backend.CapabilityEvaluateTxAdditionalUtxos`. Callers must consult the
  capability before relying on evaluator-supplied UTxOs.
- `ChainContext.EvaluateTx` takes resolved additional UTxOs, so inputs a
  backend cannot yet see on chain (transaction chaining) can be evaluated.
  Blockfrost, Ogmios and Maestro forward them; UTxO RPC declines the
  capability.
- The Conway tiered reference-script fee is included in the minimum fee, wired
  from every backend's per-byte price. A reference script with no per-byte
  price in the protocol parameters is an error rather than an underpriced
  transaction.
- A single wallet UTxO may now serve as both a spending input and collateral,
  matching the ledger and other builders, so a one-UTxO wallet can build a
  script transaction.
- `ogmios.NewOgmiosChainContextFromClients` with the Apollo-owned
  `ogmios.OgmiosClient` and `ogmios.KupoClient` interfaces, for custom
  transports and tests.
- `DatumHash` and `DatumWireCbor` for hashing a datum over the bytes that
  reach the wire.
- Wallet passphrase support: `NewBursaWalletWithPassphrase` and
  `SetWalletFromMnemonicWithPassphrase`.
- `Value` with checked arithmetic: `Add`/`Sub` return an error on uint64
  overflow instead of wrapping.
- A package comment on every package plus a root `doc.go` whose quickstart is
  verified to compile out of tree.

### Changed

- Ledger types come from gouroboros. `conway.ConwayTransactionBody`,
  `conway.ConwayTransaction`, `babbage.BabbageTransactionOutput`,
  `common.MultiAsset[T]`, `common.Datum`, `common.NativeScript` and
  `common.RedeemerKey`/`common.RedeemerValue` replace their v1 counterparts.
- Backend packages moved from `txBuilding/Backend/*ChainContext` to
  `backend/blockfrost`, `backend/maestro`, `backend/ogmios`, `backend/utxorpc`,
  `backend/fixed` and `backend/cache`, and the `ChainContext` interface lives in
  `backend`.
- Version-specific builder methods collapsed into unified ones: `AttachScript`,
  `Mint`, `NewScriptRef`, `AddReferenceInput`, `PayToAddressWithReferenceScript`
  and `PayToContractWithReferenceScript` auto-detect the script type.
  `PayToContract` lost its `isInline` flag, and all staking and delegation
  methods now return `(*Apollo, error)`.
- `NewOgmiosChainContext` takes an Apollo-owned `Config` and
  returns an error, rather than taking `*ogmigo.Client` and `*kugo.Client`. No
  Apollo constructor names a third-party type any more, so the client library
  and its major version stay an implementation detail instead of being frozen
  for the life of 2.x:

  ```go
  // before
  ctx := ogmios.NewOgmiosChainContext(
      ogmigo.New(ogmigo.WithEndpoint("ws://localhost:1337")),
      kugo.New(kugo.WithEndpoint("http://localhost:1442")),
      1,
  )

  // after
  ctx, err := ogmios.NewOgmiosChainContext(ogmios.Config{
      OgmiosEndpoint: "ws://localhost:1337",
      KupoEndpoint:   "http://localhost:1442",
      NetworkId:      1,
  })
  if err != nil {
      return err
  }
  ```

- `PaymentI.ToTxOut()` returns `common.TransactionOutput` instead of
  `*babbage.BabbageTransactionOutput`, and both `ToTxOut()` and `ToValue()` now
  return an error instead of swallowing one. `PaymentI` is implemented by third
  parties, so an era-specific signature could only have been changed in a 3.0.
  Apollo builds Conway bodies, which use the Babbage output format, and reports
  a clear error for any other output type.
- `GetBurns()` returns the absolute quantities of the assets being burned. It
  was previously byte-identical to `GetMints()`, so callers asking for burns
  were handed the signed mint value; `GetMints()` still returns that value,
  negative quantities included.
- `Unit.Quantity` and `Payment.Lovelace` are `int64`, previously `int`, for
  identical precision on 32-bit and 64-bit platforms.
- Addresses are validated before use. A payee whose payload carries trailing
  extra-data bytes, or whose bech32 is not the canonical encoding of what it
  decodes to, is now rejected instead of being accepted and paid. All-uppercase
  bech32 remains valid and is still accepted. Byron addresses are accepted only
  without a bech32 address prefix.
- A transaction mixing networks now fails at `Complete()`.
  `validateAddressNetworks` compares the wallet, change, input and payment
  addresses against the context network id and names the offending address and
  both ids; previously such a transaction built and signed. A context network id
  outside `{0, 1}` is also rejected.
- A datum whose stored CBOR does not survive its own re-encoding is rejected,
  naming both the stored-bytes hash and the wire-bytes hash, rather than
  silently producing a hash no other tool can match. Accept the canonical form
  with `SetCbor(nil)`, or reference a foreign hash without the datum through
  `PayToContractAsHash`.
- The UTxO RPC backend now speaks `utxorpc.v1beta`. Because the connect protocol
  derives the request path from the protobuf package, v1alpha and v1beta are
  independent services, so the first call falls back to v1alpha when the server
  reports v1beta `UNIMPLEMENTED` and remembers the working version for the
  lifetime of the chain context. Nothing else triggers a fallback. Constructing
  the context is unchanged.
- plutigo v0.2.0 removes go-ethereum from the dependency graph, which was
  compiled into every binary built against Apollo for one hash function and
  carried an LGPL-3.0 obligation into this MIT-licensed library. Plutus data
  encoding is byte-identical, so datum hashes are unaffected.

### Removed

- `constants.Network` and `constants.MAINNET`, `constants.TESTNET`,
  `constants.PREVIEW`, `constants.PREPROD`. The enum numbered mainnet `0`, the
  inverse of the Cardano network ids every backend constructor takes — mainnet
  is `1` and testnets are `0` — so `uint8(constants.MAINNET)` selected a
  testnet. Pass the network id directly.
- `constants.BlockfrostBaseUrlTestnet`. That network was retired in 2023; use
  `constants.BlockfrostBaseUrlPreview` or `constants.BlockfrostBaseUrlPreprod`.
- `backend.AddressAmount`, an accidental export with no references anywhere in
  the module.
- The `backend` provider-response parsing helpers `BoundedInt`,
  `BoundedIntFromUint64`, `ParseAssetUnit`, `ParseRedeemerTag`, `ParseFraction`,
  `ParseRational` and `ScriptRefFromBytes` moved to `internal/backendutil` and
  are no longer public API. They were only ever called by the in-tree backends,
  and they now validate their input instead of returning a zero value.
  `backend.ComputeMaxTxFee` and `backend.ValidateAdditionalUtxo` stay exported
  for out-of-tree backends.
- The v1 `serialization/*` package tree, replaced entirely by gouroboros types.
- Builder methods with no direct replacement: `CompleteExact` (use `SetFee` then
  `Complete`), `SetWalletAsChangeAddress` (the wallet is the default change
  address), `SetWalletFromKeypair`, `SetWalletFromBech32`,
  `SetEstimateRequired`, `ConsumeAssetsFromUtxo` (use `ConsumeUTxO`),
  `GetPaymentsLength`, `GetSortedInputs`, and `GetRedeemers`/`UpdateRedeemers`,
  which leaked a private type.

### Fixed

- Metadata never reached the chain. `buildMetadata` returned a `*common.MetaMap`
  without storing its encoding, and gouroboros serializes auxiliary data from
  the stored CBOR, so every transaction with metadata was emitted with
  `auxiliary_data = f6` (CBOR null) while the body committed to an
  `auxiliary_data_hash` over the real map. The transaction could not validate
  and the fee was short by the whole of the metadata's `size * minFeeA`. All
  three entry points were affected.
- Ordinary ADA payments failed across a ~1.15 ADA band of wallet balances, with
  no scripts involved. Dust absorption never converged, because the loop
  compared a dust-inclusive fee against a size-based estimate that cannot know
  about the surcharge, and the coin-selection reserve was the protocol maximum
  fee (876277 lovelace on mainnet) rather than the fee for the transaction being
  built. Sending 2 ADA required a 3.15 ADA balance and now requires 2.17 ADA, so
  wallet sweeps are possible and the top ~0.7 ADA of every wallet is no longer
  unspendable. The fee is now monotone and never settles below a value already
  required by a shape under consideration.
- Claiming a staking reward built a transaction with an empty input set.
  Withdrawals, mints and deposit refunds are implicit inputs, and when they
  covered the selection target on their own, selection returned nothing and the
  body went out with `inputs = []`, which the ledger rejects with
  `InputSetEmptyUTxO`. Selection now contributes exactly one input — preferring
  a pure-ADA UTxO, in canonical order — when no value is required and the caller
  pinned none.
- The script data hash was computed over a preimage that omitted the tag-258 set
  prefix the witness set actually writes for `plutus_data`, so every transaction
  carrying witness (non-inline) datums plus a script was rejected with
  `ScriptIntegrityHashMismatch`. The preimage is now derived from the same
  values the witness set marshals. Separately, empty witness datums are omitted
  from the preimage entirely, which the ledger requires and which broke
  mint-only transactions.
- Datum hashes were computed over re-encoded CBOR rather than the bytes on the
  wire. plutigo normalizes on re-encode, so a datum sourced from chain in a
  non-canonical form was paid to a script address under a hash the dApp's
  off-chain code would never recognize, locking the funds.
- The Ogmios backend decoded every lovelace-valued protocol parameter as zero
  against Ogmios v6.1.0 and later, which encode them as `{"ada": {"lovelace":
  N}}` rather than `{"lovelace": N}`. That zeroed `minFeeConstant`,
  `stakeCredentialDeposit`, `stakePoolDeposit` and `minStakePoolCost`, leaving
  every fee 155381 lovelace short (`FeeTooSmallUTxO`) and mis-balancing
  certificate transactions. Both encodings are now accepted and an unrecognized
  shape or missing required key is an error. Shelley genesis decoding is fixed
  the same way for `activeSlotsCoefficient` (a ratio string), `slotLength`,
  `startTime` (which left `SystemStart` at zero) and `maxReferenceScriptsSize`
  under either spelling.
- The UTxO RPC backend could not build any Plutus transaction. Apollo forwarded
  every resolved spending input as `EvaluateTx`'s `additionalUtxos` without
  consulting `CapabilityEvaluateTxAdditionalUtxos`, and that backend rejects a
  non-empty set outright — which for a script transaction is always. Apollo now
  checks the capability and passes nil otherwise, letting the evaluator resolve
  inputs from its own chain view.
- A bech32 checksum failure silently fell through to base58 in every bech32
  entry point. Every character of a mainnet `addr1...` address is also legal
  base58, so a single mistyped character re-decoded into unrelated bytes and
  Apollo built, balanced and signed a payment to a different recipient without
  an error: 6635 of 7091 single-character corruptions were accepted. The
  human-readable part was ignored, and network nibbles 2-15 and address type
  nibbles 9-13 — neither of which exists on Cardano — were accepted too.
  `resolveCredential`, behind the string forms of `RegisterStake`,
  `DelegateStake` and `DelegateVote`, bypassed the first round of this fix and
  accepted 2697 of 2883 corruptions; it now shares `ParseAddress` with
  everything else.
- Fee estimation counted one witness plus the explicitly registered required
  signers, undercounting by ~102 bytes per missing signature whenever inputs
  span more than one payment credential or a withdrawal needs its stake-key
  signature — measured at 170121 against a 174433 minimum. The count is now
  derived as a set from the wallet, registered signers, distinct payment
  credentials across spending and collateral inputs, and the stake credentials
  behind withdrawals and certificates. It never drops below the old estimate.
- `max_val_size` was parsed by every backend and enforced nowhere, so a wallet
  holding many native assets could build a change output the node refuses with
  `OutputTooBigUTxO` while the transaction stayed inside `max_tx_size`.
  `Complete()` now checks each output's serialized value and names the offending
  output.
- Blockfrost: `UtxoByRef` left `TxHash` empty for `/txs/{hash}/utxos` responses,
  so confirmation polling never completed; `/utils/txs/evaluate` is preferred
  over `/evaluate/utxos`, with a backoff retry for indexing lag, because the
  hosted proxy faults on inline-datum and reference-script inputs;
  `cost_models_raw` is preferred over the named cost models so language views
  match the chain after a cost-model bump; UTxO script hydration is
  parallelized; `OutputIndex` widened to `int64` so its range guard is live on
  32-bit platforms; and the backend reports
  `CapabilityEvaluateTxAdditionalUtxos`, which it does honour.
- Maestro: evaluation reports are validated, and object-shaped
  `additional_utxos` are supported through a direct request, since the SDK's own
  `[]string` type cannot represent them.
- UTxO RPC: Cardano `EvalTx` errors are surfaced instead of being dropped,
  redeemer purpose encoding is validated against the transaction's redeemers,
  and pagination is hardened.
- Execution-unit estimation sends real required-signer witnesses and evaluates a
  balanced, converged draft, so evaluators no longer reject the draft body or
  price the wrong shape. A watch-only `ExternalWallet` is skipped in favor of a
  registered evaluation witness provider.
- Reference-script fee parameters are preserved exactly rather than being
  round-tripped through a lossy representation.
- Collateral handling: `total_collateral` is emitted for an explicit
  `SetCollateralAmount` even alongside `AddCollateral`, an amount below the
  ledger minimum or above the collateral inputs is rejected, an explicit amount
  leaving a sub-min-ADA return is rejected rather than silently forfeiting the
  dust, script-address collateral stays forbidden including when pinned by the
  caller, and collateral is sized inside the fee-convergence loop.
- Protocol-parameter parsing, governance redeemer tags, execution budget checks,
  and completed-transaction size checks are hardened; nil payments, zero-value
  builder state, and invalid pool-registration margins are handled rather than
  panicking or being accepted.
- The default Plutus container encoding is corrected and the documentation
  examples compile.
- The `linux/386` build and the whole test suite pass on 32-bit, and CI now runs
  `go vet` and `go test` there rather than `go build`, which skips `_test.go`
  files and had let a non-compiling 32-bit test tree through.

## Earlier releases

Apollo v1 releases up to and including `v1.8.2` predate this file; see the
[release history](https://github.com/Salvionied/apollo/releases) for those.
