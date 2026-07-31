package apollo

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

// stubLedgerState is the slice of common.LedgerState the UTxO rules under test
// actually touch. The embedded interface is nil, so any rule reaching for
// something else fails the test loudly instead of silently passing.
type stubLedgerState struct {
	common.LedgerState
	utxos []common.Utxo
}

func (s *stubLedgerState) UtxoById(
	id common.TransactionInput,
) (common.Utxo, error) {
	for _, utxo := range s.utxos {
		if utxo.Id.Id() == id.Id() && utxo.Id.Index() == id.Index() {
			return utxo, nil
		}
	}
	return common.Utxo{}, errors.New("utxo not found")
}

func scriptDataTestParams() backend.ProtocolParameters {
	return backend.ProtocolParameters{
		MinFeeConstant:      155381,
		MinFeeCoefficient:   44,
		MaxTxSize:           16384,
		CoinsPerUtxoByte:    "4310",
		CollateralPercent:   150,
		MaxCollateralInputs: 3,
		MaxValSize:          "5000",
		PriceMem:            0.0577,
		PriceStep:           0.0000721,
		MaxTxExMem:          "14000000",
		MaxTxExSteps:        "10000000000",
		KeyDeposits:         "2000000",
		PoolDeposits:        "500000000",
		CostModels: map[string][]int64{
			"PlutusV2": {4, 5, 6},
		},
	}
}

// buildWitnessDatumTx builds a signed transaction that carries a witness
// (non-inline) datum alongside a Plutus V2 mint script, and returns the
// transaction as the ledger sees it plus the ledger state resolving its inputs.
func buildWitnessDatumTx(
	t *testing.T,
	datum *common.Datum,
) (*conway.ConwayTransaction, *stubLedgerState, common.Blake2b256) {
	t.Helper()
	gp := backend.GenesisParameters{NetworkMagic: 1}
	cc := fixed.NewFixedChainContext(scriptDataTestParams(), gp, 0)
	addr := testAddress(t)

	var spendHash, collateralHash common.Blake2b256
	spendHash[0] = 0x01
	collateralHash[0] = 0x02
	inputUtxo := makeTestUtxo(t, spendHash, 0, 20_000_000)
	collateralUtxo := makeTestUtxo(t, collateralHash, 0, 5_000_000)

	redeemer := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	script := common.PlutusV2Script([]byte{0x01, 0x02})
	unit := NewUnit(strings.Repeat("ab", 28), "746f6b656e", 1)

	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddInput(inputUtxo).
		AddCollateral(collateralUtxo).
		AttachScript(script).
		DisableExecutionUnitsEstimation().
		Mint(unit, &redeemer, &common.ExUnits{Memory: 1, Steps: 1})

	a, err := a.PayToContractWithDatumHash(addr, datum, 2_000_000)
	if err != nil {
		t.Fatalf("PayToContractWithDatumHash: %v", err)
	}
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatalf("GetTxCbor: %v", err)
	}

	// Decode the serialized transaction so that every ledger check runs
	// against the on-wire bytes, exactly like a node receiving it.
	var tx conway.ConwayTransaction
	if _, err := cbor.Decode(txCbor, &tx); err != nil {
		t.Fatalf("decode built tx: %v", err)
	}
	ls := &stubLedgerState{utxos: []common.Utxo{inputUtxo, collateralUtxo}}

	// The datum hash the transaction declares on its script output.
	var declared common.Blake2b256
	var seen int
	for _, output := range tx.Outputs() {
		if h := output.DatumHash(); h != nil {
			declared = *h
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("expected exactly 1 output datum hash, got %d", seen)
	}
	return &tx, ls, declared
}

func ledgerTestPparams() *conway.ConwayProtocolParameters {
	return &conway.ConwayProtocolParameters{
		CostModels: map[uint][]int64{1: {4, 5, 6}},
	}
}

// TestScriptDataHashAcceptedByLedgerRules is the settling evidence for the
// script data hash: gouroboros' own Conway UTxO rules recompute the script
// integrity hash over the transaction's on-wire witness bytes and must accept
// what Apollo built.
func TestScriptDataHashAcceptedByLedgerRules(t *testing.T) {
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(99))}
	tx, ls, _ := buildWitnessDatumTx(t, &datum)

	if len(tx.WitnessSet.WsPlutusData.Items()) != 1 {
		t.Fatalf(
			"expected 1 witness datum, got %d",
			len(tx.WitnessSet.WsPlutusData.Items()),
		)
	}
	onWire := hex.EncodeToString(tx.WitnessSet.WsPlutusData.Cbor())
	if !strings.HasPrefix(onWire, "d90102") {
		t.Fatalf("witness plutus_data is not a tag-258 set: %s", onWire)
	}
	if onWire != "d90102811863" {
		t.Fatalf("unexpected on-wire plutus_data bytes: %s", onWire)
	}

	pp := ledgerTestPparams()
	if err := conway.UtxoValidateScriptDataHash(tx, 0, ls, pp); err != nil {
		t.Fatalf("gouroboros ledger rejected script data hash: %v", err)
	}
	if err := conway.UtxoValidateSupplementalDatums(tx, 0, ls, pp); err != nil {
		t.Fatalf("gouroboros ledger rejected witness datum: %v", err)
	}
}

