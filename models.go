package apollo

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/constants"
)

// Unit represents a native asset quantity.
type Unit struct {
	PolicyId string
	Name     string
	Quantity int64
}

// NewUnit creates a new Unit.
func NewUnit(policyId, name string, quantity int64) Unit {
	return Unit{
		PolicyId: policyId,
		Name:     name,
		Quantity: quantity,
	}
}

// ToValue converts a Unit to a Value containing this asset.
func (u *Unit) ToValue() (Value, error) {
	if u.PolicyId == "" || u.PolicyId == "lovelace" {
		if u.Quantity < 0 {
			return Value{}, fmt.Errorf("negative lovelace quantity: %d", u.Quantity)
		}
		return NewSimpleValue(uint64(u.Quantity)), nil //nolint:gosec // validated non-negative above
	}
	if u.Quantity < 0 {
		return Value{}, fmt.Errorf("negative native asset quantity: %d for policy %s", u.Quantity, u.PolicyId)
	}
	policyBytes, err := hex.DecodeString(u.PolicyId)
	if err != nil {
		return Value{}, fmt.Errorf("invalid policy ID hex %q: %w", u.PolicyId, err)
	}
	if len(policyBytes) != common.Blake2b224Size {
		return Value{}, fmt.Errorf("invalid policy ID length: expected %d bytes, got %d", common.Blake2b224Size, len(policyBytes))
	}
	var policyId common.Blake2b224
	copy(policyId[:], policyBytes)

	nameBytes, err := hex.DecodeString(u.Name)
	if err != nil {
		return Value{}, fmt.Errorf("invalid asset name hex %q: %w (asset names must be hex-encoded)", u.Name, err)
	}

	data := map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
		policyId: {
			cbor.NewByteString(nameBytes): big.NewInt(u.Quantity),
		},
	}
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](data)
	return NewValue(0, &assets), nil
}

// toMintValue converts a Unit to a Value, allowing negative quantities (for burns).
// This is an internal method used only by mintValue().
func (u *Unit) toMintValue() (Value, error) {
	if u.PolicyId == "" || u.PolicyId == "lovelace" {
		if u.Quantity < 0 {
			return Value{}, fmt.Errorf("negative lovelace quantity: %d", u.Quantity)
		}
		return NewSimpleValue(uint64(u.Quantity)), nil //nolint:gosec // validated non-negative above
	}
	policyBytes, err := hex.DecodeString(u.PolicyId)
	if err != nil {
		return Value{}, fmt.Errorf("invalid policy ID hex %q: %w", u.PolicyId, err)
	}
	if len(policyBytes) != common.Blake2b224Size {
		return Value{}, fmt.Errorf("invalid policy ID length: expected %d bytes, got %d", common.Blake2b224Size, len(policyBytes))
	}
	var policyId common.Blake2b224
	copy(policyId[:], policyBytes)

	nameBytes, err := hex.DecodeString(u.Name)
	if err != nil {
		return Value{}, fmt.Errorf("invalid asset name hex %q: %w (asset names must be hex-encoded)", u.Name, err)
	}

	data := map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
		policyId: {
			cbor.NewByteString(nameBytes): big.NewInt(u.Quantity),
		},
	}
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](data)
	return NewValue(0, &assets), nil
}

// PaymentI is the interface for payment types.
type PaymentI interface {
	EnsureMinUTXO(cc backend.ChainContext) error
	ToTxOut() (*babbage.BabbageTransactionOutput, error)
	ToValue() (Value, error)
}

// Payment represents a transaction output with receiver, lovelace, and optional assets.
type Payment struct {
	Lovelace  int64
	Receiver  common.Address
	Units     []Unit
	Datum     *common.Datum
	DatumHash []byte
	IsInline  bool
	ScriptRef *common.ScriptRef
}

// NewPayment creates a new Payment.
func NewPayment(receiver string, lovelace int64, units []Unit) (*Payment, error) {
	addr, err := common.NewAddress(receiver)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver address: %w", err)
	}
	return &Payment{
		Lovelace: lovelace,
		Receiver: addr,
		Units:    units,
	}, nil
}

// NewPaymentFromValue creates a Payment from an Address and Value.
// It returns an error if a native-asset quantity exceeds the int64 range,
// rather than silently truncating or saturating it to a wrong value.
func NewPaymentFromValue(receiver common.Address, value Value) (*Payment, error) {
	payment := &Payment{
		Receiver: receiver,
		Lovelace: int64(value.Coin), //nolint:gosec // ADA supply fits in int64
	}
	if value.Assets != nil {
		for _, policyId := range value.Assets.Policies() {
			for _, assetName := range value.Assets.Assets(policyId) {
				qty := value.Assets.Asset(policyId, assetName)
				if !qty.IsInt64() {
					return nil, fmt.Errorf("asset quantity %s for policy %s exceeds int64 range", qty.String(), hex.EncodeToString(policyId.Bytes()))
				}
				payment.Units = append(payment.Units, Unit{
					PolicyId: hex.EncodeToString(policyId.Bytes()),
					Name:     hex.EncodeToString(assetName),
					Quantity: qty.Int64(),
				})
			}
		}
	}
	return payment, nil
}

