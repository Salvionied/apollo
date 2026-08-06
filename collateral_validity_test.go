package apollo

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

// --- Collateral rounding ---

// requiredCollateral restates the ledger's collateral requirement without
// reusing apollo's arithmetic: the collateral must cover
// fee * collateralPercent / 100 rounded UP, so a product that is not an exact
// multiple of 100 needs one lovelace more than the truncating division gives.
// Flooring instead under-collateralizes and the node rejects the transaction.
func requiredCollateral(fee int64, percent int) int64 {
	product := fee * int64(percent)
	required := product / 100
	if product%100 != 0 {
		required++
	}
	return required
}

// collateralPercentContext returns a fixed context identical to the default one
// except for its collateral percentage, so the rounding can be exercised at
// residues other than the 50 that percent=150 always produces.
func collateralPercentContext(
	t *testing.T,
	percent int,
) *fixed.FixedChainContext {
	t.Helper()
	base := setupFixedContext()
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.CollateralPercent = percent
	return fixed.NewFixedChainContext(pp, gp, base.NetworkId())
}

// autoCollateralBuilder returns a script builder whose collateral has been
// auto-selected from a single ADA-only UTxO holding lovelace. Only
// auto-selected collateral with no explicit amount is resized from the final
// fee, which is the path finalizeCollateral computes the requirement on.
func autoCollateralBuilder(
	t *testing.T,
	cc backend.ChainContext,
	lovelace uint64,
) *Apollo {
	t.Helper()
	var hash common.Blake2b256
	hash[0] = 0x41
	a := New(cc).
		SetWallet(NewExternalWallet(testAddress(t))).
		AttachScript(common.PlutusV2Script([]byte{0x01, 0x02})).
		AddLoadedUTxOs(makeTestUtxo(t, hash, 0, lovelace))
	if err := a.setCollateral(); err != nil {
		t.Fatalf("setCollateral failed: %v", err)
	}
	if len(a.collaterals) != 1 {
		t.Fatalf("expected 1 auto-selected collateral, got %d",
			len(a.collaterals))
	}
	return a
}

// TestCollateralRoundsUpWhenFeeTimesPercentIsNotWhole is the regression guard
// for the collateral rounding. The existing collateral tests all land on fees
// whose product with the collateral percentage is an exact multiple of 100
// (percent=150 divides evenly for every even fee), so truncating the division
// produces the same number and no test notices. These cases pin fees where
// ceil and floor differ: flooring leaves the transaction one lovelace short of
// the ledger requirement.
func TestCollateralRoundsUpWhenFeeTimesPercentIsNotWhole(t *testing.T) {
	const collateralLovelace = 10_000_000

	tests := []struct {
		name    string
		percent int
		fee     int64
		// rounds records whether fee * percent is expected NOT to divide by
		// 100, i.e. whether this case distinguishes ceil from floor.
		rounds bool
	}{
		// percent=150: every odd fee leaves a remainder of 50.
		{name: "150pct odd fee", percent: 150, fee: 155_381, rounds: true},
		{name: "150pct odd fee high", percent: 150, fee: 300_001,
			rounds: true},
		{name: "150pct even fee", percent: 150, fee: 300_000},
		// percent=137 varies the residue away from 50.
		{name: "137pct residue 97", percent: 137, fee: 155_381,
			rounds: true},
		{name: "137pct residue 50", percent: 137, fee: 250_350,
			rounds: true},
		{name: "137pct exact", percent: 137, fee: 300_000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := requiredCollateral(tc.fee, tc.percent)
			floor := tc.fee * int64(tc.percent) / 100
			if got := want != floor; got != tc.rounds {
				t.Fatalf(
					"case premise wrong: ceil %d, floor %d, rounds=%v",
					want, floor, tc.rounds,
				)
			}

			cc := collateralPercentContext(t, tc.percent)
			a := autoCollateralBuilder(t, cc, collateralLovelace)
			if err := a.finalizeCollateral(tc.fee); err != nil {
				t.Fatalf("finalizeCollateral(%d) failed: %v", tc.fee, err)
			}

			if a.totalCollateral != want {
				t.Errorf(
					"total_collateral = %d, want %d (ceil(fee %d * %d%% / 100))",
					a.totalCollateral,
					want,
					tc.fee,
					tc.percent,
				)
			}

			// The collateral input's value must be conserved: on phase-2
			// failure the ledger takes total_collateral and returns the rest,
			// so total_collateral + collateral_return == collateral input.
			if a.collateralReturn == nil {
				t.Fatal("expected a collateral return for the ADA remainder")
			}
			ret := a.collateralReturn.Amount()
			if ret == nil {
				t.Fatal("collateral return has no amount")
			}
			sum := new(big.Int).Add(ret, big.NewInt(a.totalCollateral))
			if sum.Cmp(big.NewInt(collateralLovelace)) != 0 {
				t.Errorf(
					"collateral value not conserved: total %d + return %s = %s, input %d",
					a.totalCollateral,
					ret,
					sum,
					collateralLovelace,
				)
			}
			if a.collateralReturn.Assets() != nil {
				t.Error("ADA-only collateral produced a return carrying assets")
			}
		})
	}
}

