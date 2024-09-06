package apollo_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	testutils "github.com/Salvionied/apollo/testUtils"
	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
	"github.com/Salvionied/cbor/v2"
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

func TestUnmarshal(t *testing.T) {
	tx := Transaction.Transaction{}
	cborHex := "84a6008b8258205dc014cbcfd8ce86a4e2acb0c6a447066dfa65706a04820e36e2ec6e2264fbd7068258204c887654fa91f24c8855e2762784a30f079e92e511ae92cf6e755ef1e2cf9b8e068258203af2bb10a835f805419429c31658fc7333a43c9fcedf724b747854f989cea8fa068258205dc014cbcfd8ce86a4e2acb0c6a447066dfa65706a04820e36e2ec6e2264fbd704825820328d53f17cec0c5fe8f7726c2c9be71570918625cdb002b22bde4dcd95844ef0068258203af2bb10a835f805419429c31658fc7333a43c9fcedf724b747854f989cea8fa0482582002414578f8ea5208364f9ee1e28496495e3fdc2a8befc6cf6e2256c70a7d0e5a008258209281c9b455b9ec279c3160ab8efd22aecfc75f8f294bf9942dbd096c405ddf49008258205dc014cbcfd8ce86a4e2acb0c6a447066dfa65706a04820e36e2ec6e2264fbd705825820328d53f17cec0c5fe8f7726c2c9be71570918625cdb002b22bde4dcd95844ef0048258200ed3bbcfaa51dd1db2871195d871ab73c59294c7275e1f46d9c9fa799b66db1801018382583911a65ca58a4e9c755fa830173d2a5caed458ac0c73f97db7faae2e7e3b52563c5410bff6a0d43ccebb7c37e1f69f5eb260552521adff33b9c21a0089544082583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a000fd9768258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce821a0013a461a1581c5d16cc1a177b5d9ba9cfa9793b07e60f1fb70fea1f8aef064415d114a1434941471b0000000ba43b740002000319012c075820b64602eebf602e8bbce198e2a1d6bbb2a109ae87fa5316135d217110d6d946490b5820c1a02dc05beee9b267cd22f449ac15f3d70bda1b47a6b4ad5c855774171705eba1049fd8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd87a80d8799fd8799f581c29d222ce763455e3d7a09a665ce554f00ac89d2e99a1a83d267170c6434d494eff1b00003fd483e52478ff1a001e84801a001e8480fffff5a11902a2a1636d736781781c4d696e737761703a205377617020457861637420496e204f72646572"
	cborBytes, err := hex.DecodeString(cborHex)
	if err != nil {
		t.Error(err)
	}
	err = cbor.Unmarshal(cborBytes, &tx)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(tx.AuxiliaryData)
	remarshaled, _ := cbor.Marshal(tx)
	fmt.Println(hex.EncodeToString(remarshaled))
	if tx.AuxiliaryData == nil {
		t.Error("AuxiliaryData is nil")
	}

}

func TestEnsureTxIsBalanced(t *testing.T) {
	cc := FixedChainContext.InitFixedChainContext()
	apollob := apollo.New(&cc)
	userAddress := "addr1qymaeeefs9ff08cdplm3lvkscavm9x9vd7nmc44e9rlur08k3pj2xw9w3mvp7cg3fkzhed4zzhywdpd2t3pmc8u8nn8qm5ur5w"
	SampleUtxos := `["8282582023fca3d654c1194e776949626b3794db80a81d66cd3490b04e55268baaf7d392078258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1b00000003c2f30419"]`
	jsonutxos := make([]string, 0)
	_ = json.Unmarshal([]byte(SampleUtxos), &jsonutxos)
	utxos := make([]UTxO.UTxO, 0)
	for _, utxo := range jsonutxos {
		var loadedUtxo UTxO.UTxO
		decodedUtxo, _ := hex.DecodeString(utxo)
		err := cbor.Unmarshal(decodedUtxo, &loadedUtxo)
		if err != nil {
			t.Error(err)
		}
		utxos = append(utxos, loadedUtxo)
	}
	fmt.Println(utxos)
	apollob = apollob.AddInputAddressFromBech32(userAddress).AddLoadedUTxOs(utxos...).
		PayToAddressBech32("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh", int(2_000_000)).
		SetTtl(0 + 300)
	apollob, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	fmt.Println(hex.EncodeToString(txBytes))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf(
				"%s:%d",
				hex.EncodeToString(input.TransactionId),
				input.Index,
			) {
				//fmt.Println("INPUT", idx, utxo)
				inputVal = inputVal.Add(utxo.Output.GetAmount())
			}
		}
	}
	outputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, output := range apollob.GetTx().TransactionBody.Outputs {
		outputVal = outputVal.Add(output.GetAmount())
	}
	outputVal.AddLovelace(apollob.Fee)
	fmt.Println("INPUT VAL", inputVal)
	fmt.Println("OUTPUT VAL", outputVal)
	fmt.Println("FEE", apollob.Fee)
	if !inputVal.Equal(outputVal) {
		t.Error("Tx is not balanced")
	}
}

