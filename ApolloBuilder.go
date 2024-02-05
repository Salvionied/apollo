package apollo

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

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
	datums             []PlutusData.PlutusData
	requiredSigners    []serialization.PubKeyHash
	v1scripts          []PlutusData.PlutusV1Script
	v2scripts          []PlutusData.PlutusV2Script
	redeemers          []Redeemer.Redeemer
	redeemersToUTxO    map[string]Redeemer.Redeemer
	stakeRedeemers     map[string]Redeemer.Redeemer
	mintRedeemers      map[string]Redeemer.Redeemer
	mint               []Unit
	collaterals        []UTxO.UTxO
	Fee                int64
	FeePadding         int64
	Ttl                int64
	ValidityStart      int64
	totalCollateral    int
	referenceInputs    []TransactionInput.TransactionInput
	collateralReturn   *TransactionOutput.TransactionOutput
	withdrawals        *Withdrawal.Withdrawal
	certificates       *Certificate.Certificates
	nativescripts      []NativeScript.NativeScript
	usedUtxos          []string
	referenceScripts   []PlutusData.ScriptHashable
	wallet             apollotypes.Wallet
	scriptHashes       []string
}

/*
*

	New creates and initializes a new Apollo instance with the specified chain context,
	in which sets up various internal data structures for building and handling transactions.

	Params:
		cc (Base.ChainContext): The chain context to use for transaction building.

	Returns:
		*Apollo: A pointer to the initialized Apollo instance.
*/
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
		Fee:                0,
		FeePadding:         0,
		usedUtxos:          make([]string, 0),
		referenceInputs:    make([]TransactionInput.TransactionInput, 0),
		referenceScripts:   make([]PlutusData.ScriptHashable, 0),
		mintRedeemers:      make(map[string]Redeemer.Redeemer)}
}

/*
*

	GetWallet returns the wallet associated with the Apollo instance.

	Returns:
		apollotypes.Wallet: The wallet associated with the Apollo instance.
*/
func (b *Apollo) GetWallet() apollotypes.Wallet {
	return b.wallet
}

/*
*

	AddInput appends one or more UTxOs to the list of preselected
	UTxOs for transaction inputs.

	Params:
		utxos (...UTxO.UTxO): A set of UTxOs to be added as inputs.

	Returns:
		*Apollo: A pointer to the modified Apollo instance.
*/
func (b *Apollo) AddInput(utxos ...UTxO.UTxO) *Apollo {
	b.preselectedUtxos = append(b.preselectedUtxos, utxos...)
	return b
}

/*
*

	ConsumeUTxO adds a UTxO as an input to the transaction and deducts the specified payments from it.

	Params:
		utxo (UTxO.UTxO): The UTxO to be consumed as an input.
		payments (...PaymentI): A sett of payments to be deducted from the UTxO.

	Returns:
		*Apollo: A pointer to the modified Apollo instance.
*/
func (b *Apollo) ConsumeUTxO(utxo UTxO.UTxO, payments ...PaymentI) *Apollo {
	b.preselectedUtxos = append(b.preselectedUtxos, utxo)
	selectedValue := utxo.Output.GetAmount()
	for _, payment := range payments {
		selectedValue = selectedValue.Sub(payment.ToValue())
	}
	if selectedValue.Less(Value.Value{}) {
		panic("selected value is negative")
	}
	b.payments = append(b.payments, payments...)
	selectedValue = selectedValue.RemoveZeroAssets()
	p := NewPaymentFromValue(utxo.Output.GetAddress(), selectedValue)
	b.payments = append(b.payments, p)
	return b
}

/*
*

		ConsumeAssetsFromUtxo adds a UTxO as an input to the transaction and deducts the specified asset payments from it.

	 	Params:
	   		utxo (UTxO.UTxO): The UTxO to be consumed as an input.
	   		payments (...PaymentI): Asset payments to be deducted from the UTxO.

	 	Returns:
		   	*Apollo: A pointer to the modified Apollo instance.
*/
func (b *Apollo) ConsumeAssetsFromUtxo(utxo UTxO.UTxO, payments ...PaymentI) *Apollo {
	b.preselectedUtxos = append(b.preselectedUtxos, utxo)
	selectedValue := utxo.Output.GetAmount()
	for _, payment := range payments {
		selectedValue = selectedValue.Sub(Value.SimpleValue(0, payment.ToValue().GetAssets()))
	}
	if selectedValue.Less(Value.Value{}) {
		panic("selected value is negative")
		return b
	}
	b.payments = append(b.payments, payments...)
	selectedValue = selectedValue.RemoveZeroAssets()
	p := NewPaymentFromValue(utxo.Output.GetAddress(), selectedValue)
	b.payments = append(b.payments, p)
	return b
}

/*
*

	AddLoadedUTxOs appends one or more UTxOs to the list of loaded UTxOs.

	Params:
		utxos (...UTxO.UTxO): A set of UTxOs to be added to the loaded UTxOs.

	Returns:
		*Apollo: A pointer to the modified Apollo instance.
*/
func (b *Apollo) AddLoadedUTxOs(utxos ...UTxO.UTxO) *Apollo {
	b.utxos = append(b.utxos, utxos...)
	return b
}

/*
*

	AddInputAddress appends an input address to the list of input addresses for the transaction.

	Params:
		address (Address.Address): The input address to be added.

	Returns:
		*Apollo: A pointer to the modified Apollo instance.
*/
func (b *Apollo) AddInputAddress(address Address.Address) *Apollo {
	b.inputAddresses = append(b.inputAddresses, address)
	return b

}

