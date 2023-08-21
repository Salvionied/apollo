package apollo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/Salvionied/apollo/apollotypes"
	"github.com/Salvionied/apollo/constants"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/HDWallet"
	"github.com/Salvionied/apollo/serialization/Key"
	"github.com/Salvionied/apollo/serialization/Metadata"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/NativeScript"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/apollo/serialization/VerificationKeyWitness"
	"github.com/Salvionied/apollo/serialization/Withdrawal"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Utils"
	"github.com/Salvionied/cbor/v2"
	"golang.org/x/exp/slices"
)

const (
	EX_MEMORY_BUFFER = 0.2
	EX_STEP_BUFFER   = 0.2
)

type Apollo struct {
	Context            Base.ChainContext
	payments           []PaymentI
	isEstimateRequired bool
	auxiliaryData      *Metadata.AuxiliaryData
	utxos              []UTxO.UTxO
	preselectedUtxos   []UTxO.UTxO
	inputAddresses     []Address.Address
	tx                 *Transaction.Transaction
	datums             map[string]PlutusData.PlutusData
	requiredSigners    []serialization.PubKeyHash
	v1scripts          []PlutusData.PlutusV1Script
	v2scripts          []PlutusData.PlutusV2Script
	redeemers          []Redeemer.Redeemer
	redeemersToUTxO    map[string]Redeemer.Redeemer
	mint               []Unit
	collaterals        []UTxO.UTxO
	Fee                int64
	Ttl                int64
	ValidityStart      int64
	totalCollateral    int
	referenceInputs    []TransactionInput.TransactionInput
	collateralReturn   *TransactionOutput.TransactionOutput
	withdrawals        []*Withdrawal.Withdrawal
	certificates       *Certificate.Certificates
	nativescripts      []NativeScript.NativeScript
	usedUtxos          []string
	referenceScripts   []PlutusData.ScriptHashable
	wallet             apollotypes.Wallet
}

func New(cc Base.ChainContext) *Apollo {
	return &Apollo{
		Context:            cc,
		payments:           []PaymentI{},
		isEstimateRequired: false,
		auxiliaryData:      &Metadata.AuxiliaryData{},
		utxos:              []UTxO.UTxO{},
		preselectedUtxos:   []UTxO.UTxO{},
		inputAddresses:     []Address.Address{},
		tx:                 nil,
		datums:             make(map[string]PlutusData.PlutusData),
		requiredSigners:    make([]serialization.PubKeyHash, 0),
		v1scripts:          make([]PlutusData.PlutusV1Script, 0),
		v2scripts:          make([]PlutusData.PlutusV2Script, 0),
		redeemers:          make([]Redeemer.Redeemer, 0),
		redeemersToUTxO:    make(map[string]Redeemer.Redeemer),
		mint:               make([]Unit, 0),
		collaterals:        make([]UTxO.UTxO, 0),
		Fee:                0,
		usedUtxos:          make([]string, 0),
		referenceInputs:    make([]TransactionInput.TransactionInput, 0),
		referenceScripts:   make([]PlutusData.ScriptHashable, 0)}
}

func (b *Apollo) GetWallet() apollotypes.Wallet {
	return b.wallet
}

func (b *Apollo) AddInput(utxos ...UTxO.UTxO) *Apollo {
	b.preselectedUtxos = append(b.preselectedUtxos, utxos...)
	return b
}

func (b *Apollo) ConsumeUTxO(utxo UTxO.UTxO, payments ...PaymentI) *Apollo {
	b.preselectedUtxos = append(b.preselectedUtxos, utxo)
	selectedValue := utxo.Output.GetAmount()
	for _, payment := range payments {
		selectedValue = selectedValue.Sub(payment.ToValue())
	}
	if selectedValue.Less(Value.Value{}) {
		fmt.Println("selected value is negative")
		return b
	}
	b.payments = append(b.payments, payments...)
	p := NewPaymentFromValue(utxo.Output.GetAddress(), selectedValue)
	b.payments = append(b.payments, p)
	return b
}

