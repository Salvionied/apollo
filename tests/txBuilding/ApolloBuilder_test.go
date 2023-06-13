package txBuilding_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/Salvionied/cbor/v2"
	"github.com/salvionied/apollo"
	"github.com/salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
)

type Network int

const (
	MAINNET Network = iota
	TESTNET
	PREVIEW
	PREPROD
)

const BLOCKFROST_BASE_URL_MAINNET = "https://cardano-mainnet.blockfrost.io/api"
const BLOCKFROST_BASE_URL_TESTNET = "https://cardano-testnet.blockfrost.io/api"
const BLOCKFROST_BASE_URL_PREVIEW = "https://cardano-preview.blockfrost.io/api"
const BLOCKFROST_BASE_URL_PREPROD = "https://cardano-preprod.blockfrost.io/api"

// func TestScriptAddress(t *testing.T) {
// 	SC_CBOR := "5901ec01000032323232323232323232322223232533300a3232533300c002100114a066646002002444a66602400429404c8c94ccc040cdc78010018a5113330050050010033015003375c60260046eb0cc01cc024cc01cc024011200048040dd71980398048012400066e3cdd7198031804001240009110d48656c6c6f2c20576f726c642100149858c8014c94ccc028cdc3a400000226464a66602060240042930a99806a49334c6973742f5475706c652f436f6e73747220636f6e7461696e73206d6f7265206974656d73207468616e2065787065637465640016375c6020002601000a2a660169212b436f6e73747220696e64657820646964206e6f74206d6174636820616e7920747970652076617269616e7400163008004320033253330093370e900000089919299980798088010a4c2a66018921334c6973742f5475706c652f436f6e73747220636f6e7461696e73206d6f7265206974656d73207468616e2065787065637465640016375c601e002600e0062a660149212b436f6e73747220696e64657820646964206e6f74206d6174636820616e7920747970652076617269616e740016300700233001001480008888cccc01ccdc38008018061199980280299b8000448008c0380040080088c018dd5000918021baa0015734ae7155ceaab9e5573eae855d11"
// 	//resultingAddr := "addr1w8elsgw3y2cyzfzdup6tj42v0k7vvte57cjzvdzvp595epsljnl47"
// 	decoded_string, err := hex.DecodeString(SC_CBOR)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	p2Script := PlutusData.PlutusV2Script(decoded_string)
// 	hashOfScript := p2Script.Hash()
// 	if hex.EncodeToString(hashOfScript.Bytes()) != "f3f821d122b041244de074b9554c7dbcc62f34f62426344c0d0b4c86" {
// 		t.Error("Hash of script is not correct", hex.EncodeToString(hashOfScript.Bytes()), " != ", "f3f821d122b041244de074b9554c7dbcc62f34f62426344c0d0b4c86")
// 	}

// }

// func TestUnlock1Ada(t *testing.T) {
// 	txHash := "d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056"
// 	txIdx := 0
// 	cc := BlockFrostChainContext.NewBlockfrostChainContext("ApiKeyHere", int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
// 	SEED := "seed phrase here"
// 	apollob := apollo.New(&cc).SetWalletFromMnemonic(SEED).SetWalletAsInput()
// 	_, filename, _, _ := runtime.Caller(0)
// 	f, err := os.Open(strings.Replace(filename, "tests/txBuilding/ApolloBuilder_test.go", "samples/plutus.json", 1))
// 	if err != nil {
// 		fmt.Println(err)
// 		panic("HERE OPENING FILE")
// 	}
// 	defer f.Close()
// 	aikenPlutusJSON := apollotypes.AikenPlutusJSON{}
// 	plutus_bytes, err := ioutil.ReadAll(f)
// 	err = json.Unmarshal(plutus_bytes, &aikenPlutusJSON)
// 	if err != nil {
// 		panic("HERE UNMARSHALING")
// 	}
// 	script, err := aikenPlutusJSON.GetScript("hello_world.hello_world")
// 	datum := PlutusData.PlutusData{
// 		TagNr:          121,
// 		PlutusDataType: PlutusData.PlutusArray,
// 		Value: PlutusData.PlutusIndefArray{
// 			PlutusData.PlutusData{
// 				TagNr:          0,
// 				PlutusDataType: PlutusData.PlutusBytes,
// 				Value:          apollob.GetWallet().PkeyHash(),
// 			},
// 		},
// 	}
// 	redeemer_content := PlutusData.Datum{
// 		TagNr:          121,
// 		PlutusDataType: PlutusData.PlutusArray,
// 		Value: PlutusData.PlutusIndefArray{{
// 			TagNr:          0,
// 			PlutusDataType: PlutusData.PlutusBytes,
// 			Value:          []byte("Hello, World!")},
// 		},
// 	}
// 	redeemer := Redeemer.Redeemer{
// 		Tag:  Redeemer.SPEND,
// 		Data: redeemer_content,
// 	}
// 	inputUtxo := apollob.UtxoFromRef(txHash, txIdx)
// 	inputUtxo.Output.SetDatum(datum)

