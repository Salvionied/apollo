package apollo

import (
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/txBuilding/Utils"
)

type Unit struct {
	PolicyId string
	Name     string
	Quantity int
}

func (u *Unit) ToValue() Value.Value {
	val := Value.Value{}
	policyId := Policy.PolicyId{Value: u.PolicyId}
	ma := MultiAsset.MultiAsset[int64]{}
	aname := AssetName.NewAssetNameFromString(u.Name)
	ma[policyId] = map[AssetName.AssetName]int64{aname: int64(u.Quantity)}
	val.AddAssets(ma)
	return val
}

func NewUnit(policyId string, name string, quantity int) Unit {
	return Unit{
		PolicyId: policyId,
		Name:     name,
		Quantity: quantity,
	}
}

type PaymentI interface {
	EnsureMinUTXO(cc Base.ChainContext)
	ToTxOut() *TransactionOutput.TransactionOutput
	ToValue() Value.Value
}

type Payment struct {
	Lovelace  int
	Receiver  Address.Address
	Units     []Unit
	Datum     *PlutusData.PlutusData
	DatumHash []byte
	IsInline  bool
}

func PaymentFromTxOut(txOut *TransactionOutput.TransactionOutput) *Payment {
	payment := &Payment{
		Receiver: txOut.GetAddress(),
		Lovelace: int(txOut.GetAmount().GetCoin()),
		IsInline: false,
	}
	hasDatumHash := false
	hasInlineDatum := false
	if txOut.GetDatumHash() != nil {
		payment.DatumHash = txOut.GetDatumHash().Payload
		hasDatumHash = true
	}
	if !txOut.GetDatum().Equal(PlutusData.PlutusData{}) {
		payment.Datum = txOut.GetDatum()
		hasInlineDatum = true
	}
	if hasDatumHash && hasInlineDatum {
		payment.IsInline = true
	}

	for policyId, assets := range txOut.GetAmount().GetAssets() {
		for assetName, quantity := range assets {
			payment.Units = append(payment.Units, Unit{
				PolicyId: policyId.Value,
				Name:     assetName.String(),
				Quantity: int(quantity),
			})
		}
	}
	return payment
}

func NewPayment(receiver string, lovelace int, units []Unit) *Payment {
	decoded_address, _ := Address.DecodeAddress(receiver)
	return &Payment{
		Lovelace: lovelace,
		Receiver: decoded_address,
		Units:    units,
	}
}

func NewPaymentFromValue(receiver Address.Address, value Value.Value) *Payment {
	payment := &Payment{
		Receiver: receiver,
		Lovelace: int(value.GetCoin()),
	}
	for policyId, assets := range value.GetAssets() {
		for assetName, quantity := range assets {
			payment.Units = append(payment.Units, Unit{
				PolicyId: policyId.Value,
				Name:     assetName.String(),
				Quantity: int(quantity),
			})
		}
	}
	return payment
}

func (p *Payment) ToValue() Value.Value {
	v := Value.Value{}
	for _, unit := range p.Units {
		v.AddAssets(unit.ToValue().GetAssets())
	}
	v.AddLovelace(int64(p.Lovelace))
	return v
}

func (p *Payment) EnsureMinUTXO(cc Base.ChainContext) {
	txOut := p.ToTxOut()
	coins := Utils.MinLovelacePostAlonzo(*txOut, cc)
	if int64(p.Lovelace) < coins {
		p.Lovelace = int(coins)
	}
}

func (p *Payment) ToTxOut() *TransactionOutput.TransactionOutput {
	txOut := TransactionOutput.SimpleTransactionOutput(p.Receiver, p.ToValue())
	if p.IsInline {
		if p.Datum != nil {
			txOut.SetDatum(p.Datum)
		}
	} else {
		if p.DatumHash != nil {
			txOut.PreAlonzo.DatumHash = serialization.DatumHash{Payload: p.DatumHash}
			txOut.PreAlonzo.HasDatum = true

		}
	}

	return &txOut
}