func TestComplexTxBuild(t *testing.T) {
	cc := FixedChainContext.InitFixedChainContext()
	userAddress := "addr1qymaeeefs9ff08cdplm3lvkscavm9x9vd7nmc44e9rlur08k3pj2xw9w3mvp7cg3fkzhed4zzhywdpd2t3pmc8u8nn8qm5ur5w"
	apollob := apollo.New(&cc)
	SampleUtxos := `[
		"82825820e996196a51c5206aac8114e9e0371968e43b67d8ff4cdf0ab43ff248aa246f1f018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce821a003cd53eab581c10a49b996e2402269af553a8a96fb8eb90d79e9eca79e2b4223057b6a1444745524f1a001e8480581c1ddcb9c9de95361565392c5bdff64767492d61a96166cb16094e54bea1434f50541a03458925581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa144534e454b1928b0581c29d222ce763455e3d7a09a665ce554f00ac89d2e99a1a83d267170c6a1434d494e1a0cb30355581c5d16cc1a177b5d9ba9cfa9793b07e60f1fb70fea1f8aef064415d114a1434941471b00000022eddeef81581c8a1cfae21368b8bebbbed9800fec304e95cce39a2a57dc35e2e3ebaaa1444d494c4b05581c8b4e239aef4d1d1bc5dd628ff3ce34d392d632e5cda83e42d6fcb1cca14b586572636865723234393301581cd480f68af028d6324ad77df489176e7f5e5d793e09a6b133392ff2f6aa524e7563617374496e63657074696f6e31343101524e7563617374496e63657074696f6e32303601524e7563617374496e63657074696f6e33323101524e7563617374496e63657074696f6e33383501524e7563617374496e63657074696f6e34303001524e7563617374496e63657074696f6e36333701524e7563617374496e63657074696f6e36373001524e7563617374496e63657074696f6e37383701524e7563617374496e63657074696f6e38333301524e7563617374496e63657074696f6e38373001581ce3ff4ab89245ede61b3e2beab0443dbcc7ea8ca2c017478e4e8990e2a549746170707930333831014974617070793034313901497461707079313430390149746170707931343437014974617070793135353001581cf0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9aa24a626c7565646573657274014a6d6f6e74626c616e636f01581cf43a62fdc3965df486de8a0d32fe800963589c41b38946602a0dc535a144414749581a4ec73bbf",
		"8282582023fca3d654c1194e776949626b3794db80a81d66cd3490b04e55268baaf7d392048258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a003385dd",
		"8282582023fca3d654c1194e776949626b3794db80a81d66cd3490b04e55268baaf7d392078258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1b00000003c2f30419",
		"8282582063ac086da56aaeb699d6296cffc7d3bae4ea9cee1021fd9035e3144d28c195ef018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a001aae3f",
		"828258206f173d15f91109f4afbdb72a302f611cb4edd3f34db8f9fd7525310b0e06fc5c048258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a000faa63",
		"82825820462161505962663642d522d95220302a5eaaf589cd005357b5c4f6570b0f4f91018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1b0000000682bc10c6"
	]`
	plutusDataCbor := "d8799fd8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffff581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bc1b0000018a0308bc6fd8799fd8799f4040ffd8799f581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f44534e454bffffffd8799fd87a801a0083deb5ffff"
	decodedPlutusData, _ := hex.DecodeString(plutusDataCbor)
	pd := PlutusData.PlutusData{}
	err := cbor.Unmarshal(decodedPlutusData, &pd)
	if err != nil {
		t.Error(err)
	}
	jsonutxos := make([]string, 0)
	_ = json.Unmarshal([]byte(SampleUtxos), &jsonutxos)
	utxos := make([]UTxO.UTxO, 0)
	for _, utxo := range jsonutxos {
		var loadedUtxo UTxO.UTxO
		decodedUtxo, _ := hex.DecodeString(utxo)
		err := cbor.Unmarshal(decodedUtxo, &loadedUtxo)
		if err != nil {
			t.Error(err)
		}
		utxos = append(utxos, loadedUtxo)
	}
	decodedAddress, _ := Address.DecodeAddress(
		"addr1wxr2a8htmzuhj39y2gq7ftkpxv98y2g67tg8zezthgq4jkg0a4ul4",
	)
	apollob = apollob.AddInputAddressFromBech32(userAddress).AddLoadedUTxOs(utxos...).
		PayToContract(decodedAddress, &pd,
			4000000,
			false,
			apollo.NewUnit("279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f", "SNEK", 10416),
		).
		PayToAddressBech32("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh", int(2_000_000)).
		SetTtl(0 + 300).
		SetValidityStart(0)
	apollob, err = apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	fmt.Println(hex.EncodeToString(txBytes))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf(
				"%s:%d",
				hex.EncodeToString(input.TransactionId),
				input.Index,
			) {
				//fmt.Println("INPUT", idx, utxo)
				inputVal = inputVal.Add(utxo.Output.GetAmount())
			}
		}
	}
	outputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, output := range apollob.GetTx().TransactionBody.Outputs {
		outputVal = outputVal.Add(output.GetAmount())
	}
	outputVal.AddLovelace(apollob.Fee)
	fmt.Println("INPUT VAL", inputVal)
	fmt.Println("OUTPUT VAL", outputVal)
	if !inputVal.Equal(outputVal) {
		t.Error("Tx is not balanced")
	}
	// fmt.Println(apollob.GetTx().TransactionBody.Outputs)
	//t.Error("STOP")
	if err != nil {
		t.Error(err)
	}

}