func (b *Apollo) ConsumeAssetsFromUtxo(utxo UTxO.UTxO, payments ...PaymentI) *Apollo {
	b.preselectedUtxos = append(b.preselectedUtxos, utxo)
	selectedValue := utxo.Output.GetAmount()
	for _, payment := range payments {
		selectedValue = selectedValue.Sub(Value.SimpleValue(0, payment.ToValue().GetAssets()))
	}
	if selectedValue.Less(Value.Value{}) {
		fmt.Println("selected value is negative")
		return b
	}
	b.payments = append(b.payments, payments...)
	p := NewPaymentFromValue(utxo.Output.GetAddress(), selectedValue)
	b.payments = append(b.payments, p)
	return b
}

func (b *Apollo) AddLoadedUTxOs(utxos ...UTxO.UTxO) *Apollo {
	b.utxos = append(b.utxos, utxos...)
	return b
}

func (b *Apollo) AddInputAddress(address Address.Address) *Apollo {
	b.inputAddresses = append(b.inputAddresses, address)
	return b

}
func (b *Apollo) AddInputAddressFromBech32(address string) *Apollo {
	decoded_addr, _ := Address.DecodeAddress(address)
	b.inputAddresses = append(b.inputAddresses, decoded_addr)
	return b
}

func (b *Apollo) AddPayment(payment PaymentI) *Apollo {
	b.payments = append(b.payments, payment)
	return b
}

func (b *Apollo) PayToAddressBech32(address string, lovelace int, units ...Unit) *Apollo {
	decoded_addr, _ := Address.DecodeAddress(address)
	return b.AddPayment(&Payment{lovelace, decoded_addr, units, nil, nil, false})
}

func (b *Apollo) PayToAddress(address Address.Address, lovelace int, units ...Unit) *Apollo {
	return b.AddPayment(&Payment{lovelace, address, units, nil, nil, false})
}

func (b *Apollo) AddDatum(pd *PlutusData.PlutusData) *Apollo {
	hash := hex.EncodeToString(PlutusData.PlutusDataHash(pd).Payload)
	b.datums[hash] = *pd
	return b
}

func (b *Apollo) PayToContract(contractAddress Address.Address, pd *PlutusData.PlutusData, lovelace int, isInline bool, units ...Unit) *Apollo {
	if isInline {
		b = b.AddPayment(&Payment{lovelace, contractAddress, units, pd, nil, isInline})
	} else if pd != nil {
		dataHash := PlutusData.PlutusDataHash(pd)
		b = b.AddPayment(&Payment{lovelace, contractAddress, units, pd, dataHash.Payload, isInline})
	} else {
		b = b.AddPayment(&Payment{lovelace, contractAddress, units, nil, nil, isInline})
	}
	if pd != nil && !isInline {
		b = b.AddDatum(pd)
	}
	return b
}

func (b *Apollo) AddRequiredSignerFromBech32(address string, addPaymentPart, addStakingPart bool) *Apollo {
	decoded_addr, _ := Address.DecodeAddress(address)
	if addPaymentPart {
		b.requiredSigners = append(b.requiredSigners, serialization.PubKeyHash(decoded_addr.PaymentPart[0:28]))

	}
	if addStakingPart {
		b.requiredSigners = append(b.requiredSigners, serialization.PubKeyHash(decoded_addr.PaymentPart[0:28]))
	}
	return b

}

func (b *Apollo) AddRequiredSigner(pkh serialization.PubKeyHash) *Apollo {
	b.requiredSigners = append(b.requiredSigners, pkh)
	return b
}

func (b *Apollo) AddRequiredSignerFromAddress(address Address.Address, addPaymentPart, addStakingPart bool) *Apollo {
	if addPaymentPart {
		pkh := serialization.PubKeyHash(address.PaymentPart)

		b.requiredSigners = append(b.requiredSigners, pkh)

	}
	if addStakingPart {
		pkh := serialization.PubKeyHash(address.StakingPart)

		b.requiredSigners = append(b.requiredSigners, pkh)

	}
	return b
}

func (b *Apollo) buildOutputs() []TransactionOutput.TransactionOutput {
	outputs := make([]TransactionOutput.TransactionOutput, 0)
	for _, payment := range b.payments {
		outputs = append(outputs, *payment.ToTxOut())
	}
	return outputs

}