// TestForcedOddFeeCollateralIsRoundedUpInBody drives the same rounding through
// Complete() so the value that reaches the transaction body is checked, not
// just the helper's field. The fee is forced to an odd number because a
// converged fee is even as often as not, and an even fee cannot tell ceil from
// floor at percent=150.
func TestForcedOddFeeCollateralIsRoundedUpInBody(t *testing.T) {
	const (
		fee       = int64(300_001)
		spendAda  = uint64(30_000_000)
		collAda   = uint64(10_000_000)
		spendHash = byte(0x01)
		collHash  = byte(0x02)
	)

	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, spendAda, spendHash, 0)
	addTestUtxo(cc, addr, collAda, collHash, 0)

	a := mintingScriptBuilder(t, cc, addr).ForceFee(fee)
	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	body := a.GetTx().Body
	// The premise: an odd fee, so fee * 150 is not a multiple of 100.
	if body.TxFee != uint64(fee) {
		t.Fatalf("fee = %d, want the forced %d", body.TxFee, fee)
	}
	pp, err := cc.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	want := requiredCollateral(fee, pp.CollateralPercent)
	if want == fee*int64(pp.CollateralPercent)/100 {
		t.Fatalf("fee %d does not exercise rounding at %d%%",
			fee, pp.CollateralPercent)
	}
	//nolint:gosec // want is positive
	if body.TxTotalCollateral != uint64(want) {
		t.Errorf(
			"total_collateral = %d, want %d (ceil(fee %d * %d%% / 100))",
			body.TxTotalCollateral, want, fee, pp.CollateralPercent,
		)
	}

	// Conservation against the collateral inputs the body actually names,
	// valued from the UTxOs this test created rather than from apollo.
	known := map[string]uint64{
		utxoRefFor(spendHash, 0): spendAda,
		utxoRefFor(collHash, 0):  collAda,
	}
	var collateralValue uint64
	for _, in := range body.TxCollateral.Items() {
		ref := hex.EncodeToString(in.TxId.Bytes()) + "#" +
			strconv.Itoa(int(in.OutputIndex))
		lovelace, ok := known[ref]
		if !ok {
			t.Fatalf("body names an unknown collateral input %s", ref)
		}
		collateralValue += lovelace
	}
	if collateralValue == 0 {
		t.Fatal("body has no collateral inputs")
	}
	if body.TxCollateralReturn == nil {
		t.Fatal("expected a collateral return for the ADA remainder")
	}
	ret := body.TxCollateralReturn.Amount()
	if ret == nil {
		t.Fatal("collateral return has no amount")
	}
	sum := new(big.Int).Add(ret, new(big.Int).SetUint64(body.TxTotalCollateral))
	if sum.Cmp(new(big.Int).SetUint64(collateralValue)) != 0 {
		t.Errorf(
			"collateral value not conserved: total %d + return %s = %s, inputs %d",
			body.TxTotalCollateral,
			ret,
			sum,
			collateralValue,
		)
	}
}

