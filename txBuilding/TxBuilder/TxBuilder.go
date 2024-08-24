package TxBuilder

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/Key"
	"github.com/Salvionied/apollo/serialization/Metadata"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/NativeScript"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/apollo/serialization/Withdrawal"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/txBuilding/CoinSelection"
	"github.com/Salvionied/apollo/txBuilding/Errors"
	"github.com/Salvionied/apollo/txBuilding/Utils"

	"github.com/Salvionied/cbor/v2"
	"golang.org/x/exp/slices"
)

//SOON TO BE DEPRECATED

var FAKE_ADDRESS, _ = Address.DecodeAddress("addr1v8xrqjtlfluk9axpmjj5enh0uw0cduwhz7txsqyl36m3ukgqdsn8w")
var fake_vkey_decoded, err = hex.DecodeString("5797dc2cc919dfec0bb849551ebdf30d96e5cbe0f33f734a87fe826db30f7ef9")

var fake_vkey = Key.VerificationKey{Payload: fake_vkey_decoded}

var fake_tx_signature, _ = hex.DecodeString("577ccb5b487b64e396b0976c6f71558e52e44ad254db7d06dfb79843e5441a5d763dd42a")

/**
* TransactionBuilder
* This is the main object used to build a transaction. Soon To Be Deprecated
**/
type TransactionBuilder struct {
	Context                      Base.ChainContext
	UtxoSelectors                []CoinSelection.UTxOSelector
	ExecutionMemoryBuffer        float32
	ExecutionStepBuffer          float32
	Ttl                          int64
	ValidityStart                int64
	LoadedUtxos                  []UTxO.UTxO
	AuxiliaryData                Metadata.AuxiliaryData
	NativeScripts                []PlutusData.ScriptHashable
	Mint                         MultiAsset.MultiAsset[int64]
	RequiredSigners              []serialization.PubKeyHash
	Collaterals                  []UTxO.UTxO
	Certificates                 []Certificate.Certificate
	Withdrawals                  []Withdrawal.Withdrawal
	ReferenceInputs              []TransactionInput.TransactionInput
	Inputs                       []UTxO.UTxO
	ExcludedInputs               []UTxO.UTxO
	InputAddresses               []Address.Address
	Outputs                      []TransactionOutput.TransactionOutput
	Fee                          int64
	Datums                       map[string]PlutusData.PlutusData
	CollateralReturn             *TransactionOutput.TransactionOutput
	TotalCollateral              int64
	InputsToRedeemers            map[string]Redeemer.Redeemer
	MintingScriptToRedeemers     []MintingScriptToRedeemer
	InputsToScripts              map[string]PlutusData.ScriptHashable
	ReferenceScripts             []PlutusData.ScriptHashable
	ShouldEstimateExecutionUnits bool
}

func InitBuilder(context Base.ChainContext) TransactionBuilder {
	txbuilder := TransactionBuilder{}
	txbuilder.Context = context
	txbuilder.UtxoSelectors = []CoinSelection.UTxOSelector{
		CoinSelection.LargestFirstSelector{},
		CoinSelection.RandomImproveMultiAsset{}}
	txbuilder.ExecutionMemoryBuffer = 0.2
	txbuilder.ExecutionStepBuffer = 0.2
	txbuilder.ShouldEstimateExecutionUnits = true
	txbuilder.AuxiliaryData = Metadata.AuxiliaryData{}
	return txbuilder
}

func (tb *TransactionBuilder) AddLoadedUTxOs(loadedTxs []UTxO.UTxO) {
	tb.LoadedUtxos = loadedTxs[:]
}

func (tb *TransactionBuilder) Redeemers() []Redeemer.Redeemer {
	res := []Redeemer.Redeemer{}
	for _, redeemer := range tb.InputsToRedeemers {
		res = append(res, redeemer)
	}
	for _, redeemer := range tb.MintingScriptToRedeemers {
		res = append(res, redeemer.Redeemer)
	}
	return res
}

func (tb *TransactionBuilder) RedeemersReferences() []*Redeemer.Redeemer {
	res := []*Redeemer.Redeemer{}
	for _, redeemer := range tb.InputsToRedeemers {
		res = append(res, &redeemer)
	}
	for _, redeemer := range tb.MintingScriptToRedeemers {
		res = append(res, &redeemer.Redeemer)
	}
	return res
}

func (tb *TransactionBuilder) AddInput(utxo UTxO.UTxO) {
	tb.Inputs = append(tb.Inputs, utxo)
}

func (tb *TransactionBuilder) AddInputAddress(address Address.Address) {
	tb.InputAddresses = append(tb.InputAddresses, address)
}