func TestFakeBurnBalancing(t *testing.T) {
	cc := FixedChainContext.InitFixedChainContext()
	userAddress := "addr1qymaeeefs9ff08cdplm3lvkscavm9x9vd7nmc44e9rlur08k3pj2xw9w3mvp7cg3fkzhed4zzhywdpd2t3pmc8u8nn8qm5ur5w"
	apollob := apollo.New(&cc)
	SampleUtxos := `[
		"82825820e996196a51c5206aac8114e9e0371968e43b67d8ff4cdf0ab43ff248aa246f1f018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce821a003cd53eab581c10a49b996e2402269af553a8a96fb8eb90d79e9eca79e2b4223057b6a1444745524f1a001e8480581c1ddcb9c9de95361565392c5bdff64767492d61a96166cb16094e54bea1434f50541a03458925581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa144534e454b1928b0581c29d222ce763455e3d7a09a665ce554f00ac89d2e99a1a83d267170c6a1434d494e1a0cb30355581c5d16cc1a177b5d9ba9cfa9793b07e60f1fb70fea1f8aef064415d114a1434941471b00000022eddeef81581c8a1cfae21368b8bebbbed9800fec304e95cce39a2a57dc35e2e3ebaaa1444d494c4b05581c8b4e239aef4d1d1bc5dd628ff3ce34d392d632e5cda83e42d6fcb1cca14b586572636865723234393301581cd480f68af028d6324ad77df489176e7f5e5d793e09a6b133392ff2f6aa524e7563617374496e63657074696f6e31343101524e7563617374496e63657074696f6e32303601524e7563617374496e63657074696f6e33323101524e7563617374496e63657074696f6e33383501524e7563617374496e63657074696f6e34303001524e7563617374496e63657074696f6e36333701524e7563617374496e63657074696f6e36373001524e7563617374496e63657074696f6e37383701524e7563617374496e63657074696f6e38333301524e7563617374496e63657074696f6e38373001581ce3ff4ab89245ede61b3e2beab0443dbcc7ea8ca2c017478e4e8990e2a549746170707930333831014974617070793034313901497461707079313430390149746170707931343437014974617070793135353001581cf0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9aa24a626c7565646573657274014a6d6f6e74626c616e636f01581cf43a62fdc3965df486de8a0d32fe800963589c41b38946602a0dc535a144414749581a4ec73bbf",
		"8282582023fca3d654c1194e776949626b3794db80a81d66cd3490b04e55268baaf7d392048258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a003385dd",
		"8282582023fca3d654c1194e776949626b3794db80a81d66cd3490b04e55268baaf7d392078258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1b00000003c2f30419",
		"8282582063ac086da56aaeb699d6296cffc7d3bae4ea9cee1021fd9035e3144d28c195ef018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a001aae3f",
		"828258206f173d15f91109f4afbdb72a302f611cb4edd3f34db8f9fd7525310b0e06fc5c048258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a000faa63",
		"82825820462161505962663642d522d95220302a5eaaf589cd005357b5c4f6570b0f4f91018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1b0000000682bc10c6"
	]`
	plutusDataCbor := "d8799fd8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffff581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bc1b0000018a0308bc6fd8799fd8799f4040ffd8799f581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f44534e454bffffffd8799fd87a801a0083deb5ffff"
	decodedPlutusData, _ := hex.DecodeString(plutusDataCbor)
	pd := PlutusData.PlutusData{}
	err := cbor.Unmarshal(decodedPlutusData, &pd)
	if err != nil {
		t.Error(err)
	}
	jsonutxos := make([]string, 0)
	_ = json.Unmarshal([]byte(SampleUtxos), &jsonutxos)
	utxos := make([]UTxO.UTxO, 0)
	for _, utxo := range jsonutxos {
		var loadedUtxo UTxO.UTxO
		decodedUtxo, _ := hex.DecodeString(utxo)
		err := cbor.Unmarshal(decodedUtxo, &loadedUtxo)
		if err != nil {
			t.Error(err)
		}
		utxos = append(utxos, loadedUtxo)
	}
	decodedAddress, _ := Address.DecodeAddress(
		"addr1wxr2a8htmzuhj39y2gq7ftkpxv98y2g67tg8zezthgq4jkg0a4ul4",
	)
	apollob = apollob.AddInputAddressFromBech32(userAddress).AddLoadedUTxOs(utxos...).
		PayToContract(decodedAddress, &pd,
			4000000,
			false,
			apollo.NewUnit("279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f", "SNEK", 10416),
		).
		PayToAddressBech32("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh", int(2_000_000)).
		SetTtl(0 + 300).
		SetValidityStart(0).MintAssets(
		apollo.NewUnit("f0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9a", "bluedesert", -1),
	)
	apollob, err = apollob.Complete()
	if err != nil {
		fmt.Println("HERE")
		t.Error(err)
	}
	//t.Error("STOP")
	txBytes, err := apollob.GetTx().Bytes()
	fmt.Println(hex.EncodeToString(txBytes))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf(
				"%s:%d",
				hex.EncodeToString(input.TransactionId),
				input.Index,
			) {
				//fmt.Println("INPUT", idx, utxo)
				inputVal = inputVal.Add(utxo.Output.GetAmount())
			}
		}
	}
	outputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, output := range apollob.GetTx().TransactionBody.Outputs {
		outputVal = outputVal.Add(output.GetAmount())
	}
	outputVal.AddLovelace(apollob.Fee)
	outputVal = outputVal.Add(apollob.GetBurns())
	fmt.Println("INPUT VAL", inputVal)
	fmt.Println("OUTPUT VAL", outputVal)
	if !inputVal.Equal(outputVal) {
		t.Error("Tx is not balanced")
	}
	// fmt.Println(apollob.GetTx().TransactionBody.Outputs)
	//t.Error("STOP")
	if err != nil {
		t.Error(err)
	}

}

func TestScriptAddress(t *testing.T) {
	SC_CBOR := "5901ec01000032323232323232323232322223232533300a3232533300c002100114a066646002002444a66602400429404c8c94ccc040cdc78010018a5113330050050010033015003375c60260046eb0cc01cc024cc01cc024011200048040dd71980398048012400066e3cdd7198031804001240009110d48656c6c6f2c20576f726c642100149858c8014c94ccc028cdc3a400000226464a66602060240042930a99806a49334c6973742f5475706c652f436f6e73747220636f6e7461696e73206d6f7265206974656d73207468616e2065787065637465640016375c6020002601000a2a660169212b436f6e73747220696e64657820646964206e6f74206d6174636820616e7920747970652076617269616e7400163008004320033253330093370e900000089919299980798088010a4c2a66018921334c6973742f5475706c652f436f6e73747220636f6e7461696e73206d6f7265206974656d73207468616e2065787065637465640016375c601e002600e0062a660149212b436f6e73747220696e64657820646964206e6f74206d6174636820616e7920747970652076617269616e740016300700233001001480008888cccc01ccdc38008018061199980280299b8000448008c0380040080088c018dd5000918021baa0015734ae7155ceaab9e5573eae855d11"
	//resultingAddr := "addr1w8elsgw3y2cyzfzdup6tj42v0k7vvte57cjzvdzvp595epsljnl47"
	decoded_string, err := hex.DecodeString(SC_CBOR)
	if err != nil {
		t.Error(err)
	}
	p2Script := PlutusData.PlutusV2Script(decoded_string)
	hashOfScript, _ := p2Script.Hash()
	if hex.EncodeToString(
		hashOfScript.Bytes(),
	) != "f3f821d122b041244de074b9554c7dbcc62f34f62426344c0d0b4c86" {
		t.Error(
			"Hash of script is not correct",
			hex.EncodeToString(hashOfScript.Bytes()),
			" != ",
			"f3f821d122b041244de074b9554c7dbcc62f34f62426344c0d0b4c86",
		)
	}

}

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

// func TestSend1Ada(t *testing.T) {
// 	bfc := BlockFrostChainContext.NewBlockfrostChainContext("blockfrost_api_key", int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
// 	cc := apollo.NewEmptyBackend()
// 	SEED := "seed phrase here"
// 	apollob := apollo.New(&cc)
// 	apollob = apollob.
// 		SetWalletFromMnemonic(SEED).
// 		SetWalletAsChangeAddress()
// 	utxos := bfc.Utxos(*apollob.GetWallet().GetAddress())
// 	apollob, err := apollob.
// 		AddLoadedUTxOs(utxos).
// 		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 1_000_000, nil).
// 		Complete()
// 	if err != nil {
// 		fmt.Println(err)
// 		t.Error(err)
// 	}
// 	apollob = apollob.Sign()
// 	tx := apollob.GetTx()
// 	cborred, err := cbor.Marshal(tx)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(hex.EncodeToString(cborred))
// 	tx_id, _ := bfc.SubmitTx(*tx)