/*
*

	AddInputAddressFromBech32 decodes a Bech32 address and
	appends it to the list of input addresses for the transaction.

	Params:
		address (string): The Bech32 address to be decoded and added

	Returns:
		*Apollo: A pointer to the modified Apollo instance.
*/
func (b *Apollo) AddInputAddressFromBech32(address string) *Apollo {
	decoded_addr, _ := Address.DecodeAddress(address)
	b.inputAddresses = append(b.inputAddresses, decoded_addr)
	return b
}

/*
*

	AddPayment appends a payment to the list of payments for the transaction.

	Params:
		payment (PaymentI): The payment to be added.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the payment added.
*/
func (b *Apollo) AddPayment(payment PaymentI) *Apollo {
	b.payments = append(b.payments, payment)
	return b
}

/*
*

	PayToAddressBech32 creates a payment to the specified Bech32 address
	with the given lovelace and units.

	Params:
		address (string): The Bech32 address to which the payment will be made.
		lovelace (int): The amount in lovelace to be paid.
		units (...Unit): The units (assets) to be paid along with the lovelace.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the payment added.
*/
func (b *Apollo) PayToAddressBech32(address string, lovelace int, units ...Unit) *Apollo {
	decoded_addr, _ := Address.DecodeAddress(address)
	return b.AddPayment(&Payment{lovelace, decoded_addr, units, nil, nil, false})
}

/*
*

		PayToAddress creates a payment to the specified address with the given lovelace and units,
		then adds it to the list of payment.

		Params:
			address (Address.Address): The recipient's address for the payment.
	   		lovelace (int): The amount in lovelace to send in the payment.
	   		units (...Unit): A set of units to include in the payment.

		Returns:
			*Apollo: A pointer to the modified Apollo instance with the payment added.
*/
func (b *Apollo) PayToAddress(address Address.Address, lovelace int, units ...Unit) *Apollo {
	return b.AddPayment(&Payment{lovelace, address, units, nil, nil, false})
}

/*
*

		AddDatum appends a Plutus datum to the list of data associated with the Apollo instance.

		Params:
	   		pd (*PlutusData.PlutusData): The Plutus datum to be added.

		Returns:
	   		*Apollo: A pointer to the modified Apollo instance with the datum added.
*/
func (b *Apollo) AddDatum(pd *PlutusData.PlutusData) *Apollo {
	b.datums = append(b.datums, *pd)
	return b
}

/*
*

		PayToContract creates a payment to a smart contract address and includes a Plutus datum, which
	 	is added to the list of payments, and if a datum is provided, it is added to the data list.

		Params:
		contractAddress (Address.Address): The smart contract address to send the payment to.
		pd (*PlutusData.PlutusData): Plutus datum to include in the payment.
		lovelace (int): The amount in lovelace to send in the payment.
		isInline (bool): Indicates if the payment is inline with the datum.
		units (...Unit): A set of units to include in the payment.

		Returns:
			*Apollo: A pointer to the modified Apollo instance with the payment and datum added.
*/
func (b *Apollo) PayToContract(
	contractAddress Address.Address,
	pd *PlutusData.PlutusData,
	lovelace int,
	isInline bool,
	units ...Unit,
) *Apollo {
	if isInline {
		b = b.AddPayment(&Payment{lovelace, contractAddress, units, pd, nil, isInline})
	} else if pd != nil {
		dataHash, _ := PlutusData.PlutusDataHash(pd)
		b = b.AddPayment(&Payment{lovelace, contractAddress, units, pd, dataHash.Payload, isInline})
	} else {
		b = b.AddPayment(&Payment{lovelace, contractAddress, units, nil, nil, isInline})
	}
	if pd != nil && !isInline {
		b = b.AddDatum(pd)
	}
	return b
}

/*
*

	AddRequiredSignerFromBech32 decodes an address in Bech32 format and adds
	its payment and staking parts as required signers.

	Params:


	address (string): The Bech32-encoded address to decode and add its parts as required signers.


	addPaymentPart (bool): Indicates whether to add the payment part as a required signer.


	   		addStakingPart (bool): Indicates whether to add the staking part as a required signer.

		Returns:
		   	*Apollo: A pointer to the modified Apollo instance with the required signers added.
*/
func (b *Apollo) AddRequiredSignerFromBech32(
	address string,
	addPaymentPart, addStakingPart bool,
) *Apollo {
	decoded_addr, _ := Address.DecodeAddress(address)
	if addPaymentPart {
		b.requiredSigners = append(
			b.requiredSigners,
			serialization.PubKeyHash(decoded_addr.PaymentPart[0:28]),
		)

	}
	if addStakingPart {
		b.requiredSigners = append(
			b.requiredSigners,
			serialization.PubKeyHash(decoded_addr.StakingPart[0:28]),
		)
	}
	return b

}

/*
*

		AddRequiredSigner appends a public key hash to the list of required signers.

	 	Params:
	   		pkh (serialization.PubKeyHash): The public key hash to add as a required signer.

	 	Returns:
	   		*Apollo: A pointer to the modified Apollo instance with the required signer added.
*/
func (b *Apollo) AddRequiredSigner(pkh serialization.PubKeyHash) *Apollo {
	b.requiredSigners = append(b.requiredSigners, pkh)
	return b
}

