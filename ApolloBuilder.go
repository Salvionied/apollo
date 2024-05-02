package apollo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/Salvionied/cbor/v2"
	"github.com/SundaeSwap-finance/apollo/apollotypes"
	"github.com/SundaeSwap-finance/apollo/constants"
	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/Amount"
	"github.com/SundaeSwap-finance/apollo/serialization/Certificate"
	"github.com/SundaeSwap-finance/apollo/serialization/HDWallet"
	"github.com/SundaeSwap-finance/apollo/serialization/Key"
	"github.com/SundaeSwap-finance/apollo/serialization/Metadata"
	"github.com/SundaeSwap-finance/apollo/serialization/MultiAsset"
	"github.com/SundaeSwap-finance/apollo/serialization/NativeScript"
	"github.com/SundaeSwap-finance/apollo/serialization/PlutusData"
	"github.com/SundaeSwap-finance/apollo/serialization/Redeemer"
	"github.com/SundaeSwap-finance/apollo/serialization/Transaction"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionBody"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionInput"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionOutput"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionWitnessSet"
	"github.com/SundaeSwap-finance/apollo/serialization/UTxO"
	"github.com/SundaeSwap-finance/apollo/serialization/Value"
	"github.com/SundaeSwap-finance/apollo/serialization/VerificationKeyWitness"
	"github.com/SundaeSwap-finance/apollo/serialization/Withdrawal"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/Base"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Utils"
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
	datums             []PlutusData.PlutusData
	requiredSigners    []serialization.PubKeyHash
	v1scripts          []PlutusData.PlutusV1Script
	v2scripts          []PlutusData.PlutusV2Script
	redeemers          []Redeemer.Redeemer
	mintRedeemers      []Redeemer.Redeemer
	redeemersToUTxO    map[string]Redeemer.Redeemer
	stakeRedeemers     map[string]Redeemer.Redeemer
	mint               []Unit
	collaterals        []UTxO.UTxO
	Fee                int64
	FeePadding         int64
	Ttl                int64
	ValidityStart      int64
	totalCollateral    int
	referenceInputs    []TransactionInput.TransactionInput
	collateralReturn   *TransactionOutput.TransactionOutput
	withdrawals        Withdrawal.Withdrawal
	certificates       *Certificate.Certificates
	nativescripts      []NativeScript.NativeScript
	usedUtxos          []string
	referenceScripts   []PlutusData.ScriptHashable
	wallet             apollotypes.Wallet
	scriptHashes       []string
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
		datums:             make([]PlutusData.PlutusData, 0),
		requiredSigners:    make([]serialization.PubKeyHash, 0),
		v1scripts:          make([]PlutusData.PlutusV1Script, 0),
		v2scripts:          make([]PlutusData.PlutusV2Script, 0),
		redeemers:          make([]Redeemer.Redeemer, 0),
		redeemersToUTxO:    make(map[string]Redeemer.Redeemer),
		stakeRedeemers:     make(map[string]Redeemer.Redeemer),
		mint:               make([]Unit, 0),
		collaterals:        make([]UTxO.UTxO, 0),
		withdrawals:        Withdrawal.New(),
		Fee:                0,
		FeePadding:         0,
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
	b.datums = append(b.datums, *pd)
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

func (b *Apollo) SetFeePadding(padding int64) *Apollo {
	b.FeePadding = padding
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
	plutusdata = append(plutusdata, b.datums...)
	return TransactionWitnessSet.TransactionWitnessSet{
		NativeScripts:  b.nativescripts,
		PlutusV1Script: b.v1scripts,
		PlutusV2Script: b.v2scripts,
		PlutusData:     PlutusData.PlutusIndefArray(plutusdata),
		Redeemer:       b.redeemers,
	}
}

