package apollo

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func testRedeemerDatum() common.Datum {
	return common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
}

type evalCapture struct {
	TxCbor []byte
	Tx     conway.ConwayTransaction
	Utxos  []common.Utxo
}

// capturingEvalContext is a FixedChainContext wrapper that records every
// EvaluateTx call, optionally asserts value conservation, and returns
// configured ExUnits. It never performs network I/O.
type capturingEvalContext struct {
	*fixed.FixedChainContext
	t *testing.T

	calls []evalCapture

	// assertTx is invoked after decoding each EvaluateTx payload.
	assertTx func(call int, tx *conway.ConwayTransaction, utxos []common.Utxo)

	// resultFor returns ExUnits (or an error) for the call.
	resultFor func(call int, tx *conway.ConwayTransaction, utxos []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error)
}

func (c *capturingEvalContext) EvaluateTx(txCbor []byte, additionalUtxos []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	c.t.Helper()
	var tx conway.ConwayTransaction
	if _, err := cbor.Decode(txCbor, &tx); err != nil {
		return nil, fmt.Errorf("capturingEvalContext: decode EvaluateTx CBOR: %w", err)
	}
	call := len(c.calls)
	cap := evalCapture{
		TxCbor: append([]byte(nil), txCbor...),
		Tx:     tx,
		Utxos:  append([]common.Utxo(nil), additionalUtxos...),
	}
	c.calls = append(c.calls, cap)
	if c.assertTx != nil {
		c.assertTx(call, &tx, additionalUtxos)
	}
	if c.resultFor == nil {
		return nil, errors.New("capturingEvalContext: resultFor not configured")
	}
	return c.resultFor(call, &tx, additionalUtxos)
}

func mintRedeemerUnits(memory, steps int64) map[common.RedeemerKey]common.ExUnits {
	return map[common.RedeemerKey]common.ExUnits{
		{Tag: common.RedeemerTagMint, Index: 0}: {Memory: memory, Steps: steps},
	}
}

func setupMintEvalBuilder(t *testing.T, cc *capturingEvalContext, paymentLovelace int64, mintQty int64) *Apollo {
	t.Helper()
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 50_000_000, 0x01, 0)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x02, 0)

	policyHex := strings.Repeat("ab", 28)
	redeemer := testRedeemerDatum()
	units := []Unit(nil)
	if mintQty > 0 {
		units = []Unit{NewUnit(policyHex, "746f6b656e", mintQty)}
	}
	p, err := NewPayment(validTestAddrBech32, paymentLovelace, units)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Mint(NewUnit(policyHex, "746f6b656e", mintQty), &redeemer, nil)
	return a
}

func sumInputCoin(utxos []common.Utxo) uint64 {
	var total uint64
	for _, u := range utxos {
		total += u.Output.Amount().Uint64()
	}
	return total
}

func sumOutputCoin(tx *conway.ConwayTransaction) uint64 {
	var total uint64
	for _, out := range tx.Body.TxOutputs {
		total += out.OutputAmount.Amount
	}
	return total
}

func sumWithdrawalCoin(tx *conway.ConwayTransaction) uint64 {
	var total uint64
	for _, amt := range tx.Body.TxWithdrawals {
		total += amt
	}
	return total
}

func assertAdaConservedDraft(t *testing.T, tx *conway.ConwayTransaction, utxos []common.Utxo) {
	t.Helper()
	inCoin := sumInputCoin(utxos) + sumWithdrawalCoin(tx)
	outCoin := sumOutputCoin(tx) + tx.Body.TxFee + tx.Body.TxDonation
	for _, proposal := range tx.Body.TxProposalProcedures {
		outCoin += proposal.Deposit()
	}
	// Stake registration deposits / deregistration refunds.
	for _, cert := range tx.Body.TxCertificates {
		switch cert.Type {
		case uint(common.CertificateTypeStakeRegistration),
			uint(common.CertificateTypeRegistration),
			uint(common.CertificateTypeStakeRegistrationDelegation),
			uint(common.CertificateTypeVoteRegistrationDelegation),
			uint(common.CertificateTypeStakeVoteRegistrationDelegation):
			outCoin += StakeDeposit
		case uint(common.CertificateTypeStakeDeregistration),
			uint(common.CertificateTypeDeregistration):
			inCoin += StakeDeposit
		}
	}
	if inCoin != outCoin {
		t.Fatalf("EvaluateTx draft does not conserve ADA: inputs(+wd/refund)=%d outputs(+fee/gov/deposit)=%d fee=%d outs=%d",
			inCoin, outCoin, tx.Body.TxFee, len(tx.Body.TxOutputs))
	}
}

