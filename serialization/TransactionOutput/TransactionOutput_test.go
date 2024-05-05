package TransactionOutput_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/cbor/v2"
)

const TEST_POLICY = "115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818"
const TEST_ADDRESS = "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh"

func TestTransactionOutputWithDatumHash(t *testing.T) {
	cborHex := "83583911a65ca58a4e9c755fa830173d2a5caed458ac0c73f97db7faae2e7e3b52563c5410bff6a0d43ccebb7c37e1f69f5eb260552521adff33b9c21a00895440582070c5d760293d3d92bfa7369e472891ab36041cbf81fd5ed103462fb7c03f2a6e"
	cborBytes, _ := hex.DecodeString(cborHex)
	txOut := TransactionOutput.TransactionOutput{}
	err := txOut.UnmarshalCBOR(cborBytes)
	if err != nil {
		t.Errorf("Error while unmarshaling")
	}
	outBytes, err := cbor.Marshal(txOut)
	if err != nil {
		t.Errorf("Error while marshaling")
	}

	if hex.EncodeToString(outBytes) != cborHex {
		t.Errorf("Invalid Reserialization")
	}
}

func TestPostAlonzo(t *testing.T) {
	txO := TransactionOutput.TransactionOutput{}
	cborHex := "d8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd87a80d8799fd8799f581c25f0fc240e91bd95dcdaebd2ba7713fc5168ac77234a3d79449fc20c47534f4349455459ff1b00002cc16be02b37ff1a001e84801a001e8480ff"
	decoded_cbor, _ := hex.DecodeString(cborHex)
	var pd PlutusData.PlutusData
	cbor.Unmarshal(decoded_cbor, &pd)
	txO.IsPostAlonzo = true
	decoded_address, _ := Address.DecodeAddress("addr1wynp362vmvr8jtc946d3a3utqgclfdl5y9d3kn849e359hsskr20n")
	txO.PostAlonzo = TransactionOutput.TransactionOutputAlonzo{}
	txO.PostAlonzo.Address = decoded_address
	txO.PostAlonzo.Amount = Value.PureLovelaceValue(1000000).ToAlonzoValue()
	d := PlutusData.DatumOptionInline(&pd)
	txO.PostAlonzo.Datum = &d
	resultHex := "a300581d712618e94cdb06792f05ae9b1ec78b0231f4b7f4215b1b4cf52e6342de011a000f4240028201d81858e8d8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd87a80d8799fd8799f581c25f0fc240e91bd95dcdaebd2ba7713fc5168ac77234a3d79449fc20c47534f4349455459ff1b00002cc16be02b37ff1a001e84801a001e8480ff"
	cborred, _ := cbor.Marshal(txO)
	if hex.EncodeToString(cborred) != resultHex {
		t.Errorf("Invalid marshaling")
	}

}