func (b *Apollo) buildFakeWitnessSet() TransactionWitnessSet.TransactionWitnessSet {
	plutusdata := make([]PlutusData.PlutusData, 0)
	plutusdata = append(plutusdata, b.datums...)
	fakeVkWitnesses := make([]VerificationKeyWitness.VerificationKeyWitness, 0)
	fakeVkWitnesses = append(fakeVkWitnesses, VerificationKeyWitness.VerificationKeyWitness{
		Vkey:      constants.FAKE_VKEY,
		Signature: constants.FAKE_SIGNATURE})
	for range b.requiredSigners {
		fakeVkWitnesses = append(fakeVkWitnesses, VerificationKeyWitness.VerificationKeyWitness{
			Vkey:      constants.FAKE_VKEY,
			Signature: constants.FAKE_SIGNATURE})
	}
	return TransactionWitnessSet.TransactionWitnessSet{
		NativeScripts:  b.nativescripts,
		PlutusV1Script: b.v1scripts,
		PlutusV2Script: b.v2scripts,
		PlutusData:     PlutusData.PlutusIndefArray(plutusdata),
		Redeemer:       b.redeemers,
		VkeyWitnesses:  fakeVkWitnesses,
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

func (b *Apollo) MintAssets(mintUnit Unit) *Apollo {
	b.mint = append(b.mint, mintUnit)
	return b
}

func (b *Apollo) MintAssetsWithRedeemer(mintUnit Unit, redeemerData PlutusData.PlutusData) *Apollo {
	b.mint = append(b.mint, mintUnit)
	newRedeemer := Redeemer.Redeemer{
		Tag:     Redeemer.MINT,
		Index:   0, // This will be computed later when we iterate over mintRedeemers
		Data:    redeemerData,
		ExUnits: Redeemer.ExecutionUnits{},
	}
	b.mintRedeemers = append(b.mintRedeemers, newRedeemer)
	return b
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
	var withdrawals *Withdrawal.Withdrawal
	if b.withdrawals != nil && b.withdrawals.Size() > 0 {
		withdrawals = &b.withdrawals
	}
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
		Withdrawals:       withdrawals,
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
	witness := b.buildFakeWitnessSet()
	tx := Transaction.Transaction{
		TransactionBody:       txBody,
		TransactionWitnessSet: witness,
		Valid:                 true,
		AuxiliaryData:         b.auxiliaryData}
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
	estimatedFee := Utils.Fee(b.Context, len(fakeTxBytes), pExU.Steps, pExU.Mem)
	estimatedFee += b.FeePadding
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
		key := hex.EncodeToString(utxo.Input.TransactionId) + fmt.Sprint(utxo.Input.Index)
		val, ok := b.redeemersToUTxO[key]
		if ok && val.Tag == Redeemer.SPEND {
			done = append(done, key)
			redeem := b.redeemersToUTxO[key]
			redeem.Index = i
			b.redeemersToUTxO[key] = redeem
		} else if ok && val.Tag == Redeemer.MINT {
			//TODO: IMPLEMENT FOR MINTS
		}
	}
	return b
}

func (b *Apollo) AttachDatum(datum *PlutusData.PlutusData) *Apollo {
	b.datums = append(b.datums, *datum)
	return b
}

func (b *Apollo) setCollateral() *Apollo {
	if len(b.collaterals) > 0 {
		return b
	}
	witnesses := b.buildWitnessSet()
	if len(witnesses.PlutusV1Script) == 0 &&
		len(witnesses.PlutusV2Script) == 0 &&
		len(b.referenceInputs) == 0 {
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

func (b *Apollo) estimateExunits() (map[string]Redeemer.ExecutionUnits, []byte, error) {
	cloned_b := b.Clone()
	cloned_b.isEstimateRequired = false
	updated_b, _, _ := cloned_b.Complete()
	//updated_b = updated_b.fakeWitness()
	tx_cbor, _ := cbor.Marshal(updated_b.tx)
	result, err := b.Context.EvaluateTx(tx_cbor)
	return result, tx_cbor, err
}
func (b *Apollo) updateExUnits() (*Apollo, []byte, error) {
	if b.isEstimateRequired {
		estimated_execution_units, tx_cbor, err := b.estimateExunits()
		if err != nil {
			return nil, tx_cbor, err
		}
		for k, redeemer := range b.redeemersToUTxO {
			key := fmt.Sprintf("%s:%d", Redeemer.RedeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.redeemersToUTxO[k] = redeemer
			}
		}
		for k, redeemer := range b.stakeRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RedeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.stakeRedeemers[k] = redeemer
			}
		}
		for k, redeemer := range b.mintRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RedeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.mintRedeemers[k] = redeemer
			}
		}
		for _, redeemer := range b.redeemersToUTxO {
			b.redeemers = append(b.redeemers, redeemer)
		}
		for _, redeemer := range b.stakeRedeemers {
			b.redeemers = append(b.redeemers, redeemer)
		}
		for _, redeemer := range b.mintRedeemers {
			b.redeemers = append(b.redeemers, redeemer)
		}
	} else {
		for _, redeemer := range b.redeemersToUTxO {
			b.redeemers = append(b.redeemers, redeemer)
		}
		for _, redeemer := range b.stakeRedeemers {
			b.redeemers = append(b.redeemers, redeemer)
		}
		for _, redeemer := range b.mintRedeemers {
			b.redeemers = append(b.redeemers, redeemer)
		}
	}
	return b, nil, nil
}

