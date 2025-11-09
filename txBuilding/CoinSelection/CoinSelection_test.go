package CoinSelection_test

var TESTADDRESS = "addr_test1vrm9x2zsux7va6w892g38tvchnzahvcd9tykqf3ygnmwtaqyfg52x"

// func AssertRequestFulfilled(request []TransactionOutput.TransactionOutput, selected []UTxO.UTxO) bool {
// 	//TODO IMPLEMENT
// 	return true
// }
// func TestLargestFirstAdaOnly(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()

// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(15_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, true, true)
// 	if len(selected) != 2 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(3_999_900) {
// 		t.Errorf("Expected change to be 500_000, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestLargestFirstRequestOutputs(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()
// 	//ONlY ADA TEST
// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(9_000_000)),
// 		TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(6_000_000))}

// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, true, true)
// 	if len(selected) != 2 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(3_999_900) {
// 		t.Errorf("Expected change to be 3_000_900, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestFeeEffectLargestFirst(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()
// 	//ONlY ADA TEST
// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(10_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, true, false)
// 	if len(selected) != 2 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(8_999_900) {
// 		t.Errorf("Expected change to be 500_000, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestNoFeeEffectLargestFirst(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()
// 	//ONlY ADA TEST
// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(10_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, false, false)
// 	if len(selected) != 1 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(0) {
// 		t.Errorf("Expected change to be 0, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestInsufficientBalance(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()
// 	//ONlY ADA TEST
// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(1_000_000_000))}
// 	_, _, err := selector.Select(utxos, request, chain_context, -1, false, false)
// 	if err == nil {
// 		t.Errorf("Expected error, got nil")
// 	}
// }

// func TestMaxInputCountLargestFirst(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()
// 	//ONlY ADA TEST
// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address, Value.PureLovelaceValue(15000000))}
// 	_, _, err := selector.Select(utxos, request, chain_context, 1, false, false)
// 	if err == nil {
// 		t.Errorf("Expected error, got nil")
// 	}
// }

// func TestMultiAsset(t *testing.T) {
// 	chain_context := FixedChainContext.InitFixedChainContext()
// 	decoded_address, _ := Address.DecodeAddress(TESTADDRESS)
// 	selector := CoinSelection.LargestFirstSelector{}
// 	utxos := testutils.InitUtxos()
// 	//ONlY ADA TEST
// 	request := []TransactionOutput.TransactionOutput{TransactionOutput.SimpleTransactionOutput(decoded_address,
// 		Value.SimpleValue(15000000, MultiAsset.MultiAsset[int64]{
// 			Policy.PolicyId{Value: "00000000000000000000000000000000000000000000000000000000"}: Asset.Asset[int64]{AssetName.NewAssetNameFromString("token0"): int64(50)},
// 		}))}
// 	selected, change, err := selector.Select(utxos, request, chain_context, -1, false, false)
// 	if err != nil {
// 		t.Errorf("Expected no error, got %s", err)
// 	}
// 	if len(selected) != 10 {
// 		t.Errorf("Expected 10 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != 40_000_000 {
// 		t.Errorf("Expected change to be 40_000_000, got %d", change.GetCoin())
// 	}

// }

// func TestRandomImproveAdaOnly(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(15_000_000))}

// 	selected, change, err := selector.Select(utxos, request, chain_context, -1, true, true)
// 	if err != nil {
// 		t.Errorf("Expected no error, got %s", err)
// 	}
// 	if len(selected) != 4 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(4_999_900) {
// 		t.Errorf("Expected change to be 7000000, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// 	utxos = initUtxos()
// 	request = []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(6_000_000)),
// 		serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(9_000_000))}
// 	selected, change, _ = selector.Select(utxos, request, chain_context, -1, true, true)
// 	if len(selected) != 3 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(3_999_900) {
// 		t.Errorf("Expected change to be 1000000, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestRandomImproveFeeEffect(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(9_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, true, true)
// 	if len(selected) != 3 {
// 		t.Errorf("Expected 3 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(6999900) {
// 		t.Errorf("Expected change to be 0, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestRandomImproveNoFeeEffect(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(9_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, false, false)
// 	if len(selected) != 2 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(1000000) {
// 		t.Errorf("Expected change to be 0, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestRandomImproveNoFeeButMin(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(5_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, false, true)
// 	if len(selected) != 2 {
// 		t.Errorf("Expected 3 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(5_000_000) {
// 		t.Errorf("Expected change to be 7000_000, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestRandomImproveUtxoDepleted(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(100_000_000))}
// 	selected, change, _ := selector.Select(utxos, request, chain_context, -1, false, false)
// 	if len(selected) != 0 {
// 		t.Errorf("Expected 0 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(0) {
// 		t.Errorf("Expected change to be 0, got %d", change.GetCoin())
// 	}
// 	// if AssertRequestFulfilled(request, selected) {
// 	// 	t.Errorf("Expected request to be fulfilled")
// 	// }
// }

// func TestRandomImproveMaxInput(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{serialization.SimpleTransactionOutput(decoded_address, serialization.PureLovelaceValue(9_000_000))}
// 	selected, change, err := selector.Select(utxos, request, chain_context, 1, false, false)
// 	if err == nil {
// 		t.Errorf("Expected error to be returned")
// 	}
// 	if len(selected) != 0 {
// 		t.Errorf("Expected 2 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(0) {
// 		t.Errorf("Expected change to be 0, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }

// func TestRandomImproveMultiAsset(t *testing.T) {
// 	chain_context := backend.InitFixedChainContext()
// 	decoded_address, _ := serialization.DecodeAddress(Address)
// 	selector := builder.RandomImproveMultiAsset{}
// 	utxos := initUtxos()
// 	//ONlY ADA TEST
// 	request := []serialization.TransactionOutput{
// 		serialization.SimpleTransactionOutput(
// 			decoded_address,
// 			serialization.SimpleValue(
// 				15000000,
// 				serialization.MultiAsset[int64]{
// 					serialization.PolicyId{
// 						Value: "00000000000000000000000000000000000000000000000000000000",
// 					}: serialization.Asset[int64]{
// 						serialization.NewAssetNameFromString("token0"): int64(50),
// 						serialization.NewAssetNameFromString("token3"): int64(50),
// 					},
// 				},
// 			))}
// 	selected, change, err := selector.Select(utxos, request, chain_context, -1, true, true)
// 	if err != nil {
// 		t.Errorf("Expected no error to be returned, got %s", err.Error())
// 	}
// 	if len(selected) != 3 {
// 		t.Errorf("Expected 3 utxos to be selected, got %d", len(selected))
// 	}
// 	if change.GetCoin() != int64(39999900) {
// 		t.Errorf("Expected change to be 0, got %d", change.GetCoin())
// 	}
// 	if !AssertRequestFulfilled(request, selected) {
// 		t.Errorf("Expected request to be fulfilled")
// 	}
// }