func TestDeSerializeTxWithPostAlonzoOut(t *testing.T) {
	cborHex := "84a500838258205628043acaccaf3e07ce6d93bec8da6ae013d2546aa1f491c68dfa2942e6aab401825820250cb6fab4bab5fe0746748cdb8dd42b545328ecc8109e16cd56c0ca9382c7bb028258205e9344d4529b623cb1e17b5a041f58f8275e0fdea54c52a7dc73e0d47ff2fe1a010183a300581d712618e94cdb06792f05ae9b1ec78b0231f4b7f4215b1b4cf52e6342de01821a00e4e1c0a0028201d81858bfd8799fd8799f4040ffd8799f581cf43a62fdc3965df486de8a0d32fe800963589c41b38946602a0dc5354441474958ffd8799f581cfd011feb9dc34f85e58e56838989816343f5c62619a82f6a089f05484c414749585f4144415f4e4654ff1903e51b002904d642c7b27c1b7fffffffffffffff581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceff1a009896801a4d6fd4bcff82583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a000fea4c8258390137dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cce821a0633d59aab581c10a49b996e2402269af553a8a96fb8eb90d79e9eca79e2b4223057b6a1444745524f1a001e8480581c25f0fc240e91bd95dcdaebd2ba7713fc5168ac77234a3d79449fc20ca147534f43494554591b00000019e1ae3741581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa144534e454b1928b0581c29d222ce763455e3d7a09a665ce554f00ac89d2e99a1a83d267170c6a1434d494e1a0cb30355581c533bb94a8850ee3ccbe483106489399112b74c905342cb1792a797a0a144494e44591a156f14e4581c5d16cc1a177b5d9ba9cfa9793b07e60f1fb70fea1f8aef064415d114a1434941471b0000002e921a6381581c8a1cfae21368b8bebbbed9800fec304e95cce39a2a57dc35e2e3ebaaa1444d494c4b05581c8b4e239aef4d1d1bc5dd628ff3ce34d392d632e5cda83e42d6fcb1cca14b586572636865723234393301581cd480f68af028d6324ad77df489176e7f5e5d793e09a6b133392ff2f6aa524e7563617374496e63657074696f6e31343101524e7563617374496e63657074696f6e32303601524e7563617374496e63657074696f6e33323101524e7563617374496e63657074696f6e33383501524e7563617374496e63657074696f6e34303001524e7563617374496e63657074696f6e36333701524e7563617374496e63657074696f6e36373001524e7563617374496e63657074696f6e37383701524e7563617374496e63657074696f6e38333301524e7563617374496e63657074696f6e38373001581ce3ff4ab89245ede61b3e2beab0443dbcc7ea8ca2c017478e4e8990e2a549746170707930333831014974617070793034313901497461707079313430390149746170707931343437014974617070793135353001581cf0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9aa24a626c7565646573657274014a6d6f6e74626c616e636f01021a000342dd031a05fd33e3081a05fd32b7a1049ffff5f6"

	decoded_cbor, _ := hex.DecodeString(cborHex)
	var tx Transaction.Transaction

	err := cbor.Unmarshal(decoded_cbor, &tx)
	if err != nil {
		t.Error("Error while unmarshaling", err)
	}
	remarshaled, err := cbor.Marshal(tx)
	if err != nil {
		t.Error("Error While remarshaling", err)
	}
	if hex.EncodeToString(remarshaled) != cborHex {
		t.Error("Error while reserializing", hex.EncodeToString(remarshaled))
	}

}

func TestValueSerialization(t *testing.T) {
	ShelleyValueWithNoAssets := Value.PureLovelaceValue(1000000)
	ShelleyValueWithAssets := Value.SimpleValue(1_000_000, MultiAsset.MultiAsset[int64]{
		Policy.PolicyId{"115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818"}: Asset.Asset[int64]{
			AssetName.NewAssetNameFromString("Token1"): 1,
		},
	})
	AlonzoValueWithNoAssets := Value.PureLovelaceValue(1000000).ToAlonzoValue()
	AlonzoValueWithAssets := Value.SimpleValue(1_000_000, MultiAsset.MultiAsset[int64]{
		Policy.PolicyId{"115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818"}: Asset.Asset[int64]{
			AssetName.NewAssetNameFromString("Token1"): 1,
		},
	}).ToAlonzoValue()
	ShelleyValueWithNoAssetsBytes, _ := cbor.Marshal(ShelleyValueWithNoAssets)
	ShelleyValueWithAssetsBytes, _ := cbor.Marshal(ShelleyValueWithAssets)
	AlonzoValueWithNoAssetsBytes, _ := cbor.Marshal(AlonzoValueWithNoAssets)
	AlonzoValueWithAssetsBytes, _ := cbor.Marshal(AlonzoValueWithAssets)
	if hex.EncodeToString(ShelleyValueWithNoAssetsBytes) != "1a000f4240" {
		t.Error("ShelleyValueWithNoAssetsBytes")
	}
	if hex.EncodeToString(AlonzoValueWithNoAssetsBytes) != "1a000f4240" {
		t.Error("AlonzoValueWithNoAssetsBytes")
	}
	if hex.EncodeToString(ShelleyValueWithAssetsBytes) != "821a000f4240a1581c115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818a146546f6b656e3101" {
		t.Error("ShelleyValueWithAssetsBytes")
	}
	if hex.EncodeToString(AlonzoValueWithAssetsBytes) != "821a000f4240a1581c115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818a146546f6b656e3101" {
		t.Error("AlonzoValueWithAssetsBytes")
	}
}