func assetQty(assets *common.MultiAsset[common.MultiAssetTypeOutput], policy common.Blake2b224, name []byte) *big.Int {
	if assets == nil {
		return big.NewInt(0)
	}
	q := assets.Asset(policy, name)
	if q == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(q)
}

func mintQty(mint *common.MultiAsset[common.MultiAssetTypeMint], policy common.Blake2b224, name []byte) *big.Int {
	if mint == nil {
		return big.NewInt(0)
	}
	q := mint.Asset(policy, name)
	if q == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(q)
}

func assertAssetConservedDraft(t *testing.T, tx *conway.ConwayTransaction, utxos []common.Utxo, policy common.Blake2b224, name []byte) {
	t.Helper()
	in := big.NewInt(0)
	for _, u := range utxos {
		in.Add(in, assetQty(u.Output.Assets(), policy, name))
	}
	in.Add(in, mintQty(tx.Body.TxMint, policy, name))

	out := big.NewInt(0)
	for _, o := range tx.Body.TxOutputs {
		out.Add(out, assetQty(o.OutputAmount.Assets, policy, name))
	}
	if in.Cmp(out) != 0 {
		t.Fatalf("EvaluateTx draft does not conserve asset %x: in(+mint)=%s out=%s", name, in, out)
	}
}

func TestEstimateExecutionUnitsDraftConservesAda(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		assertTx: func(_ int, tx *conway.ConwayTransaction, utxos []common.Utxo) {
			assertAdaConservedDraft(t, tx, utxos)
			if len(tx.WitnessSet.VkeyWitnesses.Items()) != 0 {
				t.Fatal("estimateExecutionUnits must not inject fake vkey witnesses")
			}
		},
		resultFor: func(_ int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			return mintRedeemerUnits(1_000, 1_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 5)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.calls) == 0 {
		t.Fatal("expected EvaluateTx to be called")
	}
}

func TestEstimateExecutionUnitsDraftConservesMintedAssets(t *testing.T) {
	var policy common.Blake2b224
	for i := range policy {
		policy[i] = 0xab
	}
	name := []byte("token")
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		assertTx: func(_ int, tx *conway.ConwayTransaction, utxos []common.Utxo) {
			assertAdaConservedDraft(t, tx, utxos)
			assertAssetConservedDraft(t, tx, utxos, policy, name)
			if tx.Body.TxMint == nil {
				t.Fatal("expected mint in evaluation draft")
			}
		},
		resultFor: func(_ int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			return mintRedeemerUnits(2_000, 2_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 5)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

func TestEstimateExecutionUnitsDraftConservesBurnedAssets(t *testing.T) {
	var policy common.Blake2b224
	for i := range policy {
		policy[i] = 0xab
	}
	name := []byte("token")
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
		map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
			policy: {cbor.NewByteString(name): big.NewInt(5)},
		},
	)

	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		assertTx: func(_ int, tx *conway.ConwayTransaction, utxos []common.Utxo) {
			assertAdaConservedDraft(t, tx, utxos)
			assertAssetConservedDraft(t, tx, utxos, policy, name)
		},
		resultFor: func(_ int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			return mintRedeemerUnits(2_000, 2_000), nil
		},
	}
	addr := testAddress(t)
	var txHash common.Blake2b256
	txHash[0] = 0x11
	cc.AddUtxo(addr, makeAssetTestUtxo(t, txHash, 0, 30_000_000, &assets))
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x12, 0)

	policyHex := strings.Repeat("ab", 28)
	redeemer := testRedeemerDatum()
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Mint(NewUnit(policyHex, "746f6b656e", -5), &redeemer, nil)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.calls) == 0 {
		t.Fatal("expected EvaluateTx to be called")
	}
}