// 	fmt.Println(hex.EncodeToString(tx_id.Payload))
// 	t.Error("HERE")

// }

func TestFailedSubmissionThrows(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 10_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	_, err = cc.SubmitTx(*apollob.GetTx())
	if err == nil {
		t.Error("DIDNT THROW")
	}
}

func TestBurnPlutus(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f"}
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, MultiAsset.MultiAsset[int64]{
				policy: Asset.Asset[int64]{AssetName.NewAssetNameFromString("TEST"): 1},
			})),
	}

	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: int(-1),
			},
			Redeemer.Redeemer{},
		).
		Complete()
	if err != nil {
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	if hex.EncodeToString(
		txBytes,
	) != "84a5008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000181825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00e2117b021a0002d04509a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354200b5820aed726f17f6c88739b6d5ba2e104b948bb81f6c46e8fc0809c120021c1e6e88ba203800581840000f6820000f5f6" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestMintPlutus(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f"}
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, nil)),
	}

	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: int(1),
			},
			Redeemer.Redeemer{},
		).
		Complete()
	if err != nil {
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	if hex.EncodeToString(
		txBytes,
	) != "84a5008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000181825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c9821a00e20ac7a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa1445445535401021a0002d6f909a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354010b5820aed726f17f6c88739b6d5ba2e104b948bb81f6c46e8fc0809c120021c1e6e88ba203800581840000f6820000f5f6" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestMintPlutusWithPayment(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f"}
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, nil)),
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: int(1),
			},
			Redeemer.Redeemer{},
		).PayToAddress(
		decoded_addr,
		1200000,
		apollo.NewUnit(
			policy.String(),
			"TEST",
			1,
		),
	).Complete()

	if err != nil {
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	if hex.EncodeToString(
		txBytes,
	) != "84a5008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000182825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c9821a00151c56a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa1445445535401825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00cce345021a0002e22509a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354010b5820aed726f17f6c88739b6d5ba2e104b948bb81f6c46e8fc0809c120021c1e6e88ba203800581840000f6820000f5f6" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestGetWallet(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	apollob := apollo.New(&cc)
	wall := apollob.GetWallet()
	if wall != nil {
		t.Error("Wallet should be nil")
	}
	apollob = apollob.SetWalletFromBech32(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	wallet := apollob.GetWallet()
	if wallet.GetAddress().
		String() !=
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu" {
		t.Error("Wallet address is not correct")
	}
}

func TestAddInputs(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, nil)),
	}
	apollob := apollo.New(&cc)
	apollob = apollob.AddInput(testUtxo).SetChangeAddress(decoded_addr)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}

}

func TestConsumeUtxo(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, nil)),
	}
	biAdaUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 1,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, nil)),
	}

	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		ConsumeUTxO(testUtxo,
			apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 2_000_000, nil),
			apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 2_000_000, nil),
		).
		AddLoadedUTxOs(biAdaUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}
	if len(built.GetTx().TransactionBody.Outputs) != 4 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 2_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 2_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[2].Lovelace() != 11_000_000 {
		t.Error("Tx is not correct")
	}
}

func TestConsumeAssetsFromUtxo(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, MultiAsset.MultiAsset[int64]{
				Policy.PolicyId{Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f"}: Asset.Asset[int64]{
					AssetName.NewAssetNameFromString("TEST"): 1,
				},
			})),
	}
	biAdaUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 1,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, nil)),
	}

	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		ConsumeAssetsFromUtxo(testUtxo,
			apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 2_000_000, []apollo.Unit{apollo.NewUnit("279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f", "TEST", 1)}),
		).
		AddLoadedUTxOs(biAdaUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}
	if len(built.GetTx().TransactionBody.Outputs) != 3 {
		t.Error("Tx is not correct")
	}

	if len(built.GetTx().TransactionBody.Outputs[0].GetValue().GetAssets()) != 1 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 15_000_000 {
		t.Error("Tx is not correct")
	}
	if len(built.GetTx().TransactionBody.Outputs[1].GetValue().GetAssets()) != 0 {
		t.Error("Tx is not correct")
	}
}

var decoded_addr, _ = Address.DecodeAddress(
	"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
)

var InputUtxo = UTxO.UTxO{
	Input: TransactionInput.TransactionInput{
		TransactionId: []byte("d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056"),
		Index:         1,
	},
	Output: TransactionOutput.SimpleTransactionOutput(
		decoded_addr,
		Value.SimpleValue(15_000_000, nil)),
}

func TestPayToContract(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	datum := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusBytes,
		Value:          []byte("Hello, World!")}

	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, &datum, 1_000_000, false).
		PayToContract(decoded_addr, &datum, 1_000_000, true)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if len(built.GetTx().TransactionBody.Outputs) != 3 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].IsPostAlonzo {
		t.Error("Tx is not correct")
	}
	if !built.GetTx().TransactionBody.Outputs[1].IsPostAlonzo {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].GetDatumHash() == nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].GetDatumHash() != nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionWitnessSet.PlutusData == nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].GetDatum() == nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].GetDatum() != nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].GetDatum().TagNr != 121 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Outputs[1].GetDatum().TagNr)
	}
	txBytes, _ := built.GetTx().Bytes()
	if hex.EncodeToString(
		txBytes,
	) != "84a4008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536010183835839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a000f4240582037ead362f4ab7844a8416b045caa46a91066d391c16ae4d4a81557f14f7a0984a3005839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c9011a000f4240028201d81850d8794d48656c6c6f2c20576f726c6421825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00c371ff021a0002eb410b5820e3ba8d5e90fa0152fc652eb7e2e8dfd9efa06ce2abd002e69c7b5f8a51be8cd7a1049fd8794d48656c6c6f2c20576f726c6421fff5f6" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}

}

