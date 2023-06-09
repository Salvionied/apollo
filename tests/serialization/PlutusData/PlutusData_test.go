package plutusdata_test

// func TestScriptDataHash(t *testing.T) {
// 	unit := new(PlutusData.PlutusData)
// 	redeemer := new(Redeemer.Redeemer)
// 	redeemer_cbor := "840000d87a80821a002fe1831a34801c64"
// 	datum_cbor := "d8799f581c98aeebe3627161246d1ba4444460f683e22e2e8621d10ed16452871a9fd8799fd8799fd8799f581ccab82fc1490cffe87333ebaeacac82b87e365b391c84e559a89fb76affd8799fd8799fd8799f581ca395cdb0f405e410af1219f0bbce853fcc8a04af3bf6db95dd3c6be9ffffffffa140d8799f00a1401a00fbc520ffffd8799fd8799fd8799f581c70e60f3b5ea7153e0acc7a803e4401d44b8ed1bae1c7baaad1a62a72ffd8799fd8799fd8799f581c1e78aae7c90cc36d624f7b3bb6d86b52696dc84e490f343eba89005fffffffffa140d8799f00a1401a0064b540ffffd8799fd8799fd8799f581c98aeebe3627161246d1ba4444460f683e22e2e8621d10ed16452871affd8799fd8799fd8799f581cf604d5a94dbd068f1bce584c0c6101ab56e2efd1c0cb365ecb4fb3fdffffffffa140d8799f00a1401a124aec20ffffffff"
// 	decoded_redeemer, _ := hex.DecodeString(redeemer_cbor)
// 	decoded_datum, _ := hex.DecodeString(datum_cbor)
// 	cbor.Unmarshal(decoded_redeemer, redeemer)
// 	cbor.Unmarshal(decoded_datum, unit)
// 	marshaled_redeemer, _ := cbor.Marshal(redeemer)
// 	marshaled_datum, _ := cbor.Marshal(unit)
// 	if hex.EncodeToString(marshaled_redeemer) != redeemer_cbor {
// 		t.Error("Invalid redeemer marshaling", hex.EncodeToString(marshaled_redeemer), "Expected", redeemer_cbor)
// 	}
// 	if hex.EncodeToString(marshaled_datum) != datum_cbor {
// 		t.Error("Invalid datum marshaling", hex.EncodeToString(marshaled_datum), "Expected", datum_cbor)
// 	}
// 	data_hash := TxBuilder.ScriptDataHash(nil, nil, nil, []Redeemer.Redeemer{*redeemer}, map[string]PlutusData.PlutusData{"l": *unit})
// 	if hex.EncodeToString(data_hash.Payload) != "8e191588a382fbf52c48e8956cca6a13fd5069038cb55bba30a22c310e517888" {
// 		t.Error("Invalid data hash", hex.EncodeToString(data_hash.Payload), "Expected, 8e191588a382fbf52c48e8956cca6a13fd5069038cb55bba30a22c310e517888")
// 	}
// }

// func TestScriptDataHashDatumOnly(t *testing.T) {
// 	unit := PlutusData.PlutusData{PlutusData.PlutusArray, 121, []any{}}
// 	data_hash := TxBuilder.ScriptDataHash(nil, nil, nil, []Redeemer.Redeemer{}, map[string]PlutusData.PlutusData{"l": unit})
// 	if hex.EncodeToString(data_hash.Payload) != "2f50ea2546f8ce020ca45bfcf2abeb02ff18af2283466f888ae489184b3d2d39" {
// 		t.Error("Invalid data hash", hex.EncodeToString(data_hash.Payload), "Expected, 2f50ea2546f8ce020ca45bfcf2abeb02ff18af2283466f888ae489184b3d2d39")
// 	}
// }

// func TestScriptDataHashRedeemerOnlyOnly2(t *testing.T) {
// 	data_hash := TxBuilder.ScriptDataHash(nil, nil, nil, []Redeemer.Redeemer{}, map[string]PlutusData.PlutusData{})
// 	if hex.EncodeToString(data_hash.Payload) != "a88fe2947b8d45d1f8b798e52174202579ecf847b8f17038c7398103df2d27b0" {
// 		t.Error("Invalid data hash", hex.EncodeToString(data_hash.Payload), "Expected, a88fe2947b8d45d1f8b798e52174202579ecf847b8f17038c7398103df2d27b0")
// 	}
// }

// func TestCostModels(t *testing.T) {
// 	final_cm := map[serialization.CustomBytes]PlutusData.CM{{Value: "00"}: PlutusData.PLUTUSV1COSTMODEL}
// 	fmt.Println(final_cm)
// 	new_cbor, err := cbor.Marshal(final_cm)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(hex.EncodeToString(new_cbor))
// 	if hex.EncodeToString(new_cbor) != "a141005901d59f1a000302590001011a00060bc719026d00011a000249f01903e800011a000249f018201a0025cea81971f70419744d186419744d186419744d186419744d186419744d186419744d18641864186419744d18641a000249f018201a000249f018201a000249f018201a000249f01903e800011a000249f018201a000249f01903e800081a000242201a00067e2318760001011a000249f01903e800081a000249f01a0001b79818f7011a000249f0192710011a0002155e19052e011903e81a000249f01903e8011a000249f018201a000249f018201a000249f0182001011a000249f0011a000249f0041a000194af18f8011a000194af18f8011a0002377c190556011a0002bdea1901f1011a000249f018201a000249f018201a000249f018201a000249f018201a000249f018201a000249f018201a000242201a00067e23187600010119f04c192bd200011a000249f018201a000242201a00067e2318760001011a000242201a00067e2318760001011a0025cea81971f704001a000141bb041a000249f019138800011a000249f018201a000302590001011a000249f018201a000249f018201a000249f018201a000249f018201a000249f018201a000249f018201a000249f018201a00330da70101ff" {
// 		fmt.Println("THIS", hex.EncodeToString(new_cbor))
// 		t.Error("WRONG SERIALIZATIONOF THE COSTMODEL")
// 	}
// }

// func TestPlutusScript(t *testing.T) {
// 	plutusScript := PlutusData.PlutusV1Script([]byte("test_script"))
// 	hash := PlutusData.PlutusScriptHash(plutusScript)
// 	if hex.EncodeToString(hash[:]) != "36c198e1a9d05461945c1f1db2ffb927c2dfc26dd01b59ea93b678b2" {
// 		t.Error("Invalid script hash", hex.EncodeToString(hash[:]), "Expected, 36c198e1a9d05461945c1f1db2ffb927c2dfc26dd01b59ea93b678b2")
// 	}
// }