func TestEvaluationReRunsWhenChangeOutputAppears(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		resultFor: func(_ int, tx *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			// Larger budget once a change output is present (balanced draft).
			if len(tx.Body.TxOutputs) >= 2 {
				return mintRedeemerUnits(800_000, 800_000), nil
			}
			return mintRedeemerUnits(1_000, 1_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 1)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.calls) < 2 {
		t.Fatalf("expected re-evaluation after change/fee rebuild, got %d EvaluateTx calls", len(cc.calls))
	}
	sawChange := false
	for _, call := range cc.calls {
		if len(call.Tx.Body.TxOutputs) >= 2 {
			sawChange = true
			break
		}
	}
	if !sawChange {
		t.Fatal("expected at least one evaluation draft to include a change output")
	}
}

func TestEvaluationReRunsWhenFeeChanges(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		resultFor: func(_ int, tx *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			// Fee-dependent budgets: low preliminary fee gets a larger budget;
			// once the draft fee has grown, return a slightly smaller but still
			// high budget so fee stays in the high bucket and the loop settles.
			if tx.Body.TxFee < 250_000 {
				return mintRedeemerUnits(3_000_000, 3_000_000), nil
			}
			return mintRedeemerUnits(2_800_000, 2_800_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 1)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.calls) < 2 {
		t.Fatalf("expected fee-driven re-evaluation, got %d calls", len(cc.calls))
	}
	fees := map[uint64]bool{}
	for _, call := range cc.calls {
		fees[call.Tx.Body.TxFee] = true
	}
	if len(fees) < 2 {
		t.Fatalf("expected EvaluateTx drafts with distinct fees, got %#v", fees)
	}
}

func TestEvaluationReRunsWhenCollateralChanges(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		resultFor: func(_ int, tx *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			// Collateral-dependent budgets: low preliminary collateral gets a
			// large budget; once finalizeCollateral grows with fee, return a
			// distinct but still-high budget that keeps collateral elevated.
			if tx.Body.TxTotalCollateral < 400_000 {
				return mintRedeemerUnits(3_500_000, 3_500_000), nil
			}
			return mintRedeemerUnits(3_200_000, 3_200_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 1)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.calls) < 2 {
		t.Fatalf("expected collateral-driven re-evaluation, got %d calls", len(cc.calls))
	}
	collaterals := map[uint64]bool{}
	for _, call := range cc.calls {
		collaterals[call.Tx.Body.TxTotalCollateral] = true
	}
	if len(collaterals) < 2 {
		t.Fatalf("expected EvaluateTx drafts with distinct total_collateral, got %#v", collaterals)
	}
}

func TestEvaluationConvergesOnFinalScriptVisibleShape(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		assertTx: func(_ int, tx *conway.ConwayTransaction, utxos []common.Utxo) {
			assertAdaConservedDraft(t, tx, utxos)
		},
		resultFor: func(call int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			// Stable budgets after the first call so the loop can settle.
			if call == 0 {
				return mintRedeemerUnits(500_000, 500_000), nil
			}
			return mintRedeemerUnits(500_000, 500_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 1)
	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.calls) < 2 {
		t.Fatalf("expected at least two evaluations for shape confirmation, got %d", len(cc.calls))
	}
	final := a.GetTx()
	last := cc.calls[len(cc.calls)-1].Tx
	if final.Body.TxFee != last.Body.TxFee {
		t.Fatalf("final fee %d != last evaluated fee %d", final.Body.TxFee, last.Body.TxFee)
	}
	if len(final.Body.TxOutputs) != len(last.Body.TxOutputs) {
		t.Fatalf("final outputs %d != last evaluated outputs %d", len(final.Body.TxOutputs), len(last.Body.TxOutputs))
	}
	if final.Body.TxTotalCollateral != last.Body.TxTotalCollateral {
		t.Fatalf("final collateral %d != last evaluated collateral %d", final.Body.TxTotalCollateral, last.Body.TxTotalCollateral)
	}
	for i := range final.Body.TxOutputs {
		if final.Body.TxOutputs[i].OutputAmount.Amount != last.Body.TxOutputs[i].OutputAmount.Amount {
			t.Fatalf("output %d amount mismatch: final=%d eval=%d", i,
				final.Body.TxOutputs[i].OutputAmount.Amount, last.Body.TxOutputs[i].OutputAmount.Amount)
		}
	}
}