func TestTransactionOutputPostAlonzoUtils(t *testing.T) {
	addr, _ := Address.DecodeAddress(TEST_ADDRESS)
	toNoAssets := TransactionOutput.TransactionOutputAlonzo{
		Address: addr,
		Amount:  Value.PureLovelaceValue(1000000).ToAlonzoValue(),
	}
	toWithAssets := TransactionOutput.TransactionOutputAlonzo{
		Address: addr,
		Amount: Value.SimpleValue(1_000_000, MultiAsset.MultiAsset[int64]{
			Policy.PolicyId{TEST_POLICY}: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("Token1"): 1,
			},
		}).ToAlonzoValue(),
	}
	clonedNoAssets := toNoAssets.Clone()
	clonedWithAssets := toWithAssets.Clone()
	if !toNoAssets.Amount.ToValue().Equal(clonedNoAssets.Amount.ToValue()) || toNoAssets.Address.String() != clonedNoAssets.Address.String() || &toNoAssets == &clonedNoAssets {
		t.Error("Error while cloning")
	}
	if !toWithAssets.Amount.ToValue().Equal(clonedWithAssets.Amount.ToValue()) || toWithAssets.Address.String() != clonedWithAssets.Address.String() || &toWithAssets == &clonedWithAssets {
		t.Error("Error while cloning with assets")
	}

	if toNoAssets.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh:1000000 Datum :<nil>" {
		t.Error("Error while stringifying, got", toNoAssets.String())
	}
}

func TestTransactionOutputShelleyUtils(t *testing.T) {
	addr, _ := Address.DecodeAddress(TEST_ADDRESS)
	toNoAssets := TransactionOutput.TransactionOutputShelley{
		Address: addr,
		Amount:  Value.PureLovelaceValue(1000000),
	}
	toWithAssets := TransactionOutput.TransactionOutputShelley{
		Address: addr,
		Amount: Value.SimpleValue(1_000_000, MultiAsset.MultiAsset[int64]{
			Policy.PolicyId{TEST_POLICY}: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("Token1"): 1,
			},
		}),
	}
	clonedNoAssets := toNoAssets.Clone()
	clonedWithAssets := toWithAssets.Clone()
	if !toNoAssets.Amount.Equal(clonedNoAssets.Amount) { //|| toNoAssets.Address.String() != clonedNoAssets.Address.String() || &toNoAssets == &clonedNoAssets {
		t.Error("Error while cloning og:", toNoAssets.Amount.String(), "cloned:", clonedNoAssets.Amount.String())
	}
	if !toWithAssets.Amount.Equal(clonedWithAssets.Amount) || toWithAssets.Address.String() != clonedWithAssets.Address.String() || &toWithAssets == &clonedWithAssets {
		t.Error("Error while cloning with assets")
	}
	if toNoAssets.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh:1000000 DATUM: " {
		t.Error("Error while stringifying, got", toNoAssets.String())
	}
}

var addr, _ = Address.DecodeAddress(TEST_ADDRESS)
var amNoAssets = Value.PureLovelaceValue(1000000)
var amWithAssets = Value.SimpleValue(1_000_000, MultiAsset.MultiAsset[int64]{
	Policy.PolicyId{TEST_POLICY}: Asset.Asset[int64]{
		AssetName.NewAssetNameFromString("Token1"): 1,
	},
})

var toAlonzoNoAssets = TransactionOutput.TransactionOutput{
	IsPostAlonzo: true,
	PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
		Address: addr,
		Amount:  amNoAssets.ToAlonzoValue(),
	},
}
var toAlonzoWithAssets = TransactionOutput.TransactionOutput{
	IsPostAlonzo: true,
	PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
		Address: addr,
		Amount:  amWithAssets.ToAlonzoValue(),
	},
}
var toShelleyNoAssets = TransactionOutput.TransactionOutput{
	IsPostAlonzo: false,
	PreAlonzo: TransactionOutput.TransactionOutputShelley{
		Address: addr,
		Amount:  amNoAssets,
	},
}
var toShelleyWithAssets = TransactionOutput.TransactionOutput{
	IsPostAlonzo: false,
	PreAlonzo: TransactionOutput.TransactionOutputShelley{
		Address: addr,
		Amount:  amWithAssets,
	},
}