func (tb *TransactionBuilder) AddScriptInput(utxo UTxO.UTxO, script interface{}, datum *PlutusData.PlutusData, redeemer *Redeemer.Redeemer, isV1 bool) error {
	if utxo.Output.GetAddress().AddressType != 0b0001 &&
		utxo.Output.GetAddress().AddressType != 0b0010 &&
		utxo.Output.GetAddress().AddressType != 0b0011 &&
		utxo.Output.GetAddress().AddressType != 0b0101 {
		return errors.New("expect the output address of utxo to of script type")
	}

	if datumHash, _ := PlutusData.HashDatum(datum); datum != nil &&
		utxo.Output.GetDatumHash() != nil &&
		!utxo.Output.GetDatumHash().Equal(datumHash) {
		return fmt.Errorf("datum hash in transaction output is %s, but actual datum hash from input datum is %s", hex.EncodeToString(utxo.Output.GetDatumHash().Payload[:]), hex.EncodeToString(datumHash.Payload))
	}

	if datum != nil {
		datumHash, _ := PlutusData.HashDatum(datum)
		x := hex.EncodeToString(datumHash.Payload)
		if tb.Datums == nil {
			tb.Datums = make(map[string]PlutusData.PlutusData)
		}
		tb.Datums[x] = *datum

	}
	if redeemer != nil {
		if tb.InputsToRedeemers == nil {
			tb.InputsToRedeemers = make(map[string]Redeemer.Redeemer)
		}
		tb.InputsToRedeemers[Utils.ToCbor(utxo)] = *redeemer
	}
	if script != nil {
		tb.InputsToScripts = make(map[string]PlutusData.ScriptHashable)
	}
	if utxo.Output.IsPostAlonzo && len(utxo.Output.PostAlonzo.ScriptRef.Script.Script) > 0 {
		tb.InputsToScripts[Utils.ToCbor(utxo)] = PlutusData.PlutusV2Script(utxo.Output.GetScriptRef().Script.Script)
		tb.ReferenceInputs = append(tb.ReferenceInputs, utxo.Input)
		tb.ReferenceScripts = append(tb.ReferenceScripts, PlutusData.PlutusV2Script(utxo.Output.GetScriptRef().Script.Script))
	} else if script == nil {
		for _, i := range tb.LoadedUtxos {
			if i.Output.IsPostAlonzo && len(i.Output.PostAlonzo.ScriptRef.Script.Script) > 0 {
				tb.InputsToScripts[Utils.ToCbor(i)] = PlutusData.PlutusV2Script(i.Output.GetScriptRef().Script.Script)
				tb.ReferenceInputs = append(tb.ReferenceInputs, i.Input)
				tb.ReferenceScripts = append(tb.ReferenceScripts, PlutusData.PlutusV2Script(i.Output.GetScriptRef().Script.Script))
				break
			}
		}
	} else {
		if isV1 {
			val, ok := script.(PlutusData.PlutusV1Script)
			if !ok {
				return errors.New("script type error")
			}
			tb.InputsToScripts[Utils.ToCbor(utxo)] = val
		} else {
			val, ok := script.(PlutusData.PlutusV2Script)
			if !ok {
				return errors.New("script type error")
			}
			tb.InputsToScripts[Utils.ToCbor(utxo)] = val
		}
	}

	tb.Inputs = append(tb.Inputs, utxo)
	return nil
}

func (tb *TransactionBuilder) AddMintingScript(script interface{}, redeemer Redeemer.Redeemer) {
	//TODO : implement
}

func (tb *TransactionBuilder) AddOutput(txOut TransactionOutput.TransactionOutput, datum *PlutusData.PlutusData, add_datum_to_witness bool) {
	if datum != nil {
		txOut.SetDatum(datum)
	}
	tb.Outputs = append(tb.Outputs, txOut)
	//TODO: implement
	// if datum != nil && add_datum_to_witness {
	// 	tb._datums [datum.Hash()] = datum
	// }
}

func (tb *TransactionBuilder) _GetTotalKeyDeposit() int64 {
	//TODO: Implement
	return 0
}

func (tb *TransactionBuilder) _AddingAssetMakeOutputOverflow(
	output TransactionOutput.TransactionOutput,
	tempAssets Asset.Asset[int64],
	policyId Policy.PolicyId,
	assetName AssetName.AssetName,
	amount int64,
	maxValSize string) bool {
	attemptAssets := tempAssets.Clone()
	attemptAssets.Add(Asset.Asset[int64]{assetName: amount})
	attemptMultiAsset := MultiAsset.MultiAsset[int64]{policyId: attemptAssets}

	newAmount := Value.Value{Am: Amount.Amount{Coin: 0, Value: attemptMultiAsset}, Coin: 0, HasAssets: true}
	currAmount := output.GetValue().Clone()

	attemptAmount := newAmount.Add(currAmount)

	requiredLovelace := Utils.MinLovelacePostAlonzo(TransactionOutput.SimpleTransactionOutput(output.GetAddress(), attemptAmount), tb.Context)

	attemptAmount.SetLovelace(requiredLovelace)
	bytes, _ := cbor.Marshal(attemptAmount)
	maxValSz, _ := strconv.Atoi(maxValSize)
	return len(bytes) > maxValSz
}