func TestRequiredSigner(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		AddRequiredSignerFromAddress(decoded_addr, true, true)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 2 {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[0][:],
	) != hex.EncodeToString(
		decoded_addr.PaymentPart,
	) {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[1][:],
	) != hex.EncodeToString(
		decoded_addr.StakingPart,
	) {
		t.Error("Tx is not correct")
	}
	apollob = apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		AddRequiredSignerFromBech32(decoded_addr.String(), true, true)
	built, err = apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 2 {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[0][:],
	) != hex.EncodeToString(
		decoded_addr.PaymentPart,
	) {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[1][:],
	) != hex.EncodeToString(
		decoded_addr.StakingPart,
	) {
		t.Error("Tx is not correct")
	}

	apollob = apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		AddRequiredSigner(serialization.PubKeyHash(decoded_addr.PaymentPart))
	built, err = apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 1 {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[0][:],
	) != hex.EncodeToString(
		decoded_addr.PaymentPart,
	) {
		t.Error("Tx is not correct")
	}
}

var collateralUtxo = UTxO.UTxO{
	Input: TransactionInput.TransactionInput{
		TransactionId: []byte("d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056"),
		Index:         1,
	},
	Output: TransactionOutput.SimpleTransactionOutput(
		decoded_addr,
		Value.SimpleValue(5_000_000, nil))}

var collateralUtxo2 = UTxO.UTxO{
	Input: TransactionInput.TransactionInput{
		TransactionId: []byte("d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056"),
		Index:         1,
	},
	Output: TransactionOutput.SimpleTransactionOutput(
		decoded_addr,
		Value.SimpleValue(10_000_000, nil))}

func TestFeePadding(t *testing.T) {
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if built.GetTx().TransactionBody.Fee != 683509 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Fee)
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 13316491 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Outputs[1].Lovelace())
	}
	if built.GetTx().TransactionBody.Outputs[0].IsPostAlonzo &&
		built.GetTx().TransactionBody.Outputs[0].GetDatumHash() != nil {
		t.Error(
			"Tx is not correct",
			built.GetTx().TransactionBody.Outputs[0].GetDatumHash(),
			built.GetTx().TransactionBody.Outputs[0].IsPostAlonzo,
		)
	}
}

func TestSetCollateral(t *testing.T) {
	// full 5 ada collateral
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000).
		AddCollateral(collateralUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Collateral[0].EqualTo(collateralUtxo.Input) {
		t.Error("Tx is not correct")
	}
}

func TestCollateralwithReturn(t *testing.T) {
	// full 5 ada collateral
	cc := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		"mainnetVueasSgKfYhM4PQBq0UGipAyHBpbX4oT",
	)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000).
		AddCollateral(collateralUtxo2)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if built.GetTx().TransactionBody.TotalCollateral != 5_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.CollateralReturn.Lovelace() != 5_000_000 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.CollateralReturn)
	}
	if !built.GetTx().TransactionBody.Collateral[0].EqualTo(collateralUtxo2.Input) {
		t.Error("Tx is not correct")
	}
}

func TestRedeemerCollect(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	redeemer := Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusBytes,
			Value:          []byte("Hello, World!")},
	}
	datum := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusBytes,
		Value:          []byte("Hello, World!")}

	utxos := testutils.InitUtxosDifferentiated()
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(utxos...).
		CollectFrom(InputUtxo, redeemer).
		AttachDatum(&datum).AttachV1Script([]byte("Hello, World!")).SetEstimationExUnitsRequired()
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(InputUtxo.Input) {
		t.Error("Tx is not correct")
	}
	wts := built.GetTx().TransactionWitnessSet
	if wts.Redeemer[0].Tag != Redeemer.SPEND {
		t.Error("Tx is not correct")
	}
	if wts.Redeemer[0].Index != 0 {
		t.Error("Tx is not correct")
	}
	if wts.Redeemer[0].Data.TagNr != 121 {
		t.Error("Tx is not correct")
	}
	if wts.Redeemer[0].Data.PlutusDataType != PlutusData.PlutusBytes {
		t.Error("Tx is not correct")
	}
	if string(wts.Redeemer[0].Data.Value.([]byte)) != "Hello, World!" {
		t.Error("Tx is not correct")
	}
	if wts.PlutusData == nil {
		t.Error("Tx is not correct")
	}
	if wts.PlutusData[0].TagNr != 121 {
		t.Error("Tx is not correct")
	}
	if wts.PlutusData[0].PlutusDataType != PlutusData.PlutusBytes {
		t.Error("Tx is not correct")
	}
	if string(wts.PlutusData[0].Value.([]byte)) != "Hello, World!" {
		t.Error("Tx is not correct")
	}

	if string(wts.PlutusV1Script[0]) != "Hello, World!" {
		t.Error("Tx is not correct")
	}

	if wts.Redeemer[0].ExUnits.Mem == 0 {
		t.Error("Tx is not correct", wts.Redeemer[0].ExUnits.Mem)
	}
	if wts.Redeemer[0].ExUnits.Steps == 0 {
		t.Error("Tx is not correct", wts.Redeemer[0].ExUnits.Steps)
	}
	if built.GetTx().TransactionBody.Fee != 228771 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Fee)
	}
	if built.GetTx().TransactionBody.Collateral == nil {
		t.Error("Tx is not correct")
	}

}

func TestAddSameScriptTwiceV1(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	utxos := testutils.InitUtxosDifferentiated()
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.AttachV1Script([]byte("Hello, World!")).
		AttachV1Script([]byte("Hello, World!"))
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(utxos...)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)

	}
	if len(built.GetTx().TransactionWitnessSet.PlutusV1Script) != 1 {
		t.Error("Tx is not correct")
	}
}

func TestAddSameScriptTwiceV2(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	utxos := testutils.InitUtxosDifferentiated()
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.AttachV2Script([]byte("Hello, World!")).
		AttachV2Script([]byte("Hello, World!"))
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(utxos...)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if len(built.GetTx().TransactionWitnessSet.PlutusV2Script) != 1 {
		t.Error("Tx is not correct")
	}
}

func TestSetChangeAddressBech32(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
		AddInput(InputUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if built.GetTx().TransactionBody.Outputs[0].GetAddress().
		String() !=
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu" {
		t.Error("Tx is not correct")
	}
}