func (b *Apollo) buildWitnessSet() TransactionWitnessSet.TransactionWitnessSet {
	plutusdata := make([]PlutusData.PlutusData, 0)
	for _, datum := range b.datums {
		plutusdata = append(plutusdata, datum)
	}
	if len(plutusdata) == 0 {
		return TransactionWitnessSet.TransactionWitnessSet{
			NativeScripts:  b.nativescripts,
			PlutusV1Script: b.v1scripts,
			PlutusV2Script: b.v2scripts,
			Redeemer:       b.redeemers,
			PlutusData:     nil,
		}
	}
	pd := PlutusData.PlutusIndefArray(plutusdata)
	return TransactionWitnessSet.TransactionWitnessSet{
		NativeScripts:  b.nativescripts,
		PlutusV1Script: b.v1scripts,
		PlutusV2Script: b.v2scripts,
		PlutusData:     &pd,
		Redeemer:       b.redeemers,
	}
}

func (b *Apollo) scriptDataHash() *serialization.ScriptDataHash {
	if len(b.datums) == 0 && len(b.redeemers) == 0 {
		return nil
	}
	witnessSet := b.buildWitnessSet()
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
	if redeemers == nil {
		redeemers = []Redeemer.Redeemer{}
	}
	redeemer_bytes, err := cbor.Marshal(redeemers)
	if err != nil {
		log.Fatal(err)
	}
	var datum_bytes []byte
	if datums.Len() > 0 {

		datum_bytes, err = cbor.Marshal(datums)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		datum_bytes = []byte{}
	}
	var cost_model_bytes []byte
	if isV1 {
		cost_model_bytes, err = cbor.Marshal(PlutusData.COST_MODELSV1)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		cost_model_bytes, err = cbor.Marshal(cost_models)
		if err != nil {
			log.Fatal(err)
		}
	}
	total_bytes := append(redeemer_bytes, datum_bytes...)
	total_bytes = append(total_bytes, cost_model_bytes...)
	return &serialization.ScriptDataHash{Payload: serialization.Blake2bHash(total_bytes)}

}

func (b *Apollo) getMints() MultiAsset.MultiAsset[int64] {
	ma := make(MultiAsset.MultiAsset[int64])
	for _, mintUnit := range b.mint {
		ma = ma.Add(mintUnit.ToValue().GetAssets())
	}
	return ma
}

func (b *Apollo) buildTxBody() TransactionBody.TransactionBody {
	inputs := make([]TransactionInput.TransactionInput, 0)
	for _, utxo := range b.preselectedUtxos {
		inputs = append(inputs, utxo.Input)
	}
	collaterals := make([]TransactionInput.TransactionInput, 0)
	for _, utxo := range b.collaterals {
		collaterals = append(collaterals, utxo.Input)
	}
	dataHash := b.scriptDataHash()
	scriptDataHash := make([]byte, 0)
	if dataHash != nil {
		scriptDataHash = dataHash.Payload
	}
	aux_data_hash := b.auxiliaryData.Hash()
	mints := b.getMints()
	txb := TransactionBody.TransactionBody{
		Inputs:            inputs,
		Outputs:           b.buildOutputs(),
		Fee:               b.Fee,
		Ttl:               b.Ttl,
		Mint:              mints,
		AuxiliaryDataHash: aux_data_hash,
		ScriptDataHash:    scriptDataHash,
		RequiredSigners:   b.requiredSigners,
		ValidityStart:     b.ValidityStart,
		Collateral:        collaterals,
		Certificates:      b.certificates,
		Withdrawals:       b.withdrawals,
		ReferenceInputs:   b.referenceInputs}
	if b.totalCollateral != 0 {
		txb.TotalCollateral = b.totalCollateral
		txb.CollateralReturn = b.collateralReturn
	}
	return txb
}

func (b *Apollo) buildFullFakeTx() (*Transaction.Transaction, error) {
	txBody := b.buildTxBody()
	if txBody.Fee == 0 {
		txBody.Fee = int64(b.Context.MaxTxFee())
	}
	witness := b.buildWitnessSet()
	tx := Transaction.Transaction{
		TransactionBody:       txBody,
		TransactionWitnessSet: witness}
	bytes := tx.Bytes()
	if len(bytes) > b.Context.GetProtocolParams().MaxTxSize {
		return nil, errors.New("transaction too large")
	}
	return &tx, nil
}