/*
*

		AddRequiredSignerFromAddress extracts the payment and staking parts from an address and adds them as required signers.

	 	Params:


	address (Address.Address): The address from which to extract the parts and add them as required signers.


	addPaymentPart (bool): Indicates whether to add the payment part as a required signer.


	  		addStakingPart (bool): Indicates whether to add the staking part as a required signer.

		Returns:
	  		*Apollo: A pointer to the modified Apollo instance with the required signers added.
*/
func (b *Apollo) AddRequiredSignerFromAddress(
	address Address.Address,
	addPaymentPart, addStakingPart bool,
) *Apollo {
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

/**
buildOutputs constructs and returns the transaction outputs based on the payments.

Returns:
	[]TransactionOutput.TransactionOutput: A slice of transaction outputs.
*/

func (b *Apollo) buildOutputs() []TransactionOutput.TransactionOutput {
	outputs := make([]TransactionOutput.TransactionOutput, 0)
	for _, payment := range b.payments {
		outputs = append(outputs, *payment.ToTxOut())
	}
	return outputs

}

/*
*

	buildWitnessSet constructs and returns the witness set for the transaction.

	Returns:
		TransactionWitnessSet.TransactionWitnessSet: The transaction's witness set.
*/
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

/*
*

	buildFakeWitnessSet constructs and returns a fake witness set used for testing.

	Returns:
		TransactionWitnessSet.TransactionWitnessSet: A fake witness set for testing.
*/
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

/**
scriptDataHash computes the hash of script data based on redeemers and datums.

Returns:
	*serialization.ScriptDataHash: The computed script data hash.
	error: An error if the scriptDataHash fails.
*/

func (b *Apollo) scriptDataHash() (*serialization.ScriptDataHash, error) {
	if len(b.datums) == 0 && len(b.redeemers) == 0 {
		return nil, nil
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
		return nil, err
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
	hashBytes, err := serialization.Blake2bHash(total_bytes)
	if err != nil {
		return nil, err
	}
	return &serialization.ScriptDataHash{Payload: hashBytes}, nil

}

/*
*

	getMints returns the multi-assets generated from minting.

	Returns:
		MultiAsset.MultiAsset[int64]: The generated multi-assets.
*/
func (b *Apollo) getMints() MultiAsset.MultiAsset[int64] {
	ma := make(MultiAsset.MultiAsset[int64])
	for _, mintUnit := range b.mint {
		ma = ma.Add(mintUnit.ToValue().GetAssets())
	}
	return ma
}

/*
*

	MintAssets adds a minting unit to the transaction's minting set.

	Params:
		mintUnit Unit: The minting unit to add.

	Returns:
		*Apollo: A pointer to the Apollo object to support method chaining.
*/
func (b *Apollo) MintAssets(mintUnit Unit) *Apollo {
	b.mint = append(b.mint, mintUnit)
	return b
}

/*
*

	MintAssetsWithRedeemer adds a minting unit with an associated redeemer to the transaction's minting set.

	Params:
		mintUnit Unit: The minting unit to add.
		redeemer Redeemer.Redeemer: The redeemer associated with the minting unit.

	Returns:
		*Apollo: A pointer to the Apollo object with the minting unit added.
*/
func (b *Apollo) MintAssetsWithRedeemer(mintUnit Unit, redeemer Redeemer.Redeemer) *Apollo {
	b.mint = append(b.mint, mintUnit)
	b.mintRedeemers[mintUnit.PolicyId] = redeemer
	b.isEstimateRequired = true
	return b
}

/**
buildTxBody constructs and returns the transaction body for the transaction.

Returns:
	TransactionBody.TransactionBody: The transaction body.
	error: An error if the build fails.
*/

func (b *Apollo) buildTxBody() (TransactionBody.TransactionBody, error) {
	inputs := make([]TransactionInput.TransactionInput, 0)
	for _, utxo := range b.preselectedUtxos {
		inputs = append(inputs, utxo.Input)
	}
	collaterals := make([]TransactionInput.TransactionInput, 0)
	for _, utxo := range b.collaterals {
		collaterals = append(collaterals, utxo.Input)
	}
	dataHash, err := b.scriptDataHash()
	if err != nil {
		return TransactionBody.TransactionBody{}, err
	}
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
	return txb, nil
}

/*
*

	buildFullFakeTx constructs and returns a full fake transaction for testing.

	Returns:
		*Transaction.Transaction: A pointer to the fake transaction.
		error: An error if the transaction construction fails.
*/
func (b *Apollo) buildFullFakeTx() (*Transaction.Transaction, error) {
	txBody, err := b.buildTxBody()
	if err != nil {
		return nil, err
	}
	if txBody.Fee == 0 {
		txBody.Fee = int64(b.Context.MaxTxFee())
	}
	witness := b.buildFakeWitnessSet()
	tx := Transaction.Transaction{
		TransactionBody:       txBody,
		TransactionWitnessSet: witness,
		Valid:                 true,
		AuxiliaryData:         b.auxiliaryData}
	bytes, _ := tx.Bytes()
	if len(bytes) > b.Context.GetProtocolParams().MaxTxSize {
		return nil, errors.New("transaction too large")
	}
	return &tx, nil
}

/*
*

	estimateFee estimates the transaction fee based on execution units and transaction size.

	Returns:
		int64: The estimated transaction fee.
*/
func (b *Apollo) estimateFee() int64 {
	pExU := Redeemer.ExecutionUnits{Mem: 0, Steps: 0}
	for _, redeemer := range b.redeemers {
		pExU.Sum(redeemer.ExUnits)
	}
	fftx, err := b.buildFullFakeTx()
	if err != nil {
		return 0
	}
	fakeTxBytes, _ := fftx.Bytes()
	estimatedFee := Utils.Fee(b.Context, len(fakeTxBytes), pExU.Steps, pExU.Mem)
	estimatedFee += b.FeePadding
	return estimatedFee

}

/*
*

	getAvailableUtxos returns the available unspent transaction outputs (UTXOs) for the transaction.

	Returns:
		[]UTxO.UTxO: A slice of available UTXOs.
*/
func (b *Apollo) getAvailableUtxos() []UTxO.UTxO {
	availableUtxos := make([]UTxO.UTxO, 0)
	for _, utxo := range b.utxos {
		if !slices.Contains(b.usedUtxos, utxo.GetKey()) {
			availableUtxos = append(availableUtxos, utxo)
		}
	}
	return availableUtxos
}

/*
*

	setRedeemerIndexes function sets indexes for redeemers in
	the transaction based on UTxO inputs.

	Returns:
		*Apollo: A pointer to the Apollo object with indexes and redeemers set.
*/
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

/*
*

	AttachDatum attaches a datum to the transaction.

	Params:
		datum *PlutusData.PlutusData: The datum to attach.

	Returns:
		*Apollo: A pointer to the Apollo object with the datum added.
*/
func (b *Apollo) AttachDatum(datum *PlutusData.PlutusData) *Apollo {
	b.datums = append(b.datums, *datum)
	return b
}

/**
setCollateral function sets collateral for the transaction.

Returns:
	*Apollo: A pointer to the Apollo object to support method chaining.
	error: An error if the setCollateral fails.
*/

func (b *Apollo) setCollateral() (*Apollo, error) {
	if len(b.collaterals) > 0 {
		collateral_amount := 5_000_000
		for _, utxo := range b.collaterals {
			if int(utxo.Output.GetValue().GetCoin()) >= collateral_amount+1_000_000 &&
				len(utxo.Output.GetValue().GetAssets()) <= 5 {
				b.totalCollateral = collateral_amount
				return_amount := utxo.Output.GetValue().GetCoin() - int64(collateral_amount)
				returnOutput := TransactionOutput.SimpleTransactionOutput(
					b.inputAddresses[0],
					Value.SimpleValue(return_amount, utxo.Output.GetValue().GetAssets()),
				)
				b.collateralReturn = &returnOutput
			}
		}
		return b, nil
	}
	witnesses := b.buildWitnessSet()
	if len(witnesses.PlutusV1Script) == 0 &&
		len(witnesses.PlutusV2Script) == 0 &&
		len(b.referenceInputs) == 0 {
		return b, nil
	}
	availableUtxos := b.getAvailableUtxos()
	collateral_amount := 5_000_000
	for _, utxo := range availableUtxos {
		if int(utxo.Output.GetValue().GetCoin()) >= collateral_amount &&
			len(utxo.Output.GetValue().GetAssets()) <= 5 {
			return_amount := utxo.Output.GetValue().GetCoin() - int64(collateral_amount)
			min_lovelace := Utils.MinLovelacePostAlonzo(
				TransactionOutput.SimpleTransactionOutput(
					b.inputAddresses[0],
					Value.SimpleValue(return_amount, utxo.Output.GetAmount().GetAssets()),
				),
				b.Context,
			)
			if min_lovelace > return_amount && return_amount != 0 {
				continue
			} else if return_amount == 0 && len(utxo.Output.GetAmount().GetAssets()) == 0 {
				b.collaterals = append(b.collaterals, utxo)
				b.totalCollateral = collateral_amount
				return b, nil
			} else {
				returnOutput := TransactionOutput.SimpleTransactionOutput(b.inputAddresses[0], Value.SimpleValue(return_amount, utxo.Output.GetValue().GetAssets()))
				b.collaterals = append(b.collaterals, utxo)
				b.collateralReturn = &returnOutput
				b.totalCollateral = collateral_amount
				return b, nil
			}
		}
	}
	for _, utxo := range availableUtxos {
		if int(utxo.Output.GetValue().GetCoin()) >= collateral_amount {
			return_amount := utxo.Output.GetValue().GetCoin() - int64(collateral_amount)
			min_lovelace := Utils.MinLovelacePostAlonzo(
				TransactionOutput.SimpleTransactionOutput(
					b.inputAddresses[0],
					Value.SimpleValue(return_amount, utxo.Output.GetAmount().GetAssets()),
				),
				b.Context,
			)
			if min_lovelace > return_amount && return_amount != 0 {
				continue
			} else if return_amount == 0 && len(utxo.Output.GetAmount().GetAssets()) == 0 {
				b.collaterals = append(b.collaterals, utxo)
				b.totalCollateral = collateral_amount
				return b, nil
			} else {
				returnOutput := TransactionOutput.SimpleTransactionOutput(b.inputAddresses[0], Value.SimpleValue(return_amount, utxo.Output.GetValue().GetAssets()))
				b.collaterals = append(b.collaterals, utxo)
				b.collateralReturn = &returnOutput
				b.totalCollateral = collateral_amount
				return b, nil
			}
		}
	}
	return b, errors.New("NoCollateral")
}

/*
*

	Clone creates a deep copy of the Apollo object.

	Returns:
		*Apollo: A pointer to the cloned Apollo object.
*/
func (b *Apollo) Clone() *Apollo {
	clone := *b
	return &clone
}

/*
*

	estimateExUnits estimates the execution units for redeemers and updates them.

	Returns:
		map[string]Redeemer.ExecutionUnits: A map of estimated execution units.
*/
func (b *Apollo) estimateExunits() map[string]Redeemer.ExecutionUnits {
	cloned_b := b.Clone()
	cloned_b.isEstimateRequired = false
	updated_b, _ := cloned_b.Complete()
	//updated_b = updated_b.fakeWitness()
	tx_cbor, _ := cbor.Marshal(updated_b.tx)
	return b.Context.EvaluateTx(tx_cbor)
}

/*
*

	updateExUnits updates the execution units in the transaction based on estimates.

	Returns:
		*Apollo: A pointer to the Apollo object to support method chaining.
*/
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
		for k, redeemer := range b.stakeRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.stakeRedeemers[k] = redeemer
			}
		}
		for k, redeemer := range b.mintRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
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
	return b
}