func TestSetWalletFromBech32(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	apollob := apollo.New(&cc)
	apollob = apollob.SetWalletFromBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
		SetWalletAsChangeAddress().
		AddInput(InputUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if built.GetTx().TransactionBody.Outputs[0].GetAddress().
		String() !=
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu" {
		t.Error("Tx is not correct")
	}
}

func TestRefInput(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	apollob := apollo.New(&cc)
	apollob = apollob.SetWalletFromBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
		SetWalletAsChangeAddress().
		AddInput(InputUtxo).
		AddReferenceInput(hex.EncodeToString(InputUtxo.Input.TransactionId), 0).
		AddCollateral(collateralUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.ReferenceInputs[0].TransactionId,
	) != hex.EncodeToString(
		InputUtxo.Input.TransactionId,
	) {
		t.Error("Tx is not correct")
	}
}

func TestExactComplete(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	redeemer := Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusBytes,
			Value:          []byte("Hello, World!")},
	}
	datum := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusBytes,
		Value:          []byte("Hello, World!")}

	utxos := testutils.InitUtxosDifferentiated()
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(utxos...).
		CollectFrom(InputUtxo, redeemer).
		AttachDatum(&datum).AttachV1Script([]byte("Hello, World!")).SetEstimationExUnitsRequired()
	built, err := apollob.CompleteExact(200_000)
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(InputUtxo.Input) {
		t.Error("Tx is not correct")
	}
	wts := built.GetTx().TransactionWitnessSet
	if wts.Redeemer[0].Tag != Redeemer.SPEND {
		t.Error("Tx is not correct")
	}
	if wts.Redeemer[0].Index != 0 {
		t.Error("Tx is not correct")
	}
	if wts.Redeemer[0].Data.TagNr != 121 {
		t.Error("Tx is not correct")
	}
	if wts.Redeemer[0].Data.PlutusDataType != PlutusData.PlutusBytes {
		t.Error("Tx is not correct")
	}
	if string(wts.Redeemer[0].Data.Value.([]byte)) != "Hello, World!" {
		t.Error("Tx is not correct")
	}
	if wts.PlutusData == nil {
		t.Error("Tx is not correct")
	}
	if wts.PlutusData[0].TagNr != 121 {
		t.Error("Tx is not correct")
	}
	if wts.PlutusData[0].PlutusDataType != PlutusData.PlutusBytes {
		t.Error("Tx is not correct")
	}
	if string(wts.PlutusData[0].Value.([]byte)) != "Hello, World!" {
		t.Error("Tx is not correct")
	}

	if string(wts.PlutusV1Script[0]) != "Hello, World!" {
		t.Error("Tx is not correct")
	}

	if wts.Redeemer[0].ExUnits.Mem == 0 {
		t.Error("Tx is not correct", wts.Redeemer[0].ExUnits.Mem)
	}
	if wts.Redeemer[0].ExUnits.Steps == 0 {
		t.Error("Tx is not correct", wts.Redeemer[0].ExUnits.Steps)
	}
	if built.GetTx().TransactionBody.Fee != 200_000 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Fee)
	}
	if built.GetTx().TransactionBody.Collateral == nil {
		t.Error("Tx is not correct")
	}
}

