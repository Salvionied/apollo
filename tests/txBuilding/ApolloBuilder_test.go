package txBuilding_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Salvionied/cbor/v2"
	"github.com/SundaeSwap-finance/apollo"
	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/Amount"
	"github.com/SundaeSwap-finance/apollo/serialization/MultiAsset"
	"github.com/SundaeSwap-finance/apollo/serialization/PlutusData"
	"github.com/SundaeSwap-finance/apollo/serialization/Transaction"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionInput"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionOutput"
	"github.com/SundaeSwap-finance/apollo/serialization/UTxO"
	"github.com/SundaeSwap-finance/apollo/serialization/Value"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/FixedChainContext"
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
	apollob, _, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(hex.EncodeToString(apollob.GetTx().Bytes()))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf("%s:%d", hex.EncodeToString(input.TransactionId), input.Index) {
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
	decodedAddress, _ := Address.DecodeAddress("addr1wxr2a8htmzuhj39y2gq7ftkpxv98y2g67tg8zezthgq4jkg0a4ul4")
	apollob = apollob.AddInputAddressFromBech32(userAddress).AddLoadedUTxOs(utxos...).
		PayToContract(decodedAddress, &pd,
			4000000,
			false,
			apollo.NewUnit("279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f", "SNEK", 10416)).
		PayToAddressBech32("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh", int(2_000_000)).
		SetTtl(0 + 300).
		SetValidityStart(0)
	apollob, _, err = apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(hex.EncodeToString(apollob.GetTx().Bytes()))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf("%s:%d", hex.EncodeToString(input.TransactionId), input.Index) {
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
	decodedAddress, _ := Address.DecodeAddress("addr1wxr2a8htmzuhj39y2gq7ftkpxv98y2g67tg8zezthgq4jkg0a4ul4")
	apollob = apollob.AddInputAddressFromBech32(userAddress).AddLoadedUTxOs(utxos...).
		PayToContract(decodedAddress, &pd,
			4000000,
			false,
			apollo.NewUnit("279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f", "SNEK", 10416)).
		PayToAddressBech32("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh", int(2_000_000)).
		SetTtl(0 + 300).
		SetValidityStart(0).MintAssets(
		apollo.NewUnit("f0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9a", "bluedesert", -1),
	)
	apollob, _, err = apollob.Complete()
	if err != nil {
		fmt.Println("HERE")
		t.Error(err)
	}
	//t.Error("STOP")
	txBytes := apollob.GetTx().Bytes()
	fmt.Println(hex.EncodeToString(txBytes))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf("%s:%d", hex.EncodeToString(input.TransactionId), input.Index) {
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

func makeFakeUtxo(address Address.Address, index int, lovelace int64) UTxO.UTxO {
	u := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Index:         index,
		},
		Output: TransactionOutput.TransactionOutput{
			PreAlonzo: TransactionOutput.TransactionOutputShelley{
				Address: address,
				Amount: Value.Value{
					Am: Amount.Amount{
						Coin: 0,
					},
					Coin:      lovelace,
					HasAssets: false,
				},
				DatumHash: serialization.DatumHash(serialization.ConstrainedBytes{Payload: nil}),
				HasDatum:  false,
			},
			IsPostAlonzo: false,
		},
	}
	return u
}

func TestUseInputAsCollateral(t *testing.T) {
	cc := FixedChainContext.InitFixedChainContext()
	userAddress := "addr1qymaeeefs9ff08cdplm3lvkscavm9x9vd7nmc44e9rlur08k3pj2xw9w3mvp7cg3fkzhed4zzhywdpd2t3pmc8u8nn8qm5ur5w"
	myAddress, _ := Address.DecodeAddress("addr1qymaeeefs9ff08cdplm3lvkscavm9x9vd7nmc44e9rlur08k3pj2xw9w3mvp7cg3fkzhed4zzhywdpd2t3pmc8u8nn8qm5ur5w")
	// dummy script that always passes. we don't actually spend from this
	// it's just here to encourage apollo to attach a collateral
	script, err := hex.DecodeString("51010000322253330034a229309b2b2b9a01")
	apollob := apollo.New(&cc)
	utxos := make([]UTxO.UTxO, 0)
	utxos = append(utxos, makeFakeUtxo(myAddress, 0, 100_000_000))
	apollob = apollob.AddInputAddressFromBech32(userAddress).AddLoadedUTxOs(utxos...).
		PayToAddressBech32(userAddress, int(2_000_000)).
		SetTtl(0 + 300).
		SetValidityStart(0)
	apollob = apollob.AttachV2Script(script)
	apollob, _, err = apollob.Complete()
	if err != nil {
		fmt.Println("HERE")
		t.Error(err)
	}
	txBytes := apollob.GetTx().Bytes()
	fmt.Println(hex.EncodeToString(txBytes))
	inputVal := Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{})
	inputs := apollob.GetTx().TransactionBody.Inputs
	collaterals := apollob.GetTx().TransactionBody.Collateral
	if len(inputs) != 1 {
		t.Error("Tx does not have exactly 1 input")
	}
	if len(collaterals) != 1 {
		t.Error("Tx does not have exactly 1 collateral")
	}
	if !bytes.Equal(inputs[0].TransactionId, collaterals[0].TransactionId) || inputs[0].Index != collaterals[0].Index {
		t.Error("Tx does not have the same collateral as its input")
	}
	for _, input := range apollob.GetTx().TransactionBody.Inputs {
		for _, utxo := range utxos {
			if utxo.GetKey() == fmt.Sprintf("%s:%d", hex.EncodeToString(input.TransactionId), input.Index) {
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
	if err != nil {
		t.Error(err)
	}
}

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

// func TestSimpleBuild(t *testing.T) {
// 	cc := BlockFrostChainContext.NewBlockfrostChainContext("blockfrost_api_key", int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
// 	apollob := apollo.New(&cc)
// 	apollob, err := apollob.
// 		AddInputAddressFromBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
// 		AddLoadedUTxOs(initUtxosDifferentiated()).
// 		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 10_000_000, nil).
// 		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
// 			5_000_000,
// 			[]apollo.Unit{{"00000000000000000000000000000000000000000000000000000000", "token0", 1}, {"00000000000000000000000000000000000000000000000000000000", "token2", 3}},
// 		).Complete()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	cborred, _ := cbor.Marshal(apollob.GetTx())
// 	fmt.Println(apollob.GetTx().TransactionBody.CollateralReturn, apollob.GetTx().TransactionBody.Withdrawals)
// 	fmt.Println(hex.EncodeToString(cborred))
// }