func (tb *TransactionBuilder) _pack_multiassets_for_change(ChangeAddress Address.Address, ChangeEstimator Value.Value, maxValSize string) []MultiAsset.MultiAsset[int64] {
	multiAssetArray := make([]MultiAsset.MultiAsset[int64], 0)
	base_coin := Value.PureLovelaceValue(ChangeEstimator.GetCoin())
	output := TransactionOutput.SimpleTransactionOutput(ChangeAddress, base_coin)
	for policyId, assets := range ChangeEstimator.GetAssets() {
		tempMultiAsset := MultiAsset.MultiAsset[int64]{}
		tempValue := Value.Value{}
		tempAssets := Asset.Asset[int64]{}
		oldAmount := output.GetValue().Clone()
		for asset_name, amount := range assets {
			if tb._AddingAssetMakeOutputOverflow(
				output,
				tempAssets,
				policyId,
				asset_name,
				amount,
				maxValSize) {
				tempMultiAsset = tempMultiAsset.Add(MultiAsset.MultiAsset[int64]{policyId: tempAssets})
				tempValue.SetMultiAsset(tempMultiAsset)

				multiAssetArray = append(multiAssetArray, output.GetValue().GetAssets())
				baseCoin := Value.PureLovelaceValue(0)
				output = TransactionOutput.SimpleTransactionOutput(ChangeAddress, baseCoin)
				tempMultiAsset = MultiAsset.MultiAsset[int64]{}
				tempValue = Value.Value{}
				tempAssets = Asset.Asset[int64]{}
			}

			tempAssets = tempAssets.Add(Asset.Asset[int64]{asset_name: amount})
		}
		tempMultiAsset = tempMultiAsset.Add(MultiAsset.MultiAsset[int64]{policyId: tempAssets})
		tempValue.SetMultiAsset(tempMultiAsset)
		output.SetAmount(output.GetValue().Add(tempValue))
		updatedAmount := output.GetValue().Clone()
		required_lovelace := Utils.MinLovelacePostAlonzo(TransactionOutput.SimpleTransactionOutput(ChangeAddress, updatedAmount), tb.Context)
		updatedAmount.SetLovelace(required_lovelace)
		cbor, _ := cbor.Marshal(updatedAmount)
		maxValSz, _ := strconv.Atoi(maxValSize)
		if len(cbor) > maxValSz {
			output.SetAmount(oldAmount)
			break
		}
	}
	multiAssetArray = append(multiAssetArray, output.GetValue().GetAssets())
	return multiAssetArray
}

func (tb *TransactionBuilder) _CalcChange(fees int64, inputs []UTxO.UTxO, outputs []TransactionOutput.TransactionOutput, address Address.Address, preciseFee bool, respectMinUtxo bool) ([]TransactionOutput.TransactionOutput, error) {
	changeOutputArr := make([]TransactionOutput.TransactionOutput, 0)
	requested := Value.SimpleValue(fees, MultiAsset.MultiAsset[int64]{})
	for _, output := range outputs {
		requested = requested.Add(output.GetValue())
	}
	provided := Value.Value{}
	for _, input := range inputs {
		provided = provided.Add(input.Output.GetValue())
	}
	if tb.Mint != nil {
		provided.AddAssets(tb.Mint)
	}
	//TODO: Implement withdrawals
	provided.SubLovelace(tb._GetTotalKeyDeposit())
	if !requested.Less(provided) {
		return changeOutputArr, &Errors.InvalidTransactionException{inputs, outputs, fees}
	}
	change := provided.Sub(requested)
	if change.HasAssets {
		multiAsset := change.GetAssets()
		for policyId, assets := range multiAsset {
			for assetName, amount := range assets {
				if amount == 0 {
					delete(multiAsset[policyId], assetName)
				}
			}
			if len(multiAsset[policyId]) == 0 {
				delete(multiAsset, policyId)
			}
		}
		change.SetMultiAsset(multiAsset)
	}

	if !change.HasAssets {
		minLovelace := Utils.MinLovelacePostAlonzo(
			TransactionOutput.SimpleTransactionOutput(address, change), tb.Context)
		if respectMinUtxo && change.GetCoin() < minLovelace {
			return changeOutputArr, &CoinSelection.InsufficientUtxoBalanceError{
				fmt.Sprintf("The change output %v is less than the minimum Lovelace value %v", change.GetCoin(), minLovelace)}
		}
		lovelace_change := Value.PureLovelaceValue(change.GetCoin())
		changeOutputArr = append(changeOutputArr, TransactionOutput.SimpleTransactionOutput(address, lovelace_change))
	}
	if change.HasAssets {
		multiAssetArray := tb._pack_multiassets_for_change(address, change, tb.Context.GetProtocolParams().MaxValSize)
		for i, multiAsset := range multiAssetArray {
			if respectMinUtxo && change.GetCoin() < Utils.MinLovelacePostAlonzo(TransactionOutput.SimpleTransactionOutput(address, Value.SimpleValue(0, multiAsset)), tb.Context) {
				return changeOutputArr, &CoinSelection.InsufficientUtxoBalanceError{
					fmt.Sprintf("Not Enough Ada left to cover non-ADA assets in change address")}
			}
			var changeValue Value.Value
			if i == len(multiAssetArray)-1 {
				changeValue = Value.SimpleValue(change.GetCoin(), multiAsset)
			} else {
				changeValue = Value.SimpleValue(0, multiAsset)
				changeValue.SetLovelace(Utils.MinLovelacePostAlonzo(TransactionOutput.SimpleTransactionOutput(address, changeValue), tb.Context))
			}
			changeOutputArr = append(changeOutputArr, TransactionOutput.SimpleTransactionOutput(address, changeValue))
			change = change.Sub(changeValue)
		}
	}
	return changeOutputArr, nil
}