// TestCollateralAmountOneBelowRequirementIsRejected pins the rounding from the
// other side: the ledger minimum is the rounded-UP value, so a requested
// amount equal to the floor is one lovelace short and must be refused. Under a
// flooring requirement this request looks exactly sufficient.
func TestCollateralAmountOneBelowRequirementIsRejected(t *testing.T) {
	const fee = int64(300_001)

	pp, err := setupFixedContext().ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	required := requiredCollateral(fee, pp.CollateralPercent)
	floor := fee * int64(pp.CollateralPercent) / 100
	if required != floor+1 {
		t.Fatalf("fee %d does not exercise rounding at %d%%",
			fee, pp.CollateralPercent)
	}

	build := func(amount int64) (*Apollo, error) {
		cc := setupFixedContext()
		addr := testAddress(t)
		addTestUtxo(cc, addr, 30_000_000, 0x01, 0)
		addTestUtxo(cc, addr, 10_000_000, 0x02, 0)
		return mintingScriptBuilder(t, cc, addr).
			ForceFee(fee).
			SetCollateralAmount(amount).
			Complete()
	}

	if _, err := build(floor); err == nil {
		t.Errorf(
			"collateral amount %d (floor of fee %d * %d%%) was accepted, "+
				"want rejection below the requirement %d",
			floor, fee, pp.CollateralPercent, required,
		)
	} else if !strings.Contains(err.Error(), "insufficient collateral") {
		t.Errorf("unexpected error for an under-funded amount: %v", err)
	}

	built, err := build(required)
	if err != nil {
		t.Fatalf("collateral amount %d (the requirement) was rejected: %v",
			required, err)
	}
	//nolint:gosec // required is positive
	if got := built.GetTx().Body.TxTotalCollateral; got != uint64(required) {
		t.Errorf("total_collateral = %d, want the requested %d",
			got, required)
	}
}

// mintingScriptBuilder returns a builder for a minting Plutus transaction: the
// cheapest shape that forces collateral to be selected and sized.
func mintingScriptBuilder(
	t *testing.T,
	cc backend.ChainContext,
	addr common.Address,
) *Apollo {
	t.Helper()
	unit := NewUnit(
		"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
		"746f6b656e",
		1,
	)
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AttachScript(common.PlutusV2Script([]byte{0x01, 0x02})).
		DisableExecutionUnitsEstimation().
		Mint(unit, &datum, &common.ExUnits{Memory: 1, Steps: 1})
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	return a.AddPayment(payment)
}

// utxoRefFor formats the "txid#index" reference of a UTxO created by
// addTestUtxo, which fills only the first byte of the transaction hash.
func utxoRefFor(txHashByte byte, index uint32) string {
	var hash common.Blake2b256
	hash[0] = txHashByte
	return hex.EncodeToString(hash.Bytes()) + "#" + strconv.Itoa(int(index))
}

// --- Validity interval ---

// bodyValidityKeys decodes the transaction body as the ledger sees it and
// returns the raw values of the validity-interval keys: 3 is ttl (the upper
// bound) and 8 is validity_interval_start (the lower bound) in the Conway
// transaction_body map. Reading the wire keys rather than the Go field names
// makes a swapped assignment visible even if the fields were renamed.
func bodyValidityKeys(t *testing.T, a *Apollo) (ttl, start uint64) {
	t.Helper()
	bodyCbor, err := cbor.Encode(&a.GetTx().Body)
	if err != nil {
		t.Fatal(err)
	}
	fields := make(map[uint64]cbor.RawMessage)
	if _, err := cbor.Decode(bodyCbor, &fields); err != nil {
		t.Fatalf("failed to decode the transaction body map: %v", err)
	}
	read := func(key uint64) uint64 {
		raw, ok := fields[key]
		if !ok {
			return 0
		}
		var value uint64
		if _, err := cbor.Decode(raw, &value); err != nil {
			t.Fatalf("failed to decode body key %d: %v", key, err)
		}
		return value
	}
	return read(3), read(8)
}