func (b *Apollo) estimateFee() int64 {
	pExU := Redeemer.ExecutionUnits{Mem: 0, Steps: 0}
	for _, redeemer := range b.redeemers {
		pExU.Sum(redeemer.ExUnits)
	}
	fftx, err := b.buildFullFakeTx()
	if err != nil {
		return 0
	}
	fakeTxBytes := fftx.Bytes()
	estimatedFee := Utils.Fee(b.Context, len(fakeTxBytes)+500, pExU.Steps, pExU.Mem)
	return estimatedFee

}

func (b *Apollo) getAvailableUtxos() []UTxO.UTxO {
	availableUtxos := make([]UTxO.UTxO, 0)
	for _, utxo := range b.utxos {
		if !slices.Contains(b.usedUtxos, utxo.GetKey()) {
			availableUtxos = append(availableUtxos, utxo)
		}
	}
	return availableUtxos
}

func (b *Apollo) setRedeemerIndexes() *Apollo {
	sorted_inputs := SortInputs(b.preselectedUtxos)
	done := make([]string, 0)
	for i, utxo := range sorted_inputs {
		utxo_cbor := Utils.ToCbor(utxo)
		if slices.Contains(done, utxo_cbor) {
			continue
		}
		val, ok := b.redeemersToUTxO[utxo_cbor]
		if ok && val.Tag == Redeemer.SPEND {
			done = append(done, utxo_cbor)
			redeem := b.redeemersToUTxO[utxo_cbor]
			redeem.Index = i
			b.redeemersToUTxO[utxo_cbor] = redeem
		} else if ok && val.Tag == Redeemer.MINT {
			//TODO: IMPLEMENT FOR MINTS
		}
	}
	return b
}

func (b *Apollo) AttachDatum(datum *PlutusData.PlutusData) *Apollo {
	hash := PlutusData.HashDatum(datum)

	b.datums[hex.EncodeToString(hash.Payload)] = *datum
	return b
}

func (b *Apollo) setCollateral() *Apollo {
	if len(b.collaterals) > 0 {
		return b
	}
	witnesses := b.buildWitnessSet()
	if len(witnesses.PlutusV1Script) == 0 &&
		len(witnesses.PlutusV2Script) == 0 &&
		len(b.referenceScripts) == 0 {
		return b
	}
	availableUtxos := b.getAvailableUtxos()
	collateral_amount := 5_000_000
	for _, utxo := range availableUtxos {
		if int(utxo.Output.GetValue().GetCoin()) > collateral_amount && len(utxo.Output.GetAmount().GetAssets()) == 0 {

			return_amount := utxo.Output.GetValue().GetCoin() - int64(collateral_amount)
			min_lovelace := Utils.MinLovelacePostAlonzo(TransactionOutput.SimpleTransactionOutput(b.inputAddresses[0], Value.PureLovelaceValue(return_amount)), b.Context)
			if min_lovelace > return_amount {
				continue
			} else {
				returnOutput := TransactionOutput.SimpleTransactionOutput(b.inputAddresses[0], Value.PureLovelaceValue(return_amount))
				b.collaterals = append(b.collaterals, utxo)
				b.collateralReturn = &returnOutput
				b.totalCollateral = collateral_amount
				return b
			}
		}
	}
	return b
}

func (b *Apollo) Clone() *Apollo {
	clone := *b
	return &clone
}

func (b *Apollo) estimateExunits() map[string]Redeemer.ExecutionUnits {
	cloned_b := b.Clone()
	cloned_b.isEstimateRequired = false
	updated_b, _ := cloned_b.Complete()
	//updated_b = updated_b.fakeWitness()
	tx_cbor, _ := cbor.Marshal(updated_b.tx)
	return b.Context.EvaluateTx(tx_cbor)
}
func (b *Apollo) updateExUnits() *Apollo {
	if b.isEstimateRequired {
		estimated_execution_units := b.estimateExunits()
		for k, redeemer := range b.redeemersToUTxO {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.redeemersToUTxO[k] = redeemer
			}
		}
		for _, redeemer := range b.redeemersToUTxO {
			b.redeemers = append(b.redeemers, redeemer)
		}
	} else {
		for _, redeemer := range b.redeemersToUTxO {
			b.redeemers = append(b.redeemers, redeemer)
		}
	}
	return b
}

