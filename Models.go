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

/*
*

	ToValue converts a Unit to a Value object.

	Returns:
		Value: The constructed Value object representing the asset.
*/
func (u *Unit) ToValue() Value.Value {
	val := Value.Value{}
	policyId := Policy.PolicyId{Value: u.PolicyId}
	ma := MultiAsset.MultiAsset[int64]{}
	aname := AssetName.NewAssetNameFromString(u.Name)
	ma[policyId] = map[AssetName.AssetName]int64{aname: int64(u.Quantity)}
	val.AddAssets(ma)
	return val
}

/*
*

	NewUnit creates a new Unit with the provided information.

	Params:
		policyId (string): The policy ID of the asset.
		name (string): The name of the asset.
		quantity (int): The quantity of the asset.

	Returns:
		Unit: The newly created Unit instance.
*/
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
	Lovelace     int
	Receiver     Address.Address
	Units        []Unit
	Datum        *PlutusData.PlutusData
	DatumHash    []byte
	IsInline     bool
	ScriptRef    []byte
	HasScriptRef bool
}

/*
*

	PaymentFromTxOut creates a Payment object from a TransactionOutput.

	Params:
		txOut (*TransactionOutput.TransactionOutput): The TransactionOutput to create the Payment.

	Returns:
		*Payment: The created Payment object.
*/
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

/*
*

	NewPayment creates a new Payment object.

	Params:
		receiver (string): The receiver's address.
		lovelace (int): The amount in Lovelace.
		units ([]Unit): The assets units to be included.

	Returns:
		*Payment: The newly created Payment object.
*/
func NewPayment(receiver string, lovelace int, units []Unit) *Payment {
	decoded_address, _ := Address.DecodeAddress(receiver)
	return &Payment{
		Lovelace: lovelace,
		Receiver: decoded_address,
		Units:    units,
	}
}

/*
*

	NewPaymentFromValue creates a new Payment object from an Address
	and Value object.

	Params:
		receiver (Address.Address): The receiver's address.
		value (Value.Value): The value object containing payment details.

	Returns:
		*Payment: The newly created Payment object.
*/
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

/*
*

	ToValue converts a Payment to a Value object.

	Returns:
		Value.Value: The constructed Value object representing the payment.
*/
func (p *Payment) ToValue() Value.Value {
	v := Value.Value{}
	for _, unit := range p.Units {
		v.AddAssets(unit.ToValue().GetAssets())
	}
	v.AddLovelace(int64(p.Lovelace))
	return v
}

/*
*

	EnsureMinUTXO ensures that the payment amount meets the minimun UTXO requirement.

	Params:
		cc (Base.ChainContext): The chain context.
*/
func (p *Payment) EnsureMinUTXO(cc Base.ChainContext) {
	if len(p.Units) == 0 && p.Lovelace >= 1_000_000 {
		return
	}
	txOut := p.ToTxOut()
	coins := Utils.MinLovelacePostAlonzo(*txOut, cc)
	if int64(p.Lovelace) < coins {
		p.Lovelace = int(coins)
	}
}

/*
*

		ToTxOut converts a Payment object to a TransactionOutput object.

		Returns:
	   		*TransactionOutput.TransactionOutput: The created TransactionOutput object.
*/
func (p *Payment) ToTxOut() *TransactionOutput.TransactionOutput {
	txOut := TransactionOutput.SimpleTransactionOutput(p.Receiver, p.ToValue())
	if p.HasScriptRef {
		txO := TransactionOutput.TransactionOutput{}
		txO.IsPostAlonzo = true
		l := PlutusData.DatumOptionInline(p.Datum)
		txO.PostAlonzo.Datum = &l
		txO.PostAlonzo.Address = p.Receiver
		txO.PostAlonzo.Amount = p.ToValue().ToAlonzoValue()

		if p.Datum != nil {
			txOut.SetDatum(p.Datum)
		}
		scriptRef := PlutusData.ScriptRef{p.ScriptRef}
		txO.PostAlonzo.ScriptRef = &scriptRef
		return &txO
	}
	if p.IsInline {
		txO := TransactionOutput.TransactionOutput{}
		txO.IsPostAlonzo = true
		l := PlutusData.DatumOptionInline(p.Datum)
		txO.PostAlonzo.Datum = &l
		txO.PostAlonzo.Address = p.Receiver
		txO.PostAlonzo.Amount = p.ToValue().ToAlonzoValue()

		if p.Datum != nil {
			txOut.SetDatum(p.Datum)
		}
		return &txO
	} else {
		if p.DatumHash != nil {
			txOut.PreAlonzo.DatumHash = serialization.DatumHash{Payload: p.DatumHash}
			txOut.PreAlonzo.HasDatum = true

		}
	}

	return &txOut
}