// TestValidityIntervalBoundsAreNotSwapped is the regression guard for the
// validity interval. Nothing asserted the built body's ttl and
// validity_interval_start, so exchanging the two assignments kept the suite
// green while inverting the interval, which the node rejects with
// OutsideValidityIntervalUTxO. The values here are distinct and ordered
// (start < ttl), so a swap shows up in both directions.
func TestValidityIntervalBoundsAreNotSwapped(t *testing.T) {
	tests := []struct {
		name  string
		ttl   int64
		start int64
	}{
		{name: "both bounds", ttl: 51_234_567, start: 50_000_123},
		{name: "only ttl", ttl: 51_234_567},
		{name: "only validity start", start: 50_000_123},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cc := setupFixedContext()
			addr := testAddress(t)
			addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

			payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
			if err != nil {
				t.Fatal(err)
			}
			a := New(cc).
				SetWallet(NewExternalWallet(addr)).
				AddPayment(payment)
			if tc.ttl > 0 {
				a = a.SetTtl(tc.ttl)
			}
			if tc.start > 0 {
				a = a.SetValidityStart(tc.start)
			}
			a, err = a.Complete()
			if err != nil {
				t.Fatalf("Complete failed: %v", err)
			}

			body := a.GetTx().Body
			//nolint:gosec // both bounds are non-negative slot numbers
			wantTtl, wantStart := uint64(tc.ttl), uint64(tc.start)
			if body.Ttl != wantTtl {
				t.Errorf("body ttl = %d, want %d", body.Ttl, wantTtl)
			}
			if body.TxValidityIntervalStart != wantStart {
				t.Errorf("body validity_interval_start = %d, want %d",
					body.TxValidityIntervalStart, wantStart)
			}

			// And the same two values on the wire, by ledger key.
			gotTtl, gotStart := bodyValidityKeys(t, a)
			if gotTtl != wantTtl {
				t.Errorf("encoded ttl (key 3) = %d, want %d", gotTtl, wantTtl)
			}
			if gotStart != wantStart {
				t.Errorf(
					"encoded validity_interval_start (key 8) = %d, want %d",
					gotStart,
					wantStart,
				)
			}
		})
	}
}

// TestValidityIntervalSurvivesRoundTrip checks the bounds a submitted
// transaction carries: they must decode back to the values that were set, in
// the same order.
func TestValidityIntervalSurvivesRoundTrip(t *testing.T) {
	const (
		ttl   = int64(51_234_567)
		start = int64(50_000_123)
	)

	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(payment).
		SetValidityStart(start).
		SetTtl(ttl).
		Complete()
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}

	var decoded conway.ConwayTransaction
	if _, err := cbor.Decode(txCbor, &decoded); err != nil {
		t.Fatalf("failed to decode the built transaction: %v", err)
	}
	if got := decoded.Body.Ttl; got != uint64(ttl) {
		t.Errorf("decoded ttl = %d, want %d", got, ttl)
	}
	if got := decoded.Body.TxValidityIntervalStart; got != uint64(start) {
		t.Errorf("decoded validity_interval_start = %d, want %d", got, start)
	}
	// The interval must stay non-empty: the lower bound below the upper one.
	if decoded.Body.TxValidityIntervalStart >= decoded.Body.Ttl {
		t.Errorf(
			"inverted validity interval: start %d >= ttl %d",
			decoded.Body.TxValidityIntervalStart, decoded.Body.Ttl,
		)
	}
}