func (tb *TransactionBuilder) _MergeChanges(changes []TransactionOutput.TransactionOutput, change_output_index int) {
	if change_output_index != -1 && len(changes) == 1 {
		tb.Outputs[change_output_index].SetAmount(tb.Outputs[change_output_index].GetValue().Add(changes[0].GetValue()))
	} else {
		tb.Outputs = append(tb.Outputs, changes...)
	}

}

func (tb *TransactionBuilder) _AddChangeAndFee(
	changeAddress *Address.Address,
	mergeChange bool) error {
	ogInputs := Utils.Copy(tb.Inputs)
	ogOutputs := Utils.Copy(tb.Outputs)
	changeOutputIndex := -1
	if changeAddress != nil {
		if mergeChange {
			for i, output := range ogOutputs {
				if changeAddress == output.GetAddressPointer() {
					if changeOutputIndex == -1 || output.GetValue().GetCoin() == 0 {
						changeOutputIndex = i
					}
				}
			}
		}
		tb.Fee = tb._EstimateFee()
		changes, err := tb._CalcChange(tb.Fee, tb.Inputs, tb.Outputs, *changeAddress, true, !mergeChange)
		if err != nil {
			return err
		}
		tb._MergeChanges(changes, changeOutputIndex)
	}
	tb.Fee = tb._EstimateFee()

	if changeAddress != nil {

		tb.Outputs = ogOutputs
		changes, err := tb._CalcChange(tb.Fee, ogInputs, ogOutputs, *changeAddress, true, !mergeChange)

		if err != nil {
			return err
		}
		tb._MergeChanges(changes, changeOutputIndex)
	}
	return nil
}

func (tb *TransactionBuilder) _EstimateFee() int64 {
	plutusExecutionUnits := Redeemer.ExecutionUnits{Mem: 0, Steps: 0}
	for _, redeemer := range tb.Redeemers() {
		plutusExecutionUnits.Sum(redeemer.ExUnits)
	}
	fullFakeTx, _ := tb._BuildFullFakeTx()
	fakeTxBytes, _ := cbor.Marshal(fullFakeTx)
	estimatedFee := Utils.Fee(tb.Context, len(fakeTxBytes), plutusExecutionUnits.Steps, plutusExecutionUnits.Mem, tb.ReferenceInputs)
	return estimatedFee
}

func (tb *TransactionBuilder) _ScriptDataHash() *serialization.ScriptDataHash {
	if len(tb.Datums) > 0 || len(tb.Redeemers()) > 0 {
		witnessSet := tb.BuildWitnessSet()
		sdh, _ := ScriptDataHash(
			tb.Context,
			witnessSet,
		)
		return &serialization.ScriptDataHash{Payload: sdh.Payload}

	}
	return nil
}