func TestTxoClone(t *testing.T) {
	cloned := toAlonzoNoAssets.Clone()
	if !cloned.EqualTo(toAlonzoNoAssets) {
		t.Error("Error while cloning")
	}
	cloned = toAlonzoWithAssets.Clone()
	if !cloned.EqualTo(toAlonzoWithAssets) {
		t.Error("Error while cloning")
	}
	cloned = toShelleyNoAssets.Clone()
	if !cloned.EqualTo(toShelleyNoAssets) {
		t.Error("Error while cloning")
	}
	cloned = toShelleyWithAssets.Clone()
	if !cloned.EqualTo(toShelleyWithAssets) {
		t.Error("Error while cloning")
	}
}

func TestEqualTo(t *testing.T) {
	if !toAlonzoNoAssets.EqualTo(toAlonzoNoAssets) {
		t.Error("Error while comparing")
	}
	if !toAlonzoWithAssets.EqualTo(toAlonzoWithAssets) {
		t.Error("Error while comparing")
	}
	if !toShelleyNoAssets.EqualTo(toShelleyNoAssets) {
		t.Error("Error while comparing")
	}
	if !toShelleyWithAssets.EqualTo(toShelleyWithAssets) {
		t.Error("Error while comparing")
	}
	if toAlonzoNoAssets.EqualTo(toAlonzoWithAssets) {
		t.Error("Error while comparing")
	}
	if toAlonzoWithAssets.EqualTo(toAlonzoNoAssets) {
		t.Error("Error while comparing")
	}
	if toShelleyNoAssets.EqualTo(toShelleyWithAssets) {
		t.Error("Error while comparing")
	}
	if toShelleyWithAssets.EqualTo(toShelleyNoAssets) {
		t.Error("Error while comparing")
	}
	if toAlonzoNoAssets.EqualTo(toShelleyNoAssets) {
		t.Error("Error while comparing")
	}
	if toAlonzoWithAssets.EqualTo(toShelleyWithAssets) {
		t.Error("Error while comparing")
	}
	if toShelleyNoAssets.EqualTo(toAlonzoNoAssets) {
		t.Error("Error while comparing")
	}
	if toShelleyWithAssets.EqualTo(toAlonzoWithAssets) {
		t.Error("Error while comparing")
	}
}

func TestGetAmount(t *testing.T) {
	if !toAlonzoNoAssets.GetAmount().Equal(amNoAssets) {
		t.Error("Error while getting amount")
	}
	if !toAlonzoWithAssets.GetAmount().Equal(amWithAssets) {
		t.Error("Error while getting amount")
	}
	if !toShelleyNoAssets.GetAmount().Equal(amNoAssets) {
		t.Error("Error while getting amount")
	}
	if !toShelleyWithAssets.GetAmount().Equal(amWithAssets) {
		t.Error("Error while getting amount")
	}
}

func TestSimpleTxOBuilder(t *testing.T) {
	stxo := TransactionOutput.SimpleTransactionOutput(addr, amNoAssets)
	if !stxo.EqualTo(toShelleyNoAssets) {
		t.Error("Error while building")
	}
	stxo = TransactionOutput.SimpleTransactionOutput(addr, amWithAssets)
	if !stxo.EqualTo(toShelleyWithAssets) {
		t.Error("Error while building")
	}
}

func TestGetAddress(t *testing.T) {
	if toAlonzoNoAssets.GetAddress().String() != addr.String() {
		t.Error("Error while getting address")
	}
	if toShelleyNoAssets.GetAddress().String() != addr.String() {
		t.Error("Error while getting address")
	}
}

func GetAddressPointer(t *testing.T) {
	if toAlonzoNoAssets.GetAddressPointer().String() != addr.String() {
		t.Error("Error while getting address")
	}
	if toShelleyNoAssets.GetAddressPointer().String() != addr.String() {
		t.Error("Error while getting address")
	}
}