// TestScriptDataHashPreimageUsesOnWireWitnessBytes pins the preimage halves to
// the exact bytes of witness set fields 5 and 4, and shows that the bare-array
// datum encoding that predates this fix hashes to something else.
func TestScriptDataHashPreimageUsesOnWireWitnessBytes(t *testing.T) {
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(99))}
	tx, _, _ := buildWitnessDatumTx(t, &datum)

	declared := tx.Body.TxScriptDataHash
	if declared == nil {
		t.Fatal("built transaction has no script data hash")
	}

	redeemersCbor := tx.WitnessSet.WsRedeemers.Cbor()
	datumsCbor := tx.WitnessSet.WsPlutusData.Cbor()
	langViews, err := common.EncodeLangViews(
		map[uint]struct{}{1: {}},
		map[uint][]int64{1: {4, 5, 6}},
	)
	if err != nil {
		t.Fatal(err)
	}
	preimage := make(
		[]byte,
		0,
		len(redeemersCbor)+len(datumsCbor)+len(langViews),
	)
	preimage = append(preimage, redeemersCbor...)
	preimage = append(preimage, datumsCbor...)
	preimage = append(preimage, langViews...)
	want := common.Blake2b256Hash(preimage)
	if *declared != want {
		t.Fatalf(
			"script data hash does not match on-wire preimage:\n"+
				" declared %x\n on-wire %x\n datums %x",
			declared.Bytes(),
			want.Bytes(),
			datumsCbor,
		)
	}

	// The untagged datum array is what the preimage used to be built from.
	bare, err := cbor.Encode(tx.WitnessSet.WsPlutusData.Items())
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(bare) == hex.EncodeToString(datumsCbor) {
		t.Fatal("bare array and tag-258 set encodings are identical")
	}
	stale := make([]byte, 0, len(redeemersCbor)+len(bare)+len(langViews))
	stale = append(stale, redeemersCbor...)
	stale = append(stale, bare...)
	stale = append(stale, langViews...)
	if *declared == common.Blake2b256Hash(stale) {
		t.Fatalf(
			"script data hash still matches the untagged-datums preimage %x",
			bare,
		)
	}
}

// TestComputeScriptDataHashDatumsHalfIsTaggedSet asserts the exact preimage
// bytes ComputeScriptDataHash uses for the datums half.
func TestComputeScriptDataHashDatumsHalfIsTaggedSet(t *testing.T) {
	datums := []common.Datum{
		{Data: plutigoData.NewInteger(big.NewInt(99))},
	}
	redeemers := map[common.RedeemerKey]common.RedeemerValue{}
	got, err := ComputeScriptDataHash(redeemers, datums, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected a script data hash")
	}

	set := WitnessPlutusData(datums)
	datumsCbor, err := cbor.Encode(&set)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(datumsCbor) != "d90102811863" {
		t.Fatalf(
			"unexpected witness plutus_data bytes: %x",
			datumsCbor,
		)
	}
	emptyCostModels, err := cbor.Encode(map[uint][]int64{})
	if err != nil {
		t.Fatal(err)
	}
	preimage := append([]byte{0xa0}, datumsCbor...)
	preimage = append(preimage, emptyCostModels...)
	want := common.Blake2b256Hash(preimage)
	if *got != want {
		t.Fatalf(
			"ComputeScriptDataHash = %x, want %x",
			got.Bytes(),
			want.Bytes(),
		)
	}

	bare, err := cbor.Encode(datums)
	if err != nil {
		t.Fatal(err)
	}
	stalePreimage := append([]byte{0xa0}, bare...)
	stalePreimage = append(stalePreimage, emptyCostModels...)
	if *got == common.Blake2b256Hash(stalePreimage) {
		t.Fatal("preimage still uses the untagged datum array")
	}
}

// TestPayToContractWithDatumHashMatchesWitnessDatum checks the output's datum
// hash against the hash the ledger derives from the datum's on-wire bytes.
func TestPayToContractWithDatumHashMatchesWitnessDatum(t *testing.T) {
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(99))}
	tx, ls, declared := buildWitnessDatumTx(t, &datum)

	witnessDatums := tx.WitnessSet.WsPlutusData.Items()
	if len(witnessDatums) != 1 {
		t.Fatalf("expected 1 witness datum, got %d", len(witnessDatums))
	}
	onWireHash := witnessDatums[0].Hash()
	if declared != onWireHash {
		t.Fatalf(
			"output datum hash %x does not match on-wire witness datum hash %x",
			declared.Bytes(),
			onWireHash.Bytes(),
		)
	}

	var found bool
	for _, output := range tx.Outputs() {
		if h := output.DatumHash(); h != nil && *h == declared {
			found = true
		}
	}
	if !found {
		t.Fatalf("no output carries datum hash %x", declared.Bytes())
	}
	if err := conway.UtxoValidateSupplementalDatums(
		tx,
		0,
		ls,
		ledgerTestPparams(),
	); err != nil {
		t.Fatalf("gouroboros ledger rejected witness datum: %v", err)
	}
}