func ScriptDataHash(chainContext Base.ChainContext, witnessSet TransactionWitnessSet.TransactionWitnessSet) (*serialization.ScriptDataHash, error) {
	cost_models := map[int]cbor.Marshaler{}
	redeemers := witnessSet.Redeemer
	PV1Scripts := witnessSet.PlutusV1Script
	PV2Scripts := witnessSet.PlutusV2Script
	datums := witnessSet.PlutusData

	isV1 := len(PV1Scripts) > 0
	if len(redeemers) > 0 {
		if len(PV2Scripts) > 0 {
			cost_models = PlutusData.COST_MODELSV2
		} else if !isV1 {
			cost_models = PlutusData.COST_MODELSV2
		}
	}
	var redeemer_bytes []byte

	if redeemers == nil {
		if chainContext.Epoch() > 506 {
			redeemer_bytes, _ = hex.DecodeString("A0")
		} else {
			redeemer_bytes, _ = cbor.Marshal([]Redeemer.Redeemer{})
		}
	} else {
		redeemer_bytes, _ = cbor.Marshal(redeemers)
	}

	var datum_bytes []byte
	if datums.Len() > 0 {

		datum_bytes, err = cbor.Marshal(datums)
		if err != nil {
			return nil, err
		}
	} else {
		datum_bytes = []byte{}
	}
	var cost_model_bytes []byte
	if isV1 {
		cost_model_bytes, err = cbor.Marshal(PlutusData.COST_MODELSV1)
		if err != nil {
			return nil, err
		}

	} else {
		cost_model_bytes, err = cbor.Marshal(cost_models)
		if err != nil {
			return nil, err
		}
	}
	total_bytes := append(redeemer_bytes, datum_bytes...)
	total_bytes = append(total_bytes, cost_model_bytes...)
	hash, err := serialization.Blake2bHash(total_bytes)
	if err != nil {
		return nil, err
	}
	return &serialization.ScriptDataHash{hash}, nil

}

func (tb *TransactionBuilder) _BuildTxBody() TransactionBody.TransactionBody {
	inputs := make([]TransactionInput.TransactionInput, 0)
	for _, input := range tb.Inputs {
		inputs = append(inputs, input.Input)
	}
	collaterals := make([]TransactionInput.TransactionInput, 0)
	for _, collateral := range tb.Collaterals {
		collaterals = append(collaterals, collateral.Input)
	}
	data_hash := tb._ScriptDataHash()
	script_data_hash := make([]byte, 0)
	if data_hash != nil {
		script_data_hash = data_hash.Payload
	}
	aux_data_hash := tb.AuxiliaryData.Hash()
	return TransactionBody.TransactionBody{
		Inputs:            inputs,
		Outputs:           tb.Outputs,
		Fee:               tb.Fee,
		Ttl:               tb.Ttl,
		Mint:              tb.Mint,
		AuxiliaryDataHash: aux_data_hash,
		ScriptDataHash:    script_data_hash,
		RequiredSigners:   tb.RequiredSigners,
		ValidityStart:     tb.ValidityStart,
		Collateral:        collaterals,
		// Certificates:      tb.Certificates,
		// Withdrawals:       tb.Withdrawals,
		CollateralReturn: tb.CollateralReturn,
		TotalCollateral:  int(tb.TotalCollateral),
		ReferenceInputs:  tb.ReferenceInputs,
	}
}

func (tb *TransactionBuilder) _InputVkeyHashes() []serialization.PubKeyHash {
	result := make([]serialization.PubKeyHash, 0)
	for _, input := range append(tb.Inputs, tb.Collaterals...) {
		pkh := serialization.PubKeyHash{}
		copy(pkh[:], input.Output.GetAddress().PaymentPart)
		result = append(result, pkh)
	}
	return result
}

// func (tb *TransactionBuilder) _BuildFakeVkeyWitnesses() []serialization.VerificationKeyWitness {
// 	vkey_hashes := tb._InputVkeyHashes()
// 	vkey_hashes = append(vkey_hashes, tb.RequiredSigners...)
// 	//vkey_hashes = append(vkey_hashes, tb._NativeScriptVkeyHashes()...)
// 	//vkey_hashes = append(vkey_hashes, tb._CertificateVkeyHashes()...)
// 	//vkey_hashes = append(vkey_hashes, tb._WithdrawalVkeyHashes()...)
// 	result := make([]serialization.VerificationKeyWitness, 0)
// 	for _, vkey_hash := range vkey_hashes {
// 		result = append(result, serialization.VerificationKeyWitness{Vkey: serialization.VerificationKey{vkey_hash.Payload}, Signature: fake_tx_signature})
// 	}
// 	return result
// }

func (tb *TransactionBuilder) _BuildFakeWitnessSet() TransactionWitnessSet.TransactionWitnessSet {
	witnessSet := tb.BuildWitnessSet()
	//witnessSet.VkeyWitnesses = tb._BuildFakeVkeyWitnesses()
	return witnessSet
}
func (tb *TransactionBuilder) AllScripts() []PlutusData.ScriptHashable {
	allscripts := []PlutusData.ScriptHashable{}
	allscripts = append(allscripts, tb.NativeScripts...)
	for _, s := range tb.InputsToScripts {
		allscripts = append(allscripts, s)
	}
	for _, s := range tb.MintingScriptToRedeemers {
		allscripts = append(allscripts, s.Script)
	}
	return allscripts
}

