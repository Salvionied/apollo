package apollo

import (
	"encoding/hex"

	"github.com/salvionied/apollo/serialization/Address"
	"github.com/salvionied/apollo/serialization/PlutusData"
	"github.com/salvionied/apollo/serialization/Transaction"
	"github.com/salvionied/apollo/serialization/TransactionOutput"
	"github.com/salvionied/apollo/serialization/UTxO"
	"github.com/salvionied/apollo/serialization/Value"
	"github.com/salvionied/apollo/txBuilding/TxBuilder"
)

type builder struct {
	Apollo    *Apollo
	txBuilder TxBuilder.TransactionBuilder
}

func (b *builder) Init() *builder {
	b.txBuilder = TxBuilder.InitBuilder(b.Apollo.backend)
	return b
}

func (b *builder) SetWalletAsInput() *builder {
	utxos := b.Apollo.backend.Utxos(*b.Apollo.Wallet.GetAddress())
	b.txBuilder.AddLoadedUTxOs(utxos)
	return b
}

func (b *builder) SetInputUtxosPool(utxos []UTxO.UTxO) *builder {
	b.txBuilder.AddLoadedUTxOs(utxos)
	return b
}

func (b *builder) AddInput(utxo UTxO.UTxO) *builder {
	b.txBuilder.AddInput(utxo)
	return b
}

func (b *builder) PayToAddressLovelaceBech32(address string, amount int) *builder {
	decoded_address, _ := Address.DecodeAddress(address)
	b.txBuilder.AddOutput(TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(int64(amount))), nil, false)
	return b
}

func (b *builder) PayToAddressLovelace(address Address.Address, amount int) *builder {
	b.txBuilder.AddOutput(TransactionOutput.SimpleTransactionOutput(address, Value.PureLovelaceValue(int64(amount))), nil, false)
	return b
}

func (b *builder) PayToContractLovelace(contractAddress Address.Address, pd PlutusData.PlutusData, lovelace int64) *builder {
	txO := TransactionOutput.SimpleTransactionOutput(contractAddress, Value.PureLovelaceValue(lovelace))
	txO.SetDatum(pd)
	txO.PreAlonzo.DatumHash = PlutusData.PlutusDataHash(pd)
	b.txBuilder.Outputs = append(b.txBuilder.Outputs, txO)
	if b.txBuilder.Datums == nil {
		b.txBuilder.Datums = make(map[string]PlutusData.PlutusData)
	}
	b.txBuilder.Datums[hex.EncodeToString(PlutusData.PlutusDataHash(pd).Payload)] = pd
	return b
}

func (b *builder) Complete() (*ApolloTransaction, error) {
	TransactionBody, err := b.txBuilder.Build(b.Apollo.Wallet.GetAddress(), false, b.Apollo.Wallet.GetAddress())
	if err != nil {
		return nil, err
	}
	fullTx := ApolloTransaction{
		Apollo: b.Apollo,
		Tx: Transaction.Transaction{
			TransactionBody:       TransactionBody,
			TransactionWitnessSet: b.txBuilder.BuildWitnessSet(),
			Valid:                 true,
			AuxiliaryData:         b.txBuilder.AuxiliaryData}}
	return &fullTx, nil
}