func (b *Apollo) GetTx() *Transaction.Transaction {
	return b.tx
}

// If this fails due to a script failure, it returns the failed tx cbor as
// bytes for diagnostic purposes.
func (b *Apollo) Complete() (*Apollo, []byte, error) {
	selectedUtxos := make([]UTxO.UTxO, 0)
	selectedAmount := Value.Value{}
	for _, utxo := range b.preselectedUtxos {
		selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
	}
	burnedValue := b.GetBurns()
	selectedAmount = selectedAmount.Add(burnedValue)
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
					usedIdxs := make([]int, 0)
					for idx, utxo := range available_utxos {
						ma := utxo.Output.GetValue().GetAssets()
						if ma.GetByPolicyAndId(pol, asset) >= amt {
							selectedUtxos = append(selectedUtxos, utxo)
							selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
							usedIdxs = append(usedIdxs, idx)
							b.usedUtxos = append(b.usedUtxos, utxo.GetKey())
							found = true
							break
						} else if ma.GetByPolicyAndId(pol, asset) > 0 {
							selectedUtxos = append(selectedUtxos, utxo)
							selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
							usedIdxs = append(usedIdxs, idx)
							b.usedUtxos = append(b.usedUtxos, utxo.GetKey())
							selectedSoFar += ma.GetByPolicyAndId(pol, asset)
							if selectedSoFar >= amt {
								found = true
								break
							}
						}
					}
					newAvailUtxos := make([]UTxO.UTxO, 0)
					for idx, availutxo := range available_utxos {
						if !slices.Contains(usedIdxs, idx) {
							newAvailUtxos = append(newAvailUtxos, availutxo)
						}
					}
					available_utxos = newAvailUtxos
					if !found {
						return nil, nil, errors.New("missing required assets")
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
				fmt.Println(selectedAmount, requestedAmount.Add(Value.Value{Am: Amount.Amount{}, Coin: 1_000_000, HasAssets: false}))

				return nil, nil, errors.New("not enough funds")
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
	b, tx_cbor, err := b.updateExUnits()
	if err != nil {
		return nil, tx_cbor, err
	}
	//ADDCHANGEANDFEE
	b = b.addChangeAndFee()
	//FINALIZE TX
	body := b.buildTxBody()
	witnessSet := b.buildWitnessSet()
	b.tx = &Transaction.Transaction{TransactionBody: body, TransactionWitnessSet: witnessSet, AuxiliaryData: b.auxiliaryData, Valid: true}
	return b, nil, nil
}

func isOverUtxoLimit(change Value.Value, address Address.Address, b Base.ChainContext) bool {
	txOutput := TransactionOutput.SimpleTransactionOutput(address, Value.SimpleValue(0, change.GetAssets()))
	encoded, _ := cbor.Marshal(txOutput)
	maxValSize, _ := strconv.Atoi(b.GetProtocolParams().MaxValSize)
	//fmt.Println(len(encoded), maxValSize)
	return len(encoded) > maxValSize

}

func splitPayments(c Value.Value, a Address.Address, b Base.ChainContext) []*Payment {
	lovelace := c.GetCoin()
	assets := c.GetAssets()
	payments := make([]*Payment, 0)
	newPayment := new(Payment)
	newPayment.Receiver = a
	newPayment.Lovelace = 0
	newPayment.Units = make([]Unit, 0)
	for policy, assets := range assets {
		for asset, amt := range assets {
			if !isOverUtxoLimit(newPayment.ToValue(), a, b) {
				if amt > 0 {
					newPayment.Units = append(newPayment.Units, Unit{
						PolicyId: policy.String(),
						Name:     asset.String(),
						Quantity: int(amt),
					})
				}
			} else {

				minLovelace := Utils.MinLovelacePostAlonzo(
					*newPayment.ToTxOut(), b)
				newPayment.Lovelace = int(minLovelace)
				lovelace -= minLovelace
				payments = append(payments, newPayment)
				newPayment = new(Payment)
				newPayment.Receiver = a
				newPayment.Lovelace = 0
				newPayment.Units = make([]Unit, 0)
				if amt > 0 {
					newPayment.Units = append(newPayment.Units, Unit{
						PolicyId: policy.String(),
						Name:     asset.String(),
						Quantity: int(amt),
					})
				}
			}
		}
	}
	payments = append(payments, newPayment)

	payments[len(payments)-1].Lovelace += int(lovelace)
	totalCoin := 0
	for _, payment := range payments {
		totalCoin += payment.Lovelace
	}
	return payments

}

func (b *Apollo) GetBurns() (burns Value.Value) {
	burns = Value.Value{}
	for _, mintUnit := range b.mint {
		if mintUnit.Quantity < 0 {
			usedUnit := Unit{
				PolicyId: mintUnit.PolicyId,
				Name:     mintUnit.Name,
				Quantity: -mintUnit.Quantity,
			}
			burns = burns.Add(usedUnit.ToValue())
		}

	}
	return burns
}

func (b *Apollo) addChangeAndFee() *Apollo {
	burns := b.GetBurns()
	providedAmount := Value.Value{}
	for _, utxo := range b.preselectedUtxos {
		providedAmount = providedAmount.Add(utxo.Output.GetValue())
	}
	providedAmount = providedAmount.Sub(burns)
	requestedAmount := Value.Value{}
	for _, payment := range b.payments {
		requestedAmount = requestedAmount.Add(payment.ToValue())
	}
	b.Fee = b.estimateFee()
	requestedAmount.AddLovelace(b.Fee)
	change := providedAmount.Sub(requestedAmount)

	if change.GetCoin() < Utils.MinLovelacePostAlonzo(
		TransactionOutput.SimpleTransactionOutput(b.inputAddresses[0], Value.SimpleValue(0, change.GetAssets())),
		b.Context,
	) {
		sortedUtxos := SortUtxos(b.getAvailableUtxos())
		b.preselectedUtxos = append(b.preselectedUtxos, sortedUtxos[0])
		b.usedUtxos = append(b.usedUtxos, sortedUtxos[0].GetKey())
		return b.addChangeAndFee()
	}
	if isOverUtxoLimit(change, b.inputAddresses[0], b.Context) {
		adjustedPayments := splitPayments(change, b.inputAddresses[0], b.Context)
		pp := b.payments[:]
		for _, payment := range adjustedPayments {
			b.payments = append(b.payments, payment)
		}
		newestFee := b.estimateFee()
		if newestFee > b.Fee {
			difference := newestFee - b.Fee
			adjustedPayments[len(adjustedPayments)-1].Lovelace -= int(difference)
			b.Fee = newestFee
			b.payments = pp
			for _, payment := range adjustedPayments {
				b.payments = append(b.payments, payment)
			}
		}

	} else {
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
		pp := b.payments[:]
		b.payments = append(b.payments, &payment)

		newestFee := b.estimateFee()
		if newestFee > b.Fee {
			difference := newestFee - b.Fee
			payment.Lovelace -= int(difference)
			b.payments = append(pp, &payment)
			b.Fee = newestFee
		}
	}
	return b
}

func (b *Apollo) CollectFrom(
	inputUtxo UTxO.UTxO,
	redeemerData PlutusData.PlutusData,
) *Apollo {
	b.isEstimateRequired = true
	b.preselectedUtxos = append(b.preselectedUtxos, inputUtxo)
	b.usedUtxos = append(b.usedUtxos, inputUtxo.GetKey())
	newRedeemer := Redeemer.Redeemer{
		Tag:     Redeemer.SPEND,
		Index:   0, // This will be computed later when we iterate over redeemersToUTxO
		Data:    redeemerData,
		ExUnits: Redeemer.ExecutionUnits{},
	}
	b.redeemersToUTxO[hex.EncodeToString(inputUtxo.Input.TransactionId)+fmt.Sprint(inputUtxo.Input.Index)] = newRedeemer
	return b
}

func (b *Apollo) AttachV1Script(script PlutusData.PlutusV1Script) *Apollo {
	hash := PlutusData.PlutusScriptHash(script)
	for _, scriptHash := range b.scriptHashes {
		if scriptHash == hex.EncodeToString(hash.Bytes()) {
			return b
		}
	}
	b.v1scripts = append(b.v1scripts, script)
	b.scriptHashes = append(b.scriptHashes, hex.EncodeToString(hash.Bytes()))

	return b
}
func (b *Apollo) AttachV2Script(script PlutusData.PlutusV2Script) *Apollo {
	hash := PlutusData.PlutusScriptHash(script)
	for _, scriptHash := range b.scriptHashes {
		if scriptHash == hex.EncodeToString(hash.Bytes()) {
			return b
		}
	}
	b.v2scripts = append(b.v2scripts, script)
	b.scriptHashes = append(b.scriptHashes, hex.EncodeToString(hash.Bytes()))
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

// For use with key pairs generated by cardano-cli
func (a *Apollo) SetWalletFromKeypair(vkey string, skey string, network constants.Network) *Apollo {
	verificationKey_bytes, err := hex.DecodeString(vkey)
	if err != nil {
		fmt.Println("SetWalletFromKeypair: Failed to decode vkey")
	}
	signingKey_bytes, err := hex.DecodeString(skey)
	if err != nil {
		fmt.Println("SetWalletFromKeypair: Failed to decode skey")
	}
	signingKey := Key.SigningKey{Payload: signingKey_bytes}
	verificationKey := Key.VerificationKey{Payload: verificationKey_bytes}
	vkh, _ := verificationKey.Hash()

	addr := Address.Address{}
	if network == constants.MAINNET {
		addr = Address.Address{StakingPart: nil, PaymentPart: vkh[:], Network: 1, AddressType: Address.KEY_NONE, HeaderByte: 0b01100001, Hrp: "addr"}
	} else {
		addr = Address.Address{StakingPart: nil, PaymentPart: vkh[:], Network: 0, AddressType: Address.KEY_NONE, HeaderByte: 0b01100000, Hrp: "addr_test"}
	}
	wallet := apollotypes.GenericWallet{
		SigningKey:           signingKey,
		VerificationKey:      verificationKey,
		Address:              addr,
		StakeSigningKey:      Key.StakeSigningKey{},
		StakeVerificationKey: Key.StakeVerificationKey{},
	}
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

func (b *Apollo) SignWithSkey(vkey Key.VerificationKey, skey Key.SigningKey) *Apollo {
	witness_set := b.GetTx().TransactionWitnessSet
	txHash := b.GetTx().TransactionBody.Hash()
	signature := skey.Sign(txHash)
	witness_set.VkeyWitnesses = append(witness_set.VkeyWitnesses, VerificationKeyWitness.VerificationKeyWitness{Vkey: vkey, Signature: signature})
	b.GetTx().TransactionWitnessSet = witness_set
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
	b.referenceInputs = append(b.referenceInputs, input)
	return b
}

func (b *Apollo) DisableExecutionUnitsEstimation() *Apollo {
	b.isEstimateRequired = false
	return b
}

func (b *Apollo) AddWithdrawal(address Address.Address, amount int, redeemerData PlutusData.PlutusData) *Apollo {
	var stakeAddr [29]byte
	stakeAddr[0] = address.HeaderByte
	if len(address.StakingPart) != 28 {
		fmt.Printf("AddWithdrawal: address has invalid or missing staking part: %v\n", address.StakingPart)
	}
	copy(stakeAddr[1:], address.StakingPart)
	err := b.withdrawals.Add(stakeAddr, amount)
	if err != nil {
		fmt.Printf("AddWithdrawal: %v\n", err)
		return b
	}
	newRedeemer := Redeemer.Redeemer{
		Tag:     Redeemer.REWARD,
		Index:   b.withdrawals.Size() - 1, // We just added a withdrawal
		Data:    redeemerData,
		ExUnits: Redeemer.ExecutionUnits{}, // This will be filled in when we eval later
	}
	b.stakeRedeemers[fmt.Sprint(b.withdrawals.Size()-1)] = newRedeemer
	return b
}