func TestEvaluationFailsClosedOnShapeCycle(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		resultFor: func(call int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			// Alternate budgets so fee/change/collateral oscillate and shapes cycle.
			if call%2 == 0 {
				return mintRedeemerUnits(4_000_000, 4_000_000), nil
			}
			return mintRedeemerUnits(1_000, 1_000), nil
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 1)
	_, err := a.Complete()
	if err == nil {
		t.Fatal("expected cycle/non-convergence error")
	}
	if !strings.Contains(err.Error(), "evaluation transaction did not converge after 5 iterations") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluationDoesNotMutatePaymentsOnFailure(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		resultFor: func(_ int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			return nil, errors.New("synthetic evaluator failure")
		},
	}
	a := setupMintEvalBuilder(t, cc, 2_000_000, 1)
	before := make([]Value, len(a.payments))
	for i, p := range a.payments {
		v, err := p.ToValue()
		if err != nil {
			t.Fatal(err)
		}
		before[i] = v
	}
	_, err := a.Complete()
	if err == nil {
		t.Fatal("expected Complete to fail")
	}
	if len(a.payments) != len(before) {
		t.Fatalf("payments mutated on failure: len %d -> %d", len(before), len(a.payments))
	}
	for i, p := range a.payments {
		v, err := p.ToValue()
		if err != nil {
			t.Fatal(err)
		}
		if v.Coin != before[i].Coin {
			t.Fatalf("payment %d lovelace mutated: %d -> %d", i, before[i].Coin, v.Coin)
		}
	}
	if a.tx != nil {
		t.Fatal("failed Complete must not leave a built transaction")
	}
}

func TestEvaluationKeepsDeterministicOutputOrder(t *testing.T) {
	cc := &capturingEvalContext{
		FixedChainContext: setupFixedContext(),
		t:                 t,
		resultFor: func(_ int, _ *conway.ConwayTransaction, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
			return mintRedeemerUnits(10_000, 10_000), nil
		},
	}
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 80_000_000, 0x21, 0)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x22, 0)

	policyHex := strings.Repeat("cd", 28)
	redeemer := testRedeemerDatum()
	p1, err := NewPayment(validTestAddrBech32, 3_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := NewPayment(validTestAddrBech32, 7_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p1).
		AddPayment(p2).
		SetTtl(50_000_000).
		Mint(NewUnit(policyHex, "746f6b656e", 1), &redeemer, nil)
	a, err = a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	outs := a.GetTx().Body.TxOutputs
	if len(outs) < 2 {
		t.Fatalf("expected at least payment outputs, got %d", len(outs))
	}
	if outs[0].OutputAmount.Amount != 3_000_000 || outs[1].OutputAmount.Amount != 7_000_000 {
		t.Fatalf("payment output order not preserved: %d, %d", outs[0].OutputAmount.Amount, outs[1].OutputAmount.Amount)
	}
	for _, call := range cc.calls {
		draft := call.Tx.Body.TxOutputs
		if len(draft) < 2 {
			t.Fatalf("eval draft missing payments: %d outputs", len(draft))
		}
		if draft[0].OutputAmount.Amount != 3_000_000 || draft[1].OutputAmount.Amount != 7_000_000 {
			t.Fatalf("eval draft output order not preserved: %d, %d",
				draft[0].OutputAmount.Amount, draft[1].OutputAmount.Amount)
		}
	}
}