func (b *Apollo) GetTx() *Transaction.Transaction {
	return b.tx
}

func (b *Apollo) Complete() (*Apollo, error) {
	selectedUtxos := make([]UTxO.UTxO, 0)
	selectedAmount := Value.Value{}
	for _, utxo := range b.preselectedUtxos {
		selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
	}
	for _, mintUnit := range b.mint {
		selectedAmount = selectedAmount.Add(mintUnit.ToValue())
	}

	// for _, withdrawal := range b.withdrawals {
	// 	//TODO
	// }
	requestedAmount := Value.Value{}
	for _, payment := range b.payments {
		payment.EnsureMinUTXO(b.Context)
		requestedAmount = requestedAmount.Add(payment.ToValue())
	}
	requestedAmount.AddLovelace(b.estimateFee() + constants.MIN_LOVELACE)
	unfulfilledAmount := requestedAmount.Sub(selectedAmount)
	unfulfilledAmount = unfulfilledAmount.RemoveZeroAssets()
	available_utxos := SortUtxos(b.getAvailableUtxos())
	//BALANCE TX
	if unfulfilledAmount.GreaterOrEqual(Value.Value{}) {
		//BALANCE
		if len(unfulfilledAmount.GetAssets()) > 0 {
			//BALANCE WITH ASSETS
			for pol, assets := range unfulfilledAmount.GetAssets() {
				for asset, amt := range assets {
					found := false
					selectedSoFar := int64(0)
					for idx, utxo := range available_utxos {
						ma := utxo.Output.GetValue().GetAssets()
						if ma.GetByPolicyAndId(pol, asset) >= amt {
							selectedUtxos = append(selectedUtxos, utxo)
							selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
							if idx+1 <= len(available_utxos) {
								available_utxos = append(available_utxos[:idx], available_utxos[idx+1:]...)
							} else {
								available_utxos = available_utxos[:idx]
							}
							b.usedUtxos = append(b.usedUtxos, utxo.GetKey())
							found = true
							break
						} else if ma.GetByPolicyAndId(pol, asset) > 0 {
							selectedUtxos = append(selectedUtxos, utxo)
							selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
							if idx+1 <= len(available_utxos) {
								available_utxos = append(available_utxos[:idx], available_utxos[idx+1:]...)
							} else {
								available_utxos = available_utxos[:idx]
							}
							b.usedUtxos = append(b.usedUtxos, utxo.GetKey())
							selectedSoFar += ma.GetByPolicyAndId(pol, asset)
							if selectedSoFar > amt {
								found = true
								break
							}
						}
					}
					if !found {
						return nil, errors.New("missing required assets")
					}

				}
			}
		}
		for {

			if selectedAmount.Greater(requestedAmount.Add(Value.Value{Am: Amount.Amount{}, Coin: 1_000_000, HasAssets: false})) {
				break
			}
			if len(available_utxos) == 0 {
				fmt.Println(selectedAmount.Greater(requestedAmount.Add(Value.Value{Am: Amount.Amount{}, Coin: 1_000_000, HasAssets: false})))
				fmt.Println("HERE")
				fmt.Println(selectedAmount, requestedAmount.Add(Value.Value{Am: Amount.Amount{}, Coin: 1_000_000, HasAssets: false}))

				return nil, errors.New("not enough funds")
			}
			utxo := available_utxos[0]
			selectedUtxos = append(selectedUtxos, utxo)
			selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
			available_utxos = available_utxos[1:]
			b.usedUtxos = append(b.usedUtxos, utxo.GetKey())
		}

	}
	// ADD NEW SELECTED INPUTS TO PRE SELECTION
	b.preselectedUtxos = append(b.preselectedUtxos, selectedUtxos...)

	//SET REDEEMER INDEXES
	b = b.setRedeemerIndexes()
	//SET COLLATERAL
	b = b.setCollateral()
	//UPDATE EXUNITS
	b = b.updateExUnits()
	//ADDCHANGEANDFEE
	b = b.addChangeAndFee()
	//FINALIZE TX
	body := b.buildTxBody()
	witnessSet := b.buildWitnessSet()
	b.tx = &Transaction.Transaction{TransactionBody: body, TransactionWitnessSet: witnessSet, AuxiliaryData: b.auxiliaryData, Valid: true}
	return b, nil
}