func (tb *TransactionBuilder) Scripts() ([]NativeScript.NativeScript, []PlutusData.PlutusV1Script, []PlutusData.PlutusV2Script) {
	nativeScripts := make([]NativeScript.NativeScript, 0)
	plutusV1Scripts := make([]PlutusData.PlutusV1Script, 0)
	plutusV2Scripts := make([]PlutusData.PlutusV2Script, 0)
	redeemers := tb.Redeemers()
	if len(tb.Datums) > 0 || len(redeemers) > 0 || len(tb.NativeScripts) > 0 {
		for _, script := range tb.AllScripts() {
			switch script.(type) {
			case NativeScript.NativeScript:
				nativeScripts = append(nativeScripts, script.(NativeScript.NativeScript))
			case PlutusData.PlutusV1Script:
				plutusV1Scripts = append(plutusV1Scripts, script.(PlutusData.PlutusV1Script))
			case PlutusData.PlutusV2Script:
				plutusV2Scripts = append(plutusV2Scripts, script.(PlutusData.PlutusV2Script))
			}
		}
	}
	return nativeScripts, plutusV1Scripts, plutusV2Scripts
}

func (tb *TransactionBuilder) _BuildFullFakeTx() (Transaction.Transaction, error) {
	tmp_builder := tb.Copy()
	txBody := tmp_builder._BuildTxBody()
	if txBody.Fee == 0 {
		txBody.Fee = int64(tmp_builder.Context.MaxTxFee())
	}
	witness := tmp_builder._BuildFakeWitnessSet()
	tx := Transaction.Transaction{
		TransactionBody:       txBody,
		TransactionWitnessSet: witness,
	}
	bytes, _ := cbor.Marshal(tx)
	if len(bytes) > tmp_builder.Context.GetProtocolParams().MaxTxSize {
		return tx, &Errors.TransactionTooBigError{
			fmt.Sprintf("Transaction is too big, %d bytes, max is %d", len(bytes), tmp_builder.Context.GetProtocolParams().MaxTxSize)}
	}
	return tx, nil
}

func (tb *TransactionBuilder) BuildWitnessSet() TransactionWitnessSet.TransactionWitnessSet {
	nativeScripts, plutusV1Scripts, plutusV2Scripts := tb.Scripts()
	plutusdata := make([]PlutusData.PlutusData, 0)
	for _, datum := range tb.Datums {
		plutusdata = append(plutusdata, datum)
	}
	if len(plutusdata) == 0 {
		return TransactionWitnessSet.TransactionWitnessSet{
			NativeScripts:  nativeScripts,
			PlutusV1Script: plutusV1Scripts,
			PlutusV2Script: plutusV2Scripts,
			PlutusData:     nil,
			Redeemer:       tb.Redeemers(),
		}
	}
	return TransactionWitnessSet.TransactionWitnessSet{
		NativeScripts:  nativeScripts,
		PlutusV1Script: plutusV1Scripts,
		PlutusV2Script: plutusV2Scripts,
		PlutusData:     PlutusData.PlutusIndefArray(plutusdata),
		Redeemer:       tb.Redeemers(),
	}
}

func (tb *TransactionBuilder) _EnsureNoInputExclusionConflict() error {
	for _, input := range tb.Inputs {
		for _, excluded := range tb.ExcludedInputs {
			if reflect.DeepEqual(input, excluded) {
				return &Errors.InputExclusionError{fmt.Sprintf("Input %v is both included and excluded", input.Input)}
			}
		}
	}
	return nil
}

func (tb *TransactionBuilder) _SetCollateralReturn(changeAddress *Address.Address) error {
	witnesses := tb._BuildFakeWitnessSet()
	if len(witnesses.PlutusV1Script) == 0 &&
		len(witnesses.PlutusV2Script) == 0 &&
		len(tb.ReferenceScripts) == 0 {
		return nil
	}

	if changeAddress == nil {
		return nil
	}
	collateral_amount := 5_000_000 //tb.Context.MaxTxFee() * tb.Context.GetProtocolParams().CollateralPercent / 100
	total_input := Value.Value{}
	for _, utxo := range tb.Collaterals {
		total_input = total_input.Add(utxo.Output.GetValue())
	}
	if int64(collateral_amount) > total_input.GetCoin() {
		return errors.New("Not enough collateral to cover fee")
	}
	return_amount := total_input.GetCoin() - int64(collateral_amount)
	min_lovelace := Utils.MinLovelacePostAlonzo(TransactionOutput.SimpleTransactionOutput(*changeAddress, Value.PureLovelaceValue(return_amount)), tb.Context)
	if min_lovelace > return_amount {
		return errors.New("Not enough collateral to cover fee")
	} else {
		returnOutput := TransactionOutput.SimpleTransactionOutput(*changeAddress, Value.PureLovelaceValue(return_amount))
		tb.CollateralReturn = &returnOutput
		tb.TotalCollateral = int64(collateral_amount)
	}
	return nil
}

