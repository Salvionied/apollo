package apollo

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/bursa/bip32"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

type staticSelector struct {
	selected []common.Utxo
	err      error
}

func (s *staticSelector) Name() string { return "static" }

func (s *staticSelector) Select(
	context.Context,
	[]common.Utxo,
	Value,
) ([]common.Utxo, error) {
	return s.selected, s.err
}

type cloneableTestPayment struct {
	value int
}

func (p *cloneableTestPayment) EnsureMinUTXO(backend.ChainContext) error { return nil }
func (p *cloneableTestPayment) ToTxOut() (common.TransactionOutput, error) {
	return nil, errors.New("not used")
}
func (p *cloneableTestPayment) ToValue() (Value, error) { return Value{}, nil }
func (p *cloneableTestPayment) ClonePayment() (PaymentI, error) {
	copy := *p
	return &copy, nil
}

func TestNewRejectsNilChainContext(t *testing.T) {
	if _, err := New(nil).Complete(); err == nil || !strings.Contains(err.Error(), "chain context must not be nil") {
		t.Fatalf("New(nil).Complete() error = %v", err)
	}

	var typedNil *fixed.FixedChainContext
	if _, err := New(typedNil).Complete(); err == nil || !strings.Contains(err.Error(), "chain context must not be nil") {
		t.Fatalf("New(typed nil).Complete() error = %v", err)
	}
}

func TestNegativeBuilderSettersRecordError(t *testing.T) {
	tests := []struct {
		name string
		set  func(*Apollo)
	}{
		{name: "ttl", set: func(a *Apollo) { a.SetTtl(-1) }},
		{name: "validity start", set: func(a *Apollo) { a.SetValidityStart(-1) }},
		{name: "fee", set: func(a *Apollo) { a.SetFee(-1) }},
		{name: "fee padding", set: func(a *Apollo) { a.SetFeePadding(-1) }},
		{name: "forced fee", set: func(a *Apollo) { a.ForceFee(-1) }},
		{name: "collateral amount", set: func(a *Apollo) { a.SetCollateralAmount(-1) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(setupFixedContext())
			tt.set(a)
			if a.err == nil {
				t.Fatal("negative value did not record an error")
			}
			if _, err := a.Complete(); !errors.Is(err, a.err) {
				t.Fatalf("Complete() error = %v, want %v", err, a.err)
			}
		})
	}
}

func TestAddRequiredSignerPaymentKeyRejectsScriptCredential(t *testing.T) {
	raw := make([]byte, 57)
	raw[0] = common.AddressTypeScriptKey << 4
	addr, err := common.NewAddressFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}

	a := New(setupFixedContext()).AddRequiredSignerPaymentKey(addr)
	if a.err == nil {
		t.Fatal("script payment credential was accepted as a required key signer")
	}
	if len(a.requiredSigners) != 0 {
		t.Fatalf("required signers = %d, want 0", len(a.requiredSigners))
	}
}

func TestSelectCoinsValidatesCustomSelectorResult(t *testing.T) {
	first := makeSelectorUtxo(t, 1, 0, 3_000_000, nil)
	second := makeSelectorUtxo(t, 2, 0, 4_000_000, nil)
	unavailable := makeSelectorUtxo(t, 3, 0, 10_000_000, nil)

	tests := []struct {
		name     string
		selected []common.Utxo
		want     string
	}{
		{name: "malformed", selected: []common.Utxo{{}}, want: "invalid UTxO"},
		{name: "unavailable", selected: []common.Utxo{unavailable}, want: "unavailable UTxO"},
		{name: "duplicate", selected: []common.Utxo{first, first}, want: "duplicate UTxO"},
		{name: "under target", selected: []common.Utxo{first}, want: "does not cover"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(setupFixedContext()).
				AddLoadedUTxOs(first, second).
				SetCoinSelector(&staticSelector{selected: tt.selected})
			if _, err := a.selectCoins(NewSimpleValue(5_000_000), Value{}); err == nil ||
				!strings.Contains(err.Error(), tt.want) {
				t.Fatalf("selectCoins() error = %v, want containing %q", err, tt.want)
			}
			if len(a.usedUtxos) != 0 {
				t.Fatalf("selector failure marked used UTxOs: %v", a.usedUtxos)
			}
		})
	}
}