// PaymentFromTxOut creates a Payment from a BabbageTransactionOutput.
// It returns an error if a native-asset quantity exceeds the int64 range,
// rather than silently truncating or saturating it to a wrong value.
func PaymentFromTxOut(txOut *babbage.BabbageTransactionOutput) (*Payment, error) {
	if txOut == nil {
		return nil, nil
	}
	payment := &Payment{
		Receiver: txOut.OutputAddress,
		Lovelace: int64(txOut.OutputAmount.Amount), //nolint:gosec // ADA supply fits in int64
	}
	if txOut.OutputAmount.Assets != nil {
		for _, policyId := range txOut.OutputAmount.Assets.Policies() {
			for _, assetName := range txOut.OutputAmount.Assets.Assets(policyId) {
				qty := txOut.OutputAmount.Assets.Asset(policyId, assetName)
				if !qty.IsInt64() {
					return nil, fmt.Errorf("asset quantity %s for policy %s exceeds int64 range", qty.String(), hex.EncodeToString(policyId.Bytes()))
				}
				payment.Units = append(payment.Units, Unit{
					PolicyId: hex.EncodeToString(policyId.Bytes()),
					Name:     hex.EncodeToString(assetName),
					Quantity: qty.Int64(),
				})
			}
		}
	}
	payment.ScriptRef = txOut.TxOutScriptRef
	if d := txOut.Datum(); d != nil {
		payment.Datum = d
		payment.IsInline = true
	} else if h := txOut.DatumHash(); h != nil {
		payment.DatumHash = h.Bytes()
	}
	return payment, nil
}

// ToValue converts a Payment to a Value.
func (p *Payment) ToValue() (Value, error) {
	if p.Lovelace < 0 {
		return Value{}, fmt.Errorf("negative lovelace amount: %d", p.Lovelace)
	}
	coin := uint64(p.Lovelace) //nolint:gosec // validated non-negative above
	v := NewSimpleValue(coin)
	for _, unit := range p.Units {
		if unit.Quantity < 0 {
			return Value{}, fmt.Errorf("negative asset quantity %d for policy %s", unit.Quantity, unit.PolicyId)
		}
		uv, err := unit.ToValue()
		if err != nil {
			return Value{}, fmt.Errorf("invalid unit %s: %w", unit.PolicyId, err)
		}
		v, err = v.Add(uv)
		if err != nil {
			return Value{}, err
		}
	}
	return v, nil
}

// EnsureMinUTXO ensures the payment meets the minimum UTxO requirement.
// It iterates because raising Lovelace can increase the CBOR-encoded output size,
// which in turn may require a slightly higher min UTxO. Converges in 1-2 iterations.
func (p *Payment) EnsureMinUTXO(cc backend.ChainContext) error {
	if len(p.Units) == 0 && p.Lovelace >= constants.MinLovelace && p.Datum == nil && len(p.DatumHash) == 0 && p.ScriptRef == nil {
		return nil
	}
	pp, err := cc.ProtocolParams()
	if err != nil {
		return fmt.Errorf("failed to get protocol params: %w", err)
	}
	for range 3 {
		txOut, err := p.ToTxOut()
		if err != nil {
			return fmt.Errorf("failed to build tx output: %w", err)
		}
		coins, err := MinLovelacePostAlonzo(txOut, pp.CoinsPerUtxoByteValue())
		if err != nil {
			return fmt.Errorf("failed to compute min UTxO: %w", err)
		}
		if p.Lovelace >= coins {
			return nil
		}
		p.Lovelace = coins
	}
	// If we exhausted iterations without converging, verify one final time.
	txOut, err := p.ToTxOut()
	if err != nil {
		return fmt.Errorf("failed to build tx output: %w", err)
	}
	coins, err := MinLovelacePostAlonzo(txOut, pp.CoinsPerUtxoByteValue())
	if err != nil {
		return fmt.Errorf("failed to compute min UTxO: %w", err)
	}
	if p.Lovelace < coins {
		return fmt.Errorf("min UTxO did not converge after 3 iterations: need %d, have %d", coins, p.Lovelace)
	}
	return nil
}

// ToTxOut converts a Payment to a BabbageTransactionOutput.
func (p *Payment) ToTxOut() (*babbage.BabbageTransactionOutput, error) {
	val, err := p.ToValue()
	if err != nil {
		return nil, fmt.Errorf("failed to compute payment value: %w", err)
	}
	output := NewBabbageOutput(p.Receiver, val, nil, p.ScriptRef)

	if p.IsInline && p.Datum != nil {
		datumOpt, err := NewDatumOptionInline(p.Datum)
		if err != nil {
			return nil, fmt.Errorf("failed to create inline datum: %w", err)
		}
		output.DatumOption = datumOpt
	} else if len(p.DatumHash) > 0 {
		if len(p.DatumHash) != common.Blake2b256Size {
			return nil, fmt.Errorf("invalid datum hash length: expected %d bytes, got %d", common.Blake2b256Size, len(p.DatumHash))
		}
		var hash common.Blake2b256
		copy(hash[:], p.DatumHash)
		datumOpt, err := NewDatumOptionHash(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to create datum hash: %w", err)
		}
		output.DatumOption = datumOpt
	}
	return &output, nil
}