func (b *Apollo) addChangeAndFee() *Apollo {
	providedAmount := Value.Value{}
	for _, utxo := range b.preselectedUtxos {
		providedAmount = providedAmount.Add(utxo.Output.GetValue())
	}
	requestedAmount := Value.Value{}
	for _, payment := range b.payments {
		requestedAmount = requestedAmount.Add(payment.ToValue())
	}
	b.Fee = b.estimateFee()
	requestedAmount.AddLovelace(b.Fee)
	change := providedAmount.Sub(requestedAmount)
	if change.GetCoin() < Utils.MinLovelacePostAlonzo(
		TransactionOutput.SimpleTransactionOutput(b.inputAddresses[0], change),
		b.Context,
	) {
		fmt.Println("not enough funds")
		//TODO FIX
	}
	payment := Payment{
		Receiver: b.inputAddresses[0],
		Lovelace: int(change.GetCoin()),
		Units:    make([]Unit, 0),
	}
	for policy, assets := range change.GetAssets() {
		for asset, amt := range assets {
			if amt > 0 {
				payment.Units = append(payment.Units, Unit{
					PolicyId: policy.String(),
					Name:     asset.String(),
					Quantity: int(amt),
				})
			}
		}
	}
	b.payments = append(b.payments, &payment)
	return b
}

func (b *Apollo) CollectFrom(
	inputUtxo UTxO.UTxO,
	redeemer Redeemer.Redeemer,
) *Apollo {
	b.isEstimateRequired = true
	b.preselectedUtxos = append(b.preselectedUtxos, inputUtxo)
	b.redeemersToUTxO[hex.EncodeToString(inputUtxo.Input.TransactionId)] = redeemer
	return b
}

func (b *Apollo) AttachV1Script(script PlutusData.PlutusV1Script) *Apollo {
	b.v1scripts = append(b.v1scripts, script)

	return b
}
func (b *Apollo) AttachV2Script(script PlutusData.PlutusV2Script) *Apollo {
	b.v2scripts = append(b.v2scripts, script)

	return b
}

func (a *Apollo) SetWalletFromMnemonic(mnemonic string) *Apollo {
	paymentPath := "m/1852'/1815'/0'/0/0"
	stakingPath := "m/1852'/1815'/0'/2/0"
	hdWall := HDWallet.NewHDWalletFromMnemonic(mnemonic, "")
	paymentKeyPath := hdWall.DerivePath(paymentPath)
	verificationKey_bytes := paymentKeyPath.XPrivKey.PublicKey()
	signingKey_bytes := paymentKeyPath.XPrivKey.Bytes()
	stakingKeyPath := hdWall.DerivePath(stakingPath)
	stakeVerificationKey_bytes := stakingKeyPath.XPrivKey.PublicKey()
	stakeSigningKey_bytes := stakingKeyPath.XPrivKey.Bytes()
	//stake := stakingKeyPath.RootXprivKey.Bytes()
	signingKey := Key.SigningKey{Payload: signingKey_bytes}
	verificationKey := Key.VerificationKey{Payload: verificationKey_bytes}
	stakeSigningKey := Key.StakeSigningKey{Payload: stakeSigningKey_bytes}
	stakeVerificationKey := Key.StakeVerificationKey{Payload: stakeVerificationKey_bytes}
	stakeVerKey := Key.VerificationKey{Payload: stakeVerificationKey_bytes}
	skh, _ := stakeVerKey.Hash()
	vkh, _ := verificationKey.Hash()

	addr := Address.Address{StakingPart: skh[:], PaymentPart: vkh[:], Network: 1, AddressType: Address.KEY_KEY, HeaderByte: 0b00000001, Hrp: "addr"}
	wallet := apollotypes.GenericWallet{SigningKey: signingKey, VerificationKey: verificationKey, Address: addr, StakeSigningKey: stakeSigningKey, StakeVerificationKey: stakeVerificationKey}
	a.wallet = &wallet
	return a
}

