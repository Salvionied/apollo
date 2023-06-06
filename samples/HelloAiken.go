package samples

import (
	"encoding/hex"
	"fmt"

	"github.com/Salvionied/cbor/v2"
	"github.com/salvionied/apollo"
	"github.com/salvionied/apollo/serialization/PlutusData"
)

func Lock(
	lovelace int64,
	validatorScript *PlutusData.PlutusV2Script,
	owner *PlutusData.PlutusData,
	builder *apollo.Apollo,
) {
	contractAddress := validatorScript.ToAddress(nil)
	tx, err := builder.NewTx().Init().SetWalletAsInput().PayToContractLovelace(contractAddress, *owner, lovelace).Complete()
	if err != nil {
		panic(err)
	}
	encoded, err := cbor.Marshal(tx)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(encoded))

	tx = tx.Sign()
	fmt.Println(tx)
	// if err != nil {
	// 	panic(err)
	// }
	// tx_hash, err := tx.Submit()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(tx_hash)

}

func HelloAiken() {
	backend := apollo.NewBlockfrostBackend("project_id", apollo.MAINNET)
	SEED := "area decide toilet salad avocado horn early oak glove black all orbit clean pigeon grant grace subject enforce train year front rule noise kick"
	builder := apollo.New(backend, apollo.MAINNET).SetWalletFromMnemonic(SEED)
	AikenJenJSON, err := apollo.ReadAikenJson("apollo/samples/plutus.json")
	if err != nil {
		panic(err)
	}
	script, err := AikenJenJSON.GetScript("hello_world.hello_world")
	datum := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusArray,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				TagNr:          0,
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          builder.Wallet.PkeyHash(),
			},
		},
	}
	Lock(1_000_000, script, &datum, builder)
}