func (tb *TransactionBuilder) Build(changeAddress *Address.Address, mergeChange bool, collateralChangeAddress *Address.Address) (TransactionBody.TransactionBody, error) {
	err := tb._EnsureNoInputExclusionConflict()
	if err != nil {
		return TransactionBody.TransactionBody{}, err
	}
	selectedUtxos := make([]UTxO.UTxO, 0)
	selectedAmount := Value.Value{}
	for _, input := range tb.Inputs {
		selectedUtxos = append(selectedUtxos, input)
		selectedAmount = selectedAmount.Add(input.Output.GetValue())
	}

	// TODO figure out how to handle generic type conversion.... (Mint is int64 but it can only ever be uint in a Value)
	if tb.Mint != nil {
		selectedAmount.AddAssets(tb.Mint)
	}
	if tb.Withdrawals != nil {
		//TODO: implement WIthdrawals
	}

	canMergeChange := false
	if mergeChange {
		for _, output := range tb.Outputs {
			addr := output.GetAddress()
			if addr.Equal(changeAddress) {
				canMergeChange = true
				break
			}
		}
	}

	selectedAmount.SubLovelace(tb._GetTotalKeyDeposit())
	requestedAmount := Value.Value{}
	for _, output := range tb.Outputs {
		requestedAmount = requestedAmount.Add(output.GetValue())
	}
	requestedAmount.AddLovelace(tb._EstimateFee())

	trimmedSelectedAmount := Value.SimpleValue(selectedAmount.GetCoin(),
		selectedAmount.GetAssets().Filter(func(policy Policy.PolicyId, asset AssetName.AssetName, quantity int64) bool {
			for requestedPolicy, requestedAsset := range requestedAmount.GetAssets() {
				for requestedAssetName, _ := range requestedAsset {
					if requestedPolicy == policy && requestedAssetName == asset {
						return true
					}
				}
			}
			return false
		}),
	)
	unfulfilledAmount := requestedAmount.Sub(trimmedSelectedAmount)
	if changeAddress != nil && !canMergeChange {
		if unfulfilledAmount.GetCoin() < 0 {
			estimated := unfulfilledAmount.GetCoin() +
				Utils.MinLovelacePostAlonzo(
					TransactionOutput.SimpleTransactionOutput(*changeAddress, selectedAmount.Sub(trimmedSelectedAmount)), tb.Context)
			if estimated < 0 {
				estimated = 0
			}
			unfulfilledAmount.SetLovelace(
				estimated)
		}
	} else {
		if unfulfilledAmount.GetCoin() < 0 {
			unfulfilledAmount.SetLovelace(0)
		}
	}
	unfulfilledAmount = unfulfilledAmount.RemoveZeroAssets()

	emptyVal := Value.Value{}
	if emptyVal.Less(unfulfilledAmount) && !(unfulfilledAmount.GetCoin() == 0 && len(unfulfilledAmount.GetAssets()) == 0) {
		additionalUtxoPool := make([]UTxO.UTxO, 0)
		additionalAmount := Value.Value{}
		if tb.LoadedUtxos == nil {
			for _, address := range tb.InputAddresses {
				for _, utxo := range tb.Context.Utxos(address) {
					if !Utils.Contains(selectedUtxos, utxo) &&
						!Utils.Contains(tb.ExcludedInputs, utxo) &&
						len(utxo.Output.GetDatumHash().Payload) == 0 {
						additionalUtxoPool = append(additionalUtxoPool, utxo)
						additionalAmount = additionalAmount.Add(utxo.Output.GetValue())
					}
				}
			}
		} else {
			for _, utxo := range tb.LoadedUtxos {
				if !Utils.Contains(selectedUtxos, utxo) &&
					!Utils.Contains(tb.ExcludedInputs, utxo) &&
					len(utxo.Output.GetDatumHash().Payload) == 0 {
					additionalUtxoPool = append(additionalUtxoPool, utxo)
					additionalAmount = additionalAmount.Add(utxo.Output.GetValue())
				}
			}
		}
		for _, selector := range tb.UtxoSelectors {
			selected, _, err := selector.Select(
				additionalUtxoPool,
				[]TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(FAKE_ADDRESS, unfulfilledAmount)},
				tb.Context,
				-1,
				false,
				!canMergeChange,
			)
			if err != nil {
				//TODO MULTI SELECTOR
				return TransactionBody.TransactionBody{}, err
			}
			for _, s := range selected {
				selectedUtxos = append(selectedUtxos, s)
				selectedAmount = selectedAmount.Add(s.Output.GetValue())
			}
			break
		}
	}
	tb.Inputs = selectedUtxos[:]

	tb._SetRedeemerIndex()
	if collateralChangeAddress != nil {
		tb._SetCollateralReturn(collateralChangeAddress)
	} else {
		tb._SetCollateralReturn(changeAddress)
	}

	tb._UpdateExecutionUnits(changeAddress, mergeChange, collateralChangeAddress)

	err = tb._AddChangeAndFee(changeAddress, mergeChange)
	if err != nil {
		return TransactionBody.TransactionBody{}, err
	}
	tx_body := tb._BuildTxBody()
	return tx_body, nil

}