func TestGetValue(t *testing.T) {
	if !toAlonzoNoAssets.GetValue().Equal(amNoAssets) {
		t.Error("Error while getting value")
	}
	if !toShelleyNoAssets.GetValue().Equal(amNoAssets) {
		t.Error("Error while getting value")
	}
	if !toAlonzoWithAssets.GetValue().Equal(amWithAssets) {
		t.Error("Error while getting value")
	}
	if !toShelleyWithAssets.GetValue().Equal(amWithAssets) {
		t.Error("Error while getting value")
	}
}

func TestGetLovelace(t *testing.T) {
	if toAlonzoNoAssets.Lovelace() != amNoAssets.GetCoin() {
		t.Error("Error while getting lovelace")
	}
	if toShelleyNoAssets.Lovelace() != amNoAssets.GetCoin() {
		t.Error("Error while getting lovelace")
	}
	if toAlonzoWithAssets.Lovelace() != amWithAssets.GetCoin() {
		t.Error("Error while getting lovelace")
	}
	if toShelleyWithAssets.Lovelace() != amWithAssets.GetCoin() {
		t.Error("Error while getting lovelace")
	}
}

func TestString(t *testing.T) {
	if toAlonzoNoAssets.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh:1000000 Datum :<nil>" {
		t.Error("Error while stringifying", toAlonzoNoAssets.String())
	}
	if toAlonzoWithAssets.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh:{{} 1000000 map[115a3b670ea8b6b99d1c3d1d8041d7da9bd0b45532c24481cdbd9818:map[Token1:1]]} Datum :<nil>" {
		t.Error("Error while stringifying", toAlonzoWithAssets.String())
	}
}

func TestSetAmount(t *testing.T) {
	clonedNoAssetsAlonzo := toAlonzoNoAssets.Clone()
	clonedNoAssetsShelley := toShelleyNoAssets.Clone()
	clonedNoAssetsAlonzo.SetAmount(amWithAssets)
	clonedNoAssetsShelley.SetAmount(amWithAssets)
	if !clonedNoAssetsAlonzo.EqualTo(toAlonzoWithAssets) {
		t.Error("Error while setting amount")
	}
	if !clonedNoAssetsShelley.EqualTo(toShelleyWithAssets) {
		t.Error("Error while setting amount")
	}
	if clonedNoAssetsAlonzo.GetAmount().Equal(amNoAssets) {
		t.Error("Error while setting amount")
	}
	if clonedNoAssetsShelley.GetAmount().Equal(amNoAssets) {
		t.Error("Error while setting amount")
	}
	if !clonedNoAssetsAlonzo.GetAmount().Equal(amWithAssets) {
		t.Error("Error while setting amount")
	}
	if !clonedNoAssetsShelley.GetAmount().Equal(amWithAssets) {
		t.Error("Error while setting amount")
	}
}

var sampleDatum = PlutusData.PlutusData{
	TagNr:          0,
	PlutusDataType: PlutusData.PlutusBytes,
	Value:          []byte{0x01, 0x02, 0x03},
}

func TestSetDatum(t *testing.T) {
	clonedAlonzo := toAlonzoNoAssets.Clone()
	clonedShelley := toShelleyNoAssets.Clone()
	clonedAlonzo.SetDatum(&sampleDatum)
	clonedShelley.SetDatum(&sampleDatum)
	if clonedAlonzo.GetDatum() == nil {
		t.Error("Error while setting datum")
	}
	if clonedShelley.GetDatum() != nil {
		t.Error("Error while setting datum")
	}
	if clonedAlonzo.GetDatumHash() != nil {
		t.Error("Error while setting datum")
	}
	if clonedShelley.GetDatumHash() == nil {

	}
}

func TestSetScript(t *testing.T) {
	ScriptRefBytes := []byte{0x01, 0x02, 0x03}
	clonedAlonzo := toAlonzoNoAssets.Clone()
	ScriptRef := PlutusData.NewScriptRef(ScriptRefBytes, 2)
	clonedAlonzo.PostAlonzo.ScriptRef = &ScriptRef
	cborred, _ := cbor.Marshal(clonedAlonzo)
	if hex.EncodeToString(cborred) != "a300583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4011a000f424003d81846820243010203" {
		t.Error("Error while setting script", hex.EncodeToString(cborred))
	}
}