/*
*

	GetTx returns the transaction associated with the Apollo object.

	Returns:
		*Transacction.Transaction: A pointer to the transaction.
*/
func (b *Apollo) GetTx() *Transaction.Transaction {
	return b.tx
}

/*
*

	Complete assembles and finalizes the Apollo transaction, handling
	inputs, change, fees, collateral and witness data.

	Returns:
		*Apollo: A pointer to the Apollo object representing the completed transaction.
		error: An error if any issues are encountered during the process.
*/
func (b *Apollo) Complete() (*Apollo, error) {
	selectedUtxos := make([]UTxO.UTxO, 0)
	selectedAmount := Value.Value{}
	for _, utxo := range b.preselectedUtxos {
		selectedAmount = selectedAmount.Add(utxo.Output.GetValue())
	}
	burnedValue := b.GetBurns()
	mintedValue := b.getPositiveMints()
	selectedAmount = selectedAmount.Add(mintedValue)
	requestedAmount := Value.Value{}
	requestedAmount.Add(burnedValue)
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
						return nil, errors.New("missing required assets")
					}

				}
			}
		}
		for {
			if selectedAmount.Greater(
				requestedAmount.Add(
					Value.Value{Am: Amount.Amount{}, Coin: 1_000_000, HasAssets: false},
				),
			) {
				break
			}
			if len(available_utxos) == 0 {
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
	b, err := b.setCollateral()
	if err != nil {
		return nil, err
	}
	//UPDATE EXUNITS
	b = b.updateExUnits()
	//ADDCHANGEANDFEE
	b, err = b.addChangeAndFee()
	if err != nil {
		return nil, err
	}
	//FINALIZE TX
	body, err := b.buildTxBody()
	if err != nil {
		return nil, err
	}
	witnessSet := b.buildWitnessSet()
	b.tx = &Transaction.Transaction{
		TransactionBody:       body,
		TransactionWitnessSet: witnessSet,
		AuxiliaryData:         b.auxiliaryData,
		Valid:                 true,
	}
	return b, nil
}

/*
*

	Check if adding change to a transaction ouput would exceed
	the UTxO limit for the given address.

	Params:
		change: The change amount to add.
		address: The address for which change is being calculated.
		b: The ChainContext providing protocol parameters.

	Returns:
		bool: True if adding change would exceed the UTXO limit, false otherwise.
*/
func isOverUtxoLimit(change Value.Value, address Address.Address, b Base.ChainContext) bool {
	txOutput := TransactionOutput.SimpleTransactionOutput(
		address,
		Value.SimpleValue(0, change.GetAssets()),
	)
	encoded, _ := cbor.Marshal(txOutput)
	maxValSize, _ := strconv.Atoi(b.GetProtocolParams().MaxValSize)
	return len(encoded) > maxValSize

}

/*
*

	Split payments into multiple payments if adding change
	exceeds the UTxO limit.

	Params:
		c: The change amount.
		a: The address to which change is being sent.
		b: The ChainContext providing protocol parameters.

	Returns:


	[]*Payment: An array of payment objects, split if necessary to avoid exceeding the UTxO limit.
*/
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

func (b *Apollo) getPositiveMints() (mints Value.Value) {
	mints = Value.Value{}
	for _, mintUnit := range b.mint {
		if mintUnit.Quantity > 0 {
			usedUnit := Unit{
				PolicyId: mintUnit.PolicyId,
				Name:     mintUnit.Name,
				Quantity: mintUnit.Quantity,
			}
			mints = mints.Add(usedUnit.ToValue())
		}

	}
	return mints

}

/*
*
Add change and fees to the transaction.

Returns:

	*Apollo: A pointer to the Apollo object with change and fees added.
	error: An error if addChangeAndFee fails.
*/
func (b *Apollo) addChangeAndFee() (*Apollo, error) {
	burns := b.GetBurns()
	mints := b.getPositiveMints()
	providedAmount := Value.Value{}
	for _, utxo := range b.preselectedUtxos {
		providedAmount = providedAmount.Add(utxo.Output.GetValue())
	}
	providedAmount = providedAmount.Add(mints)
	requestedAmount := Value.Value{}
	for _, payment := range b.payments {
		requestedAmount = requestedAmount.Add(payment.ToValue())
	}
	requestedAmount = requestedAmount.Add(burns)
	b.Fee = b.estimateFee()
	requestedAmount.AddLovelace(b.Fee)
	change := providedAmount.Sub(requestedAmount)
	if change.GetCoin() < Utils.MinLovelacePostAlonzo(
		TransactionOutput.SimpleTransactionOutput(
			b.inputAddresses[0],
			Value.SimpleValue(0, change.GetAssets()),
		),
		b.Context,
	) {
		if len(b.getAvailableUtxos()) == 0 {
			return b, errors.New("No Remaining UTxOs")
		}
		sortedUtxos := SortUtxos(b.getAvailableUtxos())
		if len(sortedUtxos) == 0 {
			return b, errors.New("No Remaining UTxOs")
		}
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
	return b, nil
}

/*
*

	Collect a UTXO and its associated redeemer for inclusion in the transaction.

	Params:
		inputUtxo: The UTXO to collect.
		redeemer: The redeemer associated with the UTXO.

	Returns:
		*Apollo: A pointer to the Apollo object with the collected UTXO and redeemer.
*/
func (b *Apollo) CollectFrom(
	inputUtxo UTxO.UTxO,
	redeemer Redeemer.Redeemer,
) *Apollo {
	b.isEstimateRequired = true
	b.preselectedUtxos = append(b.preselectedUtxos, inputUtxo)
	b.usedUtxos = append(b.usedUtxos, inputUtxo.GetKey())
	b.redeemersToUTxO[hex.EncodeToString(inputUtxo.Input.TransactionId)+fmt.Sprint(inputUtxo.Input.Index)] = redeemer
	return b
}

/*
*

	Attach a Plutus V1 script to the Apollo transaction.

	Params:
		script: The Plutus V1 script to attach.

	Returns:
		*Apollo: A pointer to the Apollo objecy with the attached script.
*/
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

/*
*

	Attach a Plutus V2 script to the Apollo transaction.

	Params:
		script: The Plutus V2 script to attach.

	Returns:
		*Apollo: A pointer to the Apollo objecy with the attached script.
*/
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

/**
Set the wallet for the Apollo transaction using a mnemonic.

Params:
	menmonic: The menomic phrase used to generate the wallet.

Returns:
	*Apollo: A pointer to the Apollo object with the wallet set.
	error: an error if setWalletFromMnemonic fails.
*/

func (a *Apollo) SetWalletFromMnemonic(mnemonic string) (*Apollo, error) {
	paymentPath := "m/1852'/1815'/0'/0/0"
	stakingPath := "m/1852'/1815'/0'/2/0"
	hdWall, err := HDWallet.NewHDWalletFromMnemonic(mnemonic, "")
	if err != nil {
		return a, err
	}
	paymentKeyPath, err := hdWall.DerivePath(paymentPath)
	if err != nil {
		return a, err
	}
	verificationKey_bytes := paymentKeyPath.XPrivKey.PublicKey()
	signingKey_bytes := paymentKeyPath.XPrivKey.Bytes()
	stakingKeyPath, err := hdWall.DerivePath(stakingPath)
	if err != nil {
		return a, err
	}
	stakeVerificationKey_bytes := stakingKeyPath.XPrivKey.PublicKey()
	stakeSigningKey_bytes := stakingKeyPath.XPrivKey.Bytes()
	//stake := stakingKeyPath.RootXprivKey.Bytes()
	signingKey := Key.SigningKey{Payload: signingKey_bytes}
	verificationKey := Key.VerificationKey{Payload: verificationKey_bytes}
	stakeSigningKey := Key.SigningKey{Payload: stakeSigningKey_bytes}
	stakeVerificationKey := Key.VerificationKey{Payload: stakeVerificationKey_bytes}
	stakeVerKey := Key.VerificationKey{Payload: stakeVerificationKey_bytes}
	skh, _ := stakeVerKey.Hash()
	vkh, _ := verificationKey.Hash()

	addr := Address.Address{
		StakingPart: skh[:],
		PaymentPart: vkh[:],
		Network:     1,
		AddressType: Address.KEY_KEY,
		HeaderByte:  0b00000001,
		Hrp:         "addr",
	}
	wallet := apollotypes.GenericWallet{
		SigningKey:           signingKey,
		VerificationKey:      verificationKey,
		Address:              addr,
		StakeSigningKey:      stakeSigningKey,
		StakeVerificationKey: stakeVerificationKey,
	}
	a.wallet = &wallet
	return a, nil
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
	// There are two slightly different interpretations of ed25519,
	// depending on which thing you call the "private key".
	// cardano-cli and the golang library crypto/ed25519 take opposite
	// interpretations. NewKeyFromSeed performs the necessary conversion.
	signingKey := Key.SigningKey{Payload: ed25519.NewKeyFromSeed(signingKey_bytes)}
	verificationKey := Key.VerificationKey{Payload: verificationKey_bytes}
	vkh, _ := verificationKey.Hash()

	addr := Address.Address{}
	if network == constants.MAINNET {
		addr = Address.Address{
			StakingPart: nil,
			PaymentPart: vkh[:],
			Network:     1,
			AddressType: Address.KEY_NONE,
			HeaderByte:  0b01100001,
			Hrp:         "addr",
		}
	} else {
		addr = Address.Address{StakingPart: nil, PaymentPart: vkh[:], Network: 0, AddressType: Address.KEY_NONE, HeaderByte: 0b01100000, Hrp: "addr_test"}
	}
	wallet := apollotypes.GenericWallet{
		SigningKey:           signingKey,
		VerificationKey:      verificationKey,
		Address:              addr,
		StakeSigningKey:      Key.SigningKey{},
		StakeVerificationKey: Key.VerificationKey{},
	}
	a.wallet = &wallet
	return a
}

/*
*

	Set the wallet for the Apollo transaction using a Bech32 address.

	Params:
		address: The Bech32 address to use as the wallet.

	Returns:
		*Apollo: A pointer to the Apollo object with the wallet set.
*/
func (a *Apollo) SetWalletFromBech32(address string) *Apollo {
	addr, err := Address.DecodeAddress(address)
	if err != nil {
		return a
	}
	a.wallet = &apollotypes.ExternalWallet{Address: addr}
	return a
}

/*
*

	Set the wallet as the change address for the Apollo transaction.

	Returns:
		*Apollo: A pointer to the Apollo object with the wallet set as the change address.
*/
func (b *Apollo) SetWalletAsChangeAddress() *Apollo {
	if b.wallet == nil {
		panic("wallet not set")
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

/*
*

	Sign the Apollo transaction using the wallet's keys.

	Returns:
		*Apollo: A pointer to the Apollo object with the transaction signed.
*/
func (b *Apollo) Sign() *Apollo {
	usedUtxos := make([]UTxO.UTxO, 0)
	usedUtxos = append(usedUtxos, b.preselectedUtxos...)
	usedUtxos = append(usedUtxos, b.collaterals...)
	signatures := b.wallet.SignTx(*b.tx, usedUtxos)
	b.tx.TransactionWitnessSet = signatures
	return b
}

/*
*

	Sign the Apollo transaction with the given verification key and signing key.

	Parameters:
		vkey: The verification key to sign with.
		skey: The signing key to sign with.

	Returns:
		*Apollo: A pointer to the Apollo object with the transaction signed.
		error: An error if the signing fails.
*/
func (b *Apollo) SignWithSkey(vkey Key.VerificationKey, skey Key.SigningKey) (*Apollo, error) {
	witness_set := b.GetTx().TransactionWitnessSet
	txHash, err := b.GetTx().TransactionBody.Hash()
	if err != nil {
		return b, err
	}
	signature, err := skey.Sign(txHash)
	if err != nil {
		return b, err
	}
	witness_set.VkeyWitnesses = append(
		witness_set.VkeyWitnesses,
		VerificationKeyWitness.VerificationKeyWitness{Vkey: vkey, Signature: signature},
	)
	b.GetTx().TransactionWitnessSet = witness_set
	return b, nil
}

/*
*

	Submit function submits the constructed transaction to the blockchain
	network using the associated chain context.

	Returns:
		serialization.TransactionId: The ID of the submitted transaction.
		error: An error, if any, encountered during transaction submission.
*/
func (b *Apollo) Submit() (serialization.TransactionId, error) {
	return b.Context.SubmitTx(*b.tx)
}

/*
*

	LoadTxCbor loads a transaction from its CBOR representation and updates
	the apollo instances.

	Params:
		txCbor (string): The CBOR-encoded representation of the transaction.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the loaded transaction.
		error: An error, if any, encountered during loading.
*/
func (b *Apollo) LoadTxCbor(txCbor string) (*Apollo, error) {
	tx := Transaction.Transaction{}
	err := cbor.Unmarshal([]byte(txCbor), &tx)
	if err != nil {
		return b, err
	}
	b.tx = &tx
	return b, nil
}

/*
*

		UtxoFromRef retrieves a UTxO (Unspent Transaction Output) given its transaction hash and index.

		Params:
	   		txHash (string): The hexadecimal representation of the transaction hash.
	   		txIndex (int): The index of the UTxO within the transaction's outputs.

	 	Returns:
	   		*UTxO.UTxO: A pointer to the retrieved UTxO, or nil if not found.
*/
func (b *Apollo) UtxoFromRef(txHash string, txIndex int) *UTxO.UTxO {
	utxo := b.Context.GetUtxoFromRef(txHash, txIndex)
	if utxo == nil {
		return nil
	}
	return utxo

}

/*
*

	AddVerificationKeyWitness adds a verification key witness to the transaction.

	Params:
		vkw (VerificationKeyWitness.VerificationKeyWitness): The verification key witness to add.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the added verification key witness.
*/
func (b *Apollo) AddVerificationKeyWitness(
	vkw VerificationKeyWitness.VerificationKeyWitness,
) *Apollo {
	b.tx.TransactionWitnessSet.VkeyWitnesses = append(b.tx.TransactionWitnessSet.VkeyWitnesses, vkw)
	return b
}

/*
*

		SetChangeAddressBech32 sets the change address for the transaction using a Bech32-encoded address.

		Params:
	   		address (string): The Bech32-encoded address to set as the change address.

	 	Returns:
	   		*Apollo: A pointer to the modified Apollo instance with the change address set.
*/
func (b *Apollo) SetChangeAddressBech32(address string) *Apollo {
	addr, err := Address.DecodeAddress(address)
	if err != nil {
		return b
	}
	b.inputAddresses = append(b.inputAddresses, addr)
	return b
}

/*
*

		SetChangeAddress sets the change address for the transaction using an Address object.

	 	Params:
		   	address (Address.Address): The Address object to set as the change address.

		Returns:
		   	*Apollo: A pointer to the modified Apollo instance with the change address set.
*/
func (b *Apollo) SetChangeAddress(address Address.Address) *Apollo {
	b.inputAddresses = append(b.inputAddresses, address)
	return b
}

/*
*

	SetTtl function sets the time-to-live (TTL) for the transaction.

	Params:
		ttl (int64): The TTL value to set fro the transaction.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the TTl set.
*/
func (b *Apollo) SetTtl(ttl int64) *Apollo {
	b.Ttl = ttl
	return b
}

/*
*

	SetValidityStart function sets the validity start for the transaction.

	Params:
		invalidBefore (int64): The validity start value to set for the transaction.

	Returns:
	   	*Apollo: A pointer to the modified Apollo instance with the validity start set.
*/
func (b *Apollo) SetValidityStart(invalidBefore int64) *Apollo {
	b.ValidityStart = invalidBefore
	return b
}

/*
*

	SetShelleyMetadata function sets the Shelley Mary metadata for the transaction's
	auxiliary data.

	Params:
		metadata (Metadata.ShelleyMaryMetadata): The Shelley Mary metadat to set.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the Shelley Mary metadata set.
*/
func (b *Apollo) SetShelleyMetadata(metadata Metadata.ShelleyMaryMetadata) *Apollo {
	if b.auxiliaryData == nil {
		b.auxiliaryData = &Metadata.AuxiliaryData{}
		b.auxiliaryData.SetShelleyMetadata(metadata)
	} else {
		b.auxiliaryData.SetShelleyMetadata(metadata)
	}
	return b
}

/*
*

	GetUsedUTxOs returns the list of used UTxOs in the transaction.

	Returns:
	   []string: The list of used UTxOs as strings.
*/
func (b *Apollo) GetUsedUTxOs() []string {
	return b.usedUtxos
}

/*
*

	SetEstimationExUnitsRequired enables the estimation of execution units
	for the transaction.

	Returns:


	*Apollo: A pointer to the modified Apollo instance with execution units estimation enabled.
*/
func (b *Apollo) SetEstimationExUnitsRequired() *Apollo {
	b.isEstimateRequired = true
	return b
}

/*
*

	AddReferenceInput adds a reference input to the transaction.

	Params:
		txHash (string): The hexadecimal representation of the reference transaction hash.
		index (int): The index of the reference input within its transaction.

	Returns:
		*Apollo: A pointer to the modified Apollo instance with the added reference input.
*/
func (b *Apollo) AddReferenceInput(txHash string, index int) *Apollo {
	decodedHash, _ := hex.DecodeString(txHash)
	input := TransactionInput.TransactionInput{
		TransactionId: decodedHash,
		Index:         index,
	}
	b.referenceInputs = append(b.referenceInputs, input)
	return b
}

/*
*

		DisableExecutionUnitsEstimation disables the estimation of execution units for the transaction.

	 	Returns:


	*Apollo: A pointer to the modified Apollo instance with execution units estimation disabled.
*/
func (b *Apollo) DisableExecutionUnitsEstimation() *Apollo {
	b.isEstimateRequired = false
	return b
}

func (b *Apollo) AddWithdrawal(
	address Address.Address,
	amount int,
	redeemerData PlutusData.PlutusData,
) *Apollo {
	if b.withdrawals == nil {
		newWithdrawal := Withdrawal.New()
		b.withdrawals = &newWithdrawal
	}
	var stakeAddr [29]byte
	stakeAddr[0] = address.HeaderByte
	if len(address.StakingPart) != 28 {
		fmt.Printf(
			"AddWithdrawal: address has invalid or missing staking part: %v\n",
			address.StakingPart,
		)
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

func (b *Apollo) AddCollateral(utxo UTxO.UTxO) *Apollo {
	b.collaterals = append(b.collaterals, utxo)
	return b
}

func (b *Apollo) CompleteExact(fee int) (*Apollo, error) {
	//SET REDEEMER INDEXES
	b = b.setRedeemerIndexes()
	//SET COLLATERAL
	b, err := b.setCollateral()
	if err != nil {
		return nil, err
	}
	//UPDATE EXUNITS
	b = b.updateExUnitsExact(fee)
	//ADDCHANGEANDFEE
	b.Fee = int64(fee)
	//FINALIZE TX
	body, err := b.buildTxBody()
	if err != nil {
		return nil, err
	}
	witnessSet := b.buildWitnessSet()
	b.tx = &Transaction.Transaction{
		TransactionBody:       body,
		TransactionWitnessSet: witnessSet,
		AuxiliaryData:         b.auxiliaryData,
		Valid:                 true,
	}
	return b, nil
}

func (b *Apollo) estimateExunitsExact(fee int) map[string]Redeemer.ExecutionUnits {
	cloned_b := b.Clone()
	cloned_b.isEstimateRequired = false
	updated_b, _ := cloned_b.CompleteExact(fee)
	//updated_b = updated_b.fakeWitness()
	tx_cbor, _ := cbor.Marshal(updated_b.tx)
	return b.Context.EvaluateTx(tx_cbor)
}

func (b *Apollo) updateExUnitsExact(fee int) *Apollo {
	if b.isEstimateRequired {
		estimated_execution_units := b.estimateExunitsExact(fee)
		for k, redeemer := range b.redeemersToUTxO {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.redeemersToUTxO[k] = redeemer
			}
		}
		for k, redeemer := range b.stakeRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
			if _, ok := estimated_execution_units[key]; ok {
				redeemer.ExUnits = estimated_execution_units[key]
				b.stakeRedeemers[k] = redeemer
			}
		}
		for k, redeemer := range b.mintRedeemers {
			key := fmt.Sprintf("%s:%d", Redeemer.RdeemerTagNames[redeemer.Tag], redeemer.Index)
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
	return b
}