func (tb *TransactionBuilder) Copy() *TransactionBuilder {
	InputsToRedeemers := make(map[string]Redeemer.Redeemer)
	for k, v := range tb.InputsToRedeemers {
		InputsToRedeemers[k] = v.Clone()
	}

	return &TransactionBuilder{
		tb.Context,
		tb.UtxoSelectors,
		tb.ExecutionMemoryBuffer,
		tb.ExecutionStepBuffer,
		tb.Ttl,
		tb.ValidityStart,
		Utils.Copy(tb.LoadedUtxos),
		tb.AuxiliaryData,
		tb.NativeScripts,
		tb.Mint,
		tb.RequiredSigners,
		tb.Collaterals,
		tb.Certificates,
		tb.Withdrawals,
		tb.ReferenceInputs,
		Utils.Copy(tb.Inputs),
		Utils.Copy(tb.ExcludedInputs),
		tb.InputAddresses,
		Utils.Copy(tb.Outputs),
		tb.Fee,
		tb.Datums,
		tb.CollateralReturn,
		tb.TotalCollateral,
		InputsToRedeemers,
		tb.MintingScriptToRedeemers,
		tb.InputsToScripts,
		tb.ReferenceScripts,
		false,
	}
}

func (tb *TransactionBuilder) _EstimateExecutionUnits(changeAddress *Address.Address, mergeChange bool, collateralChangeAddress *Address.Address) map[string]Redeemer.ExecutionUnits {
	tmp_builder := tb.Copy()
	tmp_builder.ShouldEstimateExecutionUnits = false
	tx_body, _ := tmp_builder.Build(changeAddress, mergeChange, collateralChangeAddress)
	witness_set := tb._BuildFakeWitnessSet()
	tx := Transaction.Transaction{TransactionBody: tx_body, TransactionWitnessSet: witness_set, Valid: false}
	tx_cbor, _ := cbor.Marshal(tx)
	return tb.Context.EvaluateTx(tx_cbor)

}

func (tb *TransactionBuilder) _UpdateExecutionUnits(changeAddress *Address.Address, mergeChange bool, collateralChangeAddress *Address.Address) {
	if tb.ShouldEstimateExecutionUnits {
		estimated_execution_units := tb._EstimateExecutionUnits(changeAddress, mergeChange, collateralChangeAddress)
		for k, redeemer := range tb.InputsToRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				tb.InputsToRedeemers[k] = redeemer
			}
		}

	}
}

func SortInputs(inputs []UTxO.UTxO) []UTxO.UTxO {
	hashes := make([]string, 0)
	relationMap := map[string]UTxO.UTxO{}
	for _, utxo := range inputs {
		hashes = append(hashes, string(utxo.Input.TransactionId))
		relationMap[string(utxo.Input.TransactionId)] = utxo
	}
	sort.Strings(hashes)
	sorted_inputs := make([]UTxO.UTxO, 0)
	for _, hash := range hashes {
		sorted_inputs = append(sorted_inputs, relationMap[hash])
	}
	return sorted_inputs
}

func (tb *TransactionBuilder) _SetRedeemerIndex() {
	sorted_inputs := SortInputs(tb.Inputs)
	done := make([]string, 0)
	for i, utxo := range sorted_inputs {
		utxo_cbor := Utils.ToCbor(utxo)
		if slices.Contains(done, utxo_cbor) {
			continue
		}
		val, ok := tb.InputsToRedeemers[utxo_cbor]
		if ok && val.Tag == Redeemer.SPEND {
			done = append(done, utxo_cbor)
			redeem := tb.InputsToRedeemers[utxo_cbor]
			redeem.Index = i
			tb.InputsToRedeemers[utxo_cbor] = redeem
		} else if ok && val.Tag == Redeemer.MINT {
			//TODO: IMPLEMENT FOR MINTS
		}
	}
	// for script,redeemer := range tb.MintingScriptToRedeemers {
	// 	//TODO IMPLEMENT THIS
	// }

}