func (a *Apollo) SetWalletFromBech32(address string) *Apollo {
	addr, err := Address.DecodeAddress(address)
	if err != nil {
		fmt.Println(err)
		return a
	}
	a.wallet = &apollotypes.ExternalWallet{Address: addr}
	return a
}

func (b *Apollo) SetWalletAsChangeAddress() *Apollo {
	if b.wallet == nil {
		fmt.Println("Wallet not set")
		return b
	}
	switch b.Context.(type) {
	case *BlockFrostChainContext.BlockFrostChainContext:

		utxos := b.Context.Utxos(*b.wallet.GetAddress())
		b = b.AddLoadedUTxOs(utxos...)
	default:
	}
	b.inputAddresses = append(b.inputAddresses, *b.wallet.GetAddress())
	return b
}
func (b *Apollo) Sign() *Apollo {
	signatures := b.wallet.SignTx(*b.tx)
	b.tx.TransactionWitnessSet = signatures
	return b
}

func (b *Apollo) Submit() (serialization.TransactionId, error) {
	return b.Context.SubmitTx(*b.tx)
}

func (b *Apollo) LoadTxCbor(txCbor string) (*Apollo, error) {
	tx := Transaction.Transaction{}
	err := cbor.Unmarshal([]byte(txCbor), &tx)
	if err != nil {
		return b, err
	}
	b.tx = &tx
	return b, nil
}

func (b *Apollo) UtxoFromRef(txHash string, txIndex int) *UTxO.UTxO {
	utxo := b.Context.GetUtxoFromRef(txHash, txIndex)
	if utxo == nil {
		return nil
	}
	return utxo

}

func (b *Apollo) AddVerificationKeyWitness(vkw VerificationKeyWitness.VerificationKeyWitness) *Apollo {
	b.tx.TransactionWitnessSet.VkeyWitnesses = append(b.tx.TransactionWitnessSet.VkeyWitnesses, vkw)
	return b
}

func (b *Apollo) SetChangeAddressBech32(address string) *Apollo {
	addr, err := Address.DecodeAddress(address)
	if err != nil {
		fmt.Println(err)
	}
	b.inputAddresses = append(b.inputAddresses, addr)
	return b
}

func (b *Apollo) SetChangeAddress(address Address.Address) *Apollo {
	b.inputAddresses = append(b.inputAddresses, address)
	return b
}

func (b *Apollo) SetTtl(ttl int64) *Apollo {
	b.Ttl = ttl
	return b
}

func (b *Apollo) SetValidityStart(invalidBefore int64) *Apollo {
	b.ValidityStart = invalidBefore
	return b
}

func (b *Apollo) SetShelleyMetadata(metadata Metadata.ShelleyMaryMetadata) *Apollo {
	if b.auxiliaryData == nil {
		b.auxiliaryData = &Metadata.AuxiliaryData{}
		b.auxiliaryData.SetShelleyMetadata(metadata)
	} else {
		b.auxiliaryData.SetShelleyMetadata(metadata)
	}
	return b
}

func (b *Apollo) GetUsedUTxOs() []string {
	return b.usedUtxos
}

func (b *Apollo) SetEstimationExUnitsRequired() *Apollo {
	b.isEstimateRequired = true
	return b
}

func (b *Apollo) AddReferenceInput(txHash string, index int) *Apollo {
	decodedHash, _ := hex.DecodeString(txHash)
	input := TransactionInput.TransactionInput{
		TransactionId: decodedHash,
		Index:         index,
	}
	utxo := b.UtxoFromRef(txHash, index)
	ref := utxo.Output.GetScriptRef()
	fmt.Println(ref)
	b.referenceInputs = append(b.referenceInputs, input)
	return b
}