func TestCongestedBuild(t *testing.T) {
	cc := apollo.NewEmptyBackend()
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	utxos := testutils.InitUtxosCongested()
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(utxos...).
		AddPayment(apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seya", 150_000_000, nil))
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if len(built.GetTx().TransactionBody.Outputs) == 2 {
		t.Error("Tx is not correct")
	}
}

func TestMapPDDecodeEncode(t *testing.T) {
	val := "84ac00828258205af098c47e6539e03dce07fb9cd83ba630bd901e6943db894c70f656a480fa5a01825820ca0ebc40b02aeeaa0c8a84be94c9146bad85f249d6b98f840fd70f894a1de370020185a30058391155ff0e63efa0694e8065122c552e80c7b51768b7f20917af25752a7c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e401821a006acfc0a1581c3c468b2a275a7df4b33625335232d4cfb45e651d289b2e0737856184a147457a43464f4e4101028201d81858b9d8799fa200a140a1401a001e848001a09fd8799f4040ffd8799f581c420000029ad9527271b1b1e3c27ee065c18df70a4a4cfc3093a41a444341584fffffd8799f581c3c468b2a275a7df4b33625335232d4cfb45e651d289b2e073785618447457a43464f4e41ffa3457072696365d87b9fd8799f1b00102a3f18d9852f1b06f05b59d3b20000ffff47656e6444617465d905009f1b000003bb2cc3d418ff49737461727444617465d905009f1b00000092f3973818ffa0ff82583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4821a001e8480a1581c3c468b2a275a7df4b33625335232d4cfb45e651d289b2e0737856184a147457a43464f4e410182583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4821a00198ef8a1581c95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb00a14448554e541a01040cc3825839018b0fac6777891b925d646af1727d1ef288338f7966455a93ce31dfbc3120b7074a8521bcc9ea2f05e38cf8924ffb7bc871ebfaa35ae98bf21a000f424082583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a1d6593a1021a0004eec9031a0743ce5a075820411feee1738c31e1ac0e860abd4afd3c8288b6cd1abb051f21bff91a7a796f59081a0743cad609a1581c3c468b2a275a7df4b33625335232d4cfb45e651d289b2e0737856184a147457a43464f4e41020b5820041cf2ae5f43362e977280abbb0e9871c1c829a01bc59560d06d384b4d1bfc2c0d81825820b4bf6f7a29915cdf1aaac9d2112fc986bb3227d9cd04d7af418991cee23b07ed010e82581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41082583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4821a03efae66b3581c078eafce5cd7edafdf63900edef2c1ea759e77f30ca81d6bbdeec924a14579756d6d691904f3581c115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818a144746573741a000f4240581c1ddcb9c9de95361565392c5bdff64767492d61a96166cb16094e54bea1434f50541a02c5e3be581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa144534e454b191585581c29d222ce763455e3d7a09a665ce554f00ac89d2e99a1a83d267170c6a1434d494e1a15b511cd581c32335fbb01744e526da8b9f97d759c2c07457c4c55eab98c372cdad6a14953505f494c7530634a02581c420000029ad9527271b1b1e3c27ee065c18df70a4a4cfc3093a41a44a14341584f1a133d12a2581c52162581184a457fad70470161179c5766f00237d4b67e0f1df1b4e6a1445452544c01581c562c0e6da43a062dde7d05b494d6b1a9d0d06a9d36131956c327b127a14953505f447a4a61657601581c5d16cc1a177b5d9ba9cfa9793b07e60f1fb70fea1f8aef064415d114a1434941471a018ece34581c826733e8e5d12c797f795be56b348982397e235d57074a992cb86b6ca14553544556451a0032b0bd581c8640914f83348a1b68d4b32205e2e4741455897e3e9edfe270d9069fa151576f6c6653796e6469636174653132343501581c8a1cfae21368b8bebbbed9800fec304e95cce39a2a57dc35e2e3ebaaa1444d494c4b01581c8fef2d34078659493ce161a6c7fba4b56afefa8535296a5743f69587a144414144411a0001e582581c95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb00a14448554e541a0e6d983e581caf2e27f580f7f08e93190a81f72462f153026d06450924726645891ba144445249501a89e84f7a581cb3ad8b975d24235a43cb2a54d58c717ed9dd11560b4deba2273ffb1da1480014df104b5749431a3a06b367581cc0ee29a85b13209423b10447d3c2e6a50641a15c57770e27cb9d5073a14a57696e675269646572731a00470943581cdda5fdb1002f7389b33e036b6afee82a8189becb6cba852e8b79b4fba1480014df1047454e531a0047165c111a004c4b40a30380068159022659022301000032323232323232323232323232223232323232533300c3370e9000001099999199111980711299980a80089128008a99980919baf301630180010041300530180011300230170010012322230020033756602c0026ea4004dd7180900099199180511998011bab001232223002003374c002244a002464a66601e6ae8c0044894004488c00800ccc02894ccc03ccdd78009ba8480004894004488c00800c004004dd598091918091809180900098088021299980699807180780099111801001a4008264649319999806111299980a00089128008a9998089801180b800899111801001980b800899801801180b0009199119baf374e60300046e9cc06000530012bd8799fd8799f58205af098c47e6539e03dce07fb9cd83ba630bd901e6943db894c70f656a480fa5aff01ff00001001200116332300c22533301300114bd700998079801980b0009801180a8009180a180a8009bac30130051533300d3300e300f0013222300200332337029000000a40082930b0b0b180980118070009baa300f300e002300f300e001300e001223300422533300b00110051323330053011300f00223300933760601c602000600200420026004601a00200297adef6c602323002233002002001230022330020020015740ae6888cc0088cc0088cdc38010008a5013300124a0294494ccc00800448940044c94ccc00c0044c888c00800cdd69804180300109128009802000aab9f5573aae895d0918011baa0015573d0581840100d87980821a00014c221a01da9e55f5a21902a2a1636d7367816f44657868756e7465722054726164651902d1a178383363343638623261323735613764663462333336323533333532333264346366623435653635316432383962326530373337383536313834a167457a43464f4e41a664616c676fa3646c696e6b82784068747470733a2f2f6170702e61786f2e74726164652f636f6d706f7365722f61326634383739302d393432362d343361372d383134372d61656135313761386363663761646e616d656c536d617274204d61726b65746b6465736372697074696f6e60646e616d6577536d617274204d61726b6574203c4144412c2041584f3e657374726174a3646c696e6b782d68747470733a2f2f6170702e61786f2e74726164652f737472617465676965732f766965772f457a43464f4e41646e616d65606b6465736372697074696f6e60666173736574738260783e343230303030303239616439353237323731623162316533633237656530363563313864663730613461346366633330393361343161343434313538346667776562736974657168747470733a2f2f61786f2e7472616465696d656469615479706569696d6167652f706e67"
	tx := Transaction.Transaction{}
	decodedCbor, _ := hex.DecodeString(val)
	err := cbor.Unmarshal(decodedCbor, &tx)
	if err != nil {
		t.Error(err)
	}
	recoded, err := cbor.Marshal(tx)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(recoded) != val {
		fmt.Println(val, hex.EncodeToString(recoded))
		t.Error("Tx is not correct")
	}
}

func TestMapPdRecode(t *testing.T) {
	for i := 0; i < 100; i++ {

		pdCbor := "d8799fa200a140a1401a01312d0001a09fd8799f4040ffd8799f581c95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb004448554e54ffffd8799f581c500cebfc153990c15c001e3441a0e0774634cfe1864d4f2e07e17474472f426b424b4f41ffa3457072696365d87b9fd8799f1b0001b57b632d33951b00071afd498d0000ffff47656e6444617465d905009f1b000003bb2cc3d418ff49737461727444617465d905009f1b00000092f3973818ffa0ff"
		pd := PlutusData.PlutusData{}
		decodedCbor, _ := hex.DecodeString(pdCbor)
		err := cbor.Unmarshal(decodedCbor, &pd)
		if err != nil {
			t.Error(err)
		}
		recoded, err := cbor.Marshal(pd)
		if err != nil {
			t.Error(err)
		}
		if hex.EncodeToString(recoded) != pdCbor {
			fmt.Println(pdCbor, hex.EncodeToString(recoded))
			t.Error("Tx is not correct after", i)
			break
		}
	}
}

func TestRecodeTx(t *testing.T) {
	for i := 0; i < 100; i++ {
		txCbor := "84ac0083825820cda3926787c51f124e530d255fe8841aac9eed3be2fe518c8a44c801e75b204e0182582010453108b22eb9ea4c43e2aef257f269b6624f39dce0175fff44513bcc2f297200825820cda3926787c51f124e530d255fe8841aac9eed3be2fe518c8a44c801e75b204e040185a30058391155ff0e63efa0694e8065122c552e80c7b51768b7f20917af25752a7cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce01821a004c4b40a2581c420000029ad9527271b1b1e3c27ee065c18df70a4a4cfc3093a41a44a14341584f1a45632e08581cc6b287b7b14d19db292f203824ac676b128eeebe8d09c3612755a09ba14754514e634d504101028201d81858d9d8799fa200a1581c420000029ad9527271b1b1e3c27ee065c18df70a4a4cfc3093a41a44a14341584f1a45632e0801a09fd8799f581c420000029ad9527271b1b1e3c27ee065c18df70a4a4cfc3093a41a444341584fffd8799f4040ffffd8799f581cc6b287b7b14d19db292f203824ac676b128eeebe8d09c3612755a09b4754514e634d5041ffa3457072696365d87b9fd8799f1b00029e05316a59991b000005af3107a400ffff47656e6444617465d905009f1b000003bb2cc3d418ff49737461727444617465d905009f1b00000092f3973818ffa0ff8258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce821a001e8480a1581cc6b287b7b14d19db292f203824ac676b128eeebe8d09c3612755a09ba14754514e634d5041018258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce821a001e8480a1581cf0fc4932134ad45a42ef9e5591a16dc88fff230bc4487eb0f1774f8ca1475250364146484101825839018b0fac6777891b925d646af1727d1ef288338f7966455a93ce31dfbc3120b7074a8521bcc9ea2f05e38cf8924ffb7bc871ebfaa35ae98bf21a000f42408258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a1deeb464021a0004c1b2031a074cf233075820b83a67b95c0136a1490445e62f9878de1ed0a5f3fd0e7d143741a923d88f820e081a074ceeaf09a1581cc6b287b7b14d19db292f203824ac676b128eeebe8d09c3612755a09ba14754514e634d5041020b58208a0898f7e654d715598e9bd1bd6d9467bd35cd41e112edfb0b704895592949040d81825820b65b15369b13ee118915606e86f28f0a84df22f599deafbd11aa9b7671623655000e82581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bc581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce108258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce1a00181f91111a004c4b40a30380068159022659022301000032323232323232323232323232223232323232533300c3370e9000001099999199111980711299980a80089128008a99980919baf301630180010041300530180011300230170010012322230020033756602c0026ea4004dd7180900099199180511998011bab001232223002003374c002244a002464a66601e6ae8c0044894004488c00800ccc02894ccc03ccdd78009ba8480004894004488c00800c004004dd598091918091809180900098088021299980699807180780099111801001a4008264649319999806111299980a00089128008a9998089801180b800899111801001980b800899801801180b0009199119baf374e60300046e9cc06000530012bd8799fd8799f5820cda3926787c51f124e530d255fe8841aac9eed3be2fe518c8a44c801e75b204eff01ff00001001200116332300c22533301300114bd700998079801980b0009801180a8009180a180a8009bac30130051533300d3300e300f0013222300200332337029000000a40082930b0b0b180980118070009baa300f300e002300f300e001300e001223300422533300b00110051323330053011300f00223300933760601c602000600200420026004601a00200297adef6c602323002233002002001230022330020020015740ae6888cc0088cc0088cdc38010008a5013300124a0294494ccc00800448940044c94ccc00c0044c888c00800cdd69804180300109128009802000aab9f5573aae895d0918011baa0015573d0581840100d87980821a0001778c1a0225e159f5a21902a2a1636d7367816f44657868756e7465722054726164651902d1a178386336623238376237623134643139646232393266323033383234616336373662313238656565626538643039633336313237353561303962a16754514e634d5041a964616c676fa4646c696e6b82784068747470733a2f2f6170702e61786f2e74726164652f636f6d706f7365722f61326634383739302d393432362d343361372d383134372d61656135313761386363663761646e616d656c536d617274204d61726b6574647574786f82784036393762386437356463323736613835383734653834363438313738626635653538363862646335356461623161323338613136643834343835613765626165a2696e756d657261746f728200806b64656e6f6d696e61746f72820181016b6465736372697074696f6e6064696e6974a263414441a2696e756d657261746f728200806b64656e6f6d696e61746f72820181016341584fa2696e756d657261746f728201811a45632e086b64656e6f6d696e61746f7282018101646e616d6577536d617274204d61726b6574203c41584f2c204144413e65696d6167657835697066733a2f2f516d517a4a64464e7a39564e42384e53764c6242546a4e7a6f74794e59353656387376664141557a5259626e3738657374726174a3646c696e6b782d68747470733a2f2f6170702e61786f2e74726164652f737472617465676965732f766965772f54514e634d5041646e616d65606b6465736372697074696f6e60666173736574738281783e3432303030303032396164393532373237316231623165336332376565303635633138646637306134613463666333303933613431613434343135383466816066706172616d73a4657072696365a2696e756d657261746f728201821ad77c1fd1016b64656e6f6d696e61746f728201811a0400000066616d6f756e74a2696e756d657261746f728201811a45632e086b64656e6f6d696e61746f728201810167656e644461746574323039392d31322d33315432333a35393a35395a6973746172744461746574313938392d31322d33315432333a35393a35395a67776562736974657168747470733a2f2f61786f2e7472616465696d656469615479706569696d6167652f706e67"
		tx := Transaction.Transaction{}
		decodedCbor, _ := hex.DecodeString(txCbor)
		err := cbor.Unmarshal(decodedCbor, &tx)
		if err != nil {
			t.Error(err)

		}
		recoded, err := cbor.Marshal(tx)
		if err != nil {
			t.Error(err)

		}
		if hex.EncodeToString(recoded) != txCbor {
			fmt.Println(txCbor, hex.EncodeToString(recoded))
			t.Error("Tx is not correct")
			break
		}
	}
}

func TestMapWithCbor(t *testing.T) {
	for i := 0; i < 100; i++ {
		data := PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusMap,
			Value: map[serialization.CustomBytes]PlutusData.PlutusData{

				serialization.NewCustomBytes("endDate"): PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusInt,
					Value:          100,
				},
				serialization.NewCustomBytes("startDate"): PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusInt,
					Value:          100,
				}, serialization.NewCustomBytes("price"): PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusInt,
					Value:          100,
				},
			},
		}
		encoded, err := cbor.Marshal(data)
		if err != nil {
			t.Error(err)
		}
		if hex.EncodeToString(encoded) != "d879a3457072696365186447656e64446174651864497374617274446174651864" {
			t.Error(hex.EncodeToString(encoded))
			break
		}
	}

}

func TestPlutusBigInt(t *testing.T) {
	x := "c249056bc75e2d63100000"
	decoded, _ := hex.DecodeString(x)
	var y PlutusData.PlutusData
	err := cbor.Unmarshal(decoded, &y)
	if err != nil {
		t.Error(err)
	}
	if y.PlutusDataType != PlutusData.PlutusBigInt {
		t.Error("Tx is not correct")
	}
	tmpBI := y.Value.(big.Int)
	if tmpBI.String() != "100000000000000000000" {
		t.Error("Tx is not correct")
	}
}