func TestSelectCoinsUsesCanonicalAvailableUtxo(t *testing.T) {
	available := makeSelectorUtxo(t, 1, 0, 3_000_000, nil)
	tampered := makeSelectorUtxo(t, 1, 0, 10_000_000, nil)
	a := New(setupFixedContext()).
		AddLoadedUTxOs(available).
		SetCoinSelector(&staticSelector{selected: []common.Utxo{tampered}})

	if _, err := a.selectCoins(NewSimpleValue(5_000_000), Value{}); err == nil ||
		!strings.Contains(err.Error(), "does not cover") {
		t.Fatalf("selectCoins() error = %v, want under-coverage error", err)
	}
}

func TestCloneCopiesOwnedMutableStateAndSelector(t *testing.T) {
	selector := &staticSelector{}
	scriptRef, err := NewScriptRef(common.PlutusV2Script{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	payment := &Payment{
		Receiver:  testAddress(t),
		Lovelace:  2_000_000,
		Units:     []Unit{{Name: "01", Quantity: 1}},
		Datum:     &datum,
		DatumHash: []byte{4, 5, 6},
		ScriptRef: scriptRef,
	}
	utxo := makeSelectorUtxo(t, 1, 0, 3_000_000, nil)
	metadataBytes := []byte{7, 8, 9}
	customPayment := &cloneableTestPayment{value: 10}
	a := New(setupFixedContext()).
		SetCoinSelector(selector).
		AddPayment(payment).
		AddPayment(customPayment).
		AddLoadedUTxOs(utxo).
		SetShelleyMetadata(map[uint64]any{1: map[string]any{"bytes": metadataBytes}})
	a.v2scripts = append(a.v2scripts, common.PlutusV2Script{10, 11})

	clone := a.Clone()
	if clone.err != nil {
		t.Fatalf("Clone() recorded error: %v", clone.err)
	}
	if clone.coinSelector != CoinSelector(selector) {
		t.Fatal("Clone() did not preserve the configured coin selector")
	}

	payment.Units[0].Quantity = 99
	payment.DatumHash[0] = 99
	payment.ScriptRef.Script.(common.PlutusV2Script)[0] = 99
	metadataBytes[0] = 99
	a.v2scripts[0][0] = 99
	utxo.Output.(*babbage.BabbageTransactionOutput).OutputAmount.Amount = 99
	customPayment.value = 99

	clonedPayment := clone.payments[0].(*Payment)
	if clonedPayment.Units[0].Quantity == 99 ||
		clonedPayment.DatumHash[0] == 99 ||
		clonedPayment.ScriptRef.Script.(common.PlutusV2Script)[0] == 99 {
		t.Fatal("built-in payment state aliases the original")
	}
	clonedMetadata := clone.auxiliaryData.metadata[1].(map[string]any)["bytes"].([]byte)
	if clonedMetadata[0] == 99 {
		t.Fatal("metadata aliases the original")
	}
	if clone.v2scripts[0][0] == 99 {
		t.Fatal("script bytes alias the original")
	}
	if clone.utxos[0].Output.Amount().Uint64() == 99 {
		t.Fatal("UTxO output aliases the original")
	}
	if clone.payments[1].(*cloneableTestPayment).value == 99 {
		t.Fatal("custom cloneable payment aliases the original")
	}
}

func TestNewKeyPairWalletCopiesPrivateKey(t *testing.T) {
	key := make(bip32.XPrv, 96)
	key[0] = 1
	wallet, err := NewKeyPairWallet(testAddress(t), key)
	if err != nil {
		t.Fatal(err)
	}
	key[0] = 99
	if wallet.privateKey[0] != 1 {
		t.Fatal("wallet private key aliases caller-owned memory")
	}
}