// 	//contractAddress := script.ToAddress(nil)
// 	apollob, err = apollob.
// 		AddRequiredSignerFromAddress(*apollob.GetWallet().GetAddress(), true, false).
// 		CollectFrom(*inputUtxo, redeemer).
// 		AttachV2Script(*script).
// 		AttachDatum(&datum).
// 		PayToAddress(*apollob.GetWallet().GetAddress(), int(1_000_000), nil).
// 		Complete()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	apollob = apollob.Sign()
// 	//tx_id, err := apollob.Submit()
// 	// if err != nil {
// 	// 	t.Error(err)
// 	// }
// 	// fmt.Println(hex.EncodeToString(tx_id.Payload))

// }

// func TestLock1Ada(t *testing.T) {
// 	cc := BlockFrostChainContext.NewBlockfrostChainContext("project_api_key", int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
// 	SEED := "seed phrase here"
// 	apollob := apollo.New(&cc).SetWalletFromMnemonic(SEED).SetWalletAsInput()
// 	_, filename, _, _ := runtime.Caller(0)
// 	fmt.Println(filename)
// 	f, err := os.Open(strings.Replace(filename, "tests/txBuilding/ApolloBuilder_test.go", "samples/plutus.json", 1))
// 	if err != nil {
// 		fmt.Println(err)
// 		panic("HERE OPENING FILE")
// 	}
// 	defer f.Close()
// 	aikenPlutusJSON := apollotypes.AikenPlutusJSON{}
// 	plutus_bytes, err := ioutil.ReadAll(f)
// 	err = json.Unmarshal(plutus_bytes, &aikenPlutusJSON)
// 	if err != nil {
// 		panic("HERE UNMARSHALING")
// 	}
// 	script, err := aikenPlutusJSON.GetScript("hello_world.hello_world")
// 	datum := PlutusData.PlutusData{
// 		TagNr:          121,
// 		PlutusDataType: PlutusData.PlutusArray,
// 		Value: PlutusData.PlutusIndefArray{
// 			PlutusData.PlutusData{
// 				TagNr:          0,
// 				PlutusDataType: PlutusData.PlutusBytes,
// 				Value:          apollob.GetWallet().PkeyHash(),
// 			},
// 		},
// 	}
// 	// redeemer_content := PlutusData.Datum{
// 	// 	TagNr:          0,
// 	// 	PlutusDataType: PlutusData.PlutusBytes,
// 	// 	Value:          []byte("Hello, World!"),
// 	// }
// 	// redeemer := Redeemer.Redeemer{
// 	// 	Tag:  Redeemer.SPEND,
// 	// 	Data: redeemer_content,
// 	// }
// 	contractAddress := script.ToAddress(nil)
// 	apollob, err = apollob.
// 		PayToContract(contractAddress, &datum, 1_000_000, nil).
// 		Complete()

// 	if err != nil {
// 		t.Error(err)
// 	}
// 	apollob = apollob.Sign()
// 	cborred, _ := cbor.Marshal(apollob.GetTx())
// 	fmt.Println(hex.EncodeToString(cborred))
// 	tx_id, err := apollob.Submit()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(tx_id)
// 	fmt.Println(hex.EncodeToString(tx_id.Payload))
// 	t.Error("HERE")

// }

func TestSend1Ada(t *testing.T) {
	bfc := BlockFrostChainContext.NewBlockfrostChainContext("blockfrost_api_key", int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
	cc := apollo.NewEmptyBackend()
	SEED := "seed phrase here"
	apollob := apollo.New(&cc)
	apollob = apollob.
		SetWalletFromMnemonic(SEED).
		SetWalletAsChangeAddress()
	utxos := bfc.Utxos(*apollob.GetWallet().GetAddress())
	apollob, err := apollob.
		AddLoadedUTxOs(utxos).
		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 1_000_000, nil).
		Complete()
	if err != nil {
		fmt.Println(err)
		t.Error(err)
	}
	apollob = apollob.Sign()
	tx := apollob.GetTx()
	cborred, err := cbor.Marshal(tx)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(hex.EncodeToString(cborred))
	tx_id, _ := bfc.SubmitTx(*tx)

	fmt.Println(hex.EncodeToString(tx_id.Payload))
	t.Error("HERE")

}

func TestSimpleBuild(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext("blockfrost_api_key", int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
		AddLoadedUTxOs(initUtxosDifferentiated()).
		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 10_000_000, nil).
		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
			5_000_000,
			[]apollo.Unit{{"00000000000000000000000000000000000000000000000000000000", "token0", 1}, {"00000000000000000000000000000000000000000000000000000000", "token2", 3}},
		).Complete()
	if err != nil {
		t.Error(err)
	}
	cborred, _ := cbor.Marshal(apollob.GetTx())
	fmt.Println(apollob.GetTx().TransactionBody.CollateralReturn, apollob.GetTx().TransactionBody.Withdrawals)
	fmt.Println(hex.EncodeToString(cborred))
}
