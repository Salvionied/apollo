package Transaction_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/fxamacker/cbor/v2"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	tx := Transaction.Transaction{
		TransactionBody: TransactionBody.TransactionBody{
			Inputs: []TransactionInput.TransactionInput{
				{
					TransactionId: []byte{0x01, 0x02, 0x03},
					Index:         0,
				},
			},
		},
		TransactionWitnessSet: TransactionWitnessSet.TransactionWitnessSet{},
		Valid:                 false,
		AuxiliaryData:         nil,
	}

	marshaled, _ := tx.Bytes()
	if hex.EncodeToString(marshaled) != "84a4008182430102030001f6020009a0a0f4f6" {
		t.Error(
			"Invalid marshaling",
			hex.EncodeToString(marshaled),
			"Expected",
			"84a4008182430102030001f6020009a0a0f4f6",
		)
	}
	tx2 := Transaction.Transaction{}
	err := cbor.Unmarshal(marshaled, &tx2)
	if err != nil {
		t.Error("Unmarshal failed", err)
	}
	if tx2.TransactionBody.Inputs[0].Index != 0 {
		t.Error(
			"Invalid unmarshaling",
			tx2.TransactionBody.Inputs[0].Index,
			"Expected",
			0,
		)
	}
	if tx2.TransactionBody.Inputs[0].TransactionId[0] != 0x01 {
		t.Error(
			"Invalid unmarshaling",
			tx2.TransactionBody.Inputs[0].TransactionId[0],
			"Expected",
			0x01,
		)
	}
}

func TestId(t *testing.T) {
	tx := Transaction.Transaction{
		TransactionBody: TransactionBody.TransactionBody{
			Inputs: []TransactionInput.TransactionInput{
				{
					TransactionId: []byte{0x01, 0x02, 0x03},
					Index:         0,
				},
			},
		},
		TransactionWitnessSet: TransactionWitnessSet.TransactionWitnessSet{},
		Valid:                 false,
		AuxiliaryData:         nil,
	}
	txId := tx.Id()
	if hex.EncodeToString(
		txId.Payload,
	) != "2d1312c2950d08c5fe35b8d1f293d13e0cf85e51a1c1779ee05b89838cf4e771" {
		t.Error(
			"Invalid transaction ID",
			hex.EncodeToString(txId.Payload),
			"Expected",
			"2d1312c2950d08c5fe35b8d1f293d13e0cf85e51a1c1779ee05b89838cf4e771",
		)
	}
}

// Conway transaction hex featuring tag 258 for input sets and post-Babbage
// transaction body fields.
const conwayTxHex = "84aa00d90102828258209461ba61e31f9adfa12cc294ec16bb776c859894e09d64c686ece4a848ae73be00825820b474d3753980e74cf9e9f00dc3acd91c385a6f3fd461f76cbd903d5a252d7988000183a300581d701edfb25decf8b4f59462c2c2a517f3bbfe896d00772a856836fbd70501821a00140ef6a1581cbda48796e4c9e16d0350c8e42d3ea6682ee084d9ffff47b50d20e7f1a14768616e646c657201028201d8185839d8799fd8799f0100009f1864ffffd8799f581cbda48796e4c9e16d0350c8e42d3ea6682ee084d9ffff47b50d20e7f14768616e646c6572ffffa300581d70953ac34e55c92445e295b175365ebb872abc14f1c468b7b950c8d8c001821a00251778a1581c911db1dbde441b8e874dd7cd6ee14ce6abd3544edbc027c79c5dde30a1581986358ebc10ae36be23a08c95066da9baa9d47a5af2c9db643001028201d818590128d8799fd8799fd8799f4c76657373656c6f7261636c65d8799f0103ff1b000075ded9f680001b0006722feb7b00001b0000008bb2c97000d8799f0000ffd8799f001912c7ff9fd8799fd8799f010001014100ffd8799f9f0001ff1821040c4001ff0000d87980ffd8799fd8799f010001014100ffd8799f9f0001ff182001014001ff0000d87980ffffffbfd8799f001912c7ffd8799f1b181c621cdfe371405820574fdf2d671ef3f4950ef9895eac248db07f449973197a1ee911c1ee0b13c16fd8799f5820f29f789e9c63a3b1399f4403a13e3e97badf7cfb6573b707f9877fac244f4433ffffffffd8799f581c911db1dbde441b8e874dd7cd6ee14ce6abd3544edbc027c79c5dde30581986358ebc10ae36be23a08c95066da9baa9d47a5af2c9db6430ffff82581d60247570b8ba7dc725e9ff37e9757b8148b4d5a125958edac2fd4417b81b00000006fbf42b21021a000a69670319186509a1581c911db1dbde441b8e874dd7cd6ee14ce6abd3544edbc027c79c5dde30a1581986358ebc10ae36be23a08c95066da9baa9d47a5af2c9db6430010b58201cba6090eb3fd2dd042360b2c9fa0c9e4cbffca7be4d03351ad1dfe833e1a90a0dd90102818258209461ba61e31f9adfa12cc294ec16bb776c859894e09d64c686ece4a848ae73be001082581d60247570b8ba7dc725e9ff37e9757b8148b4d5a125958edac2fd4417b81b00000006fbd760c0111a004c4b4012d9010282825820dabe77056214d9a6409991715c8ab51d379fd23609d27a944f1d4fc51d463f9600825820d13cba4c4e0f9cb2622628e85249d157ab90fbdbaf0482c23667ea46167481e300a10582840001d87980821a0003ab0f1a04f9d4bf840100d8799fd8799f581cbda48796e4c9e16d0350c8e42d3ea6682ee084d9ffff47b50d20e7f14768616e646c6572ffff821a000b56061a0d8011c1f5f6"

func TestConwayTransactionDeserialization(t *testing.T) {
	cborBytes, err := hex.DecodeString(conwayTxHex)
	if err != nil {
		t.Fatalf("Hex decode error: %v", err)
	}

	tx := Transaction.Transaction{}
	err = cbor.Unmarshal(cborBytes, &tx)
	if err != nil {
		t.Fatalf("CBOR unmarshal error: %v", err)
	}

	if tx.TransactionBody.Fee != 682343 {
		t.Errorf("Expected fee 682343, got %d", tx.TransactionBody.Fee)
	}

	if len(tx.TransactionBody.Inputs) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(tx.TransactionBody.Inputs))
	}

	if len(tx.TransactionBody.Outputs) != 3 {
		t.Errorf("Expected 3 outputs, got %d", len(tx.TransactionBody.Outputs))
	}
}

func TestConwayTransactionInputSetWithTag258(t *testing.T) {
	// Test decoding of CBOR tag 258 wrapped input array (Conway format).
	// d90102 = tag 258
	// 82 = array(2)
	// 825820...00 = [txid, 0]
	// 825820...00 = [txid, 0]
	inputsHex := "d90102828258209461ba61e31f9adfa12cc294ec16bb776c859894e09d64c686ece4a848ae73be00825820b474d3753980e74cf9e9f00dc3acd91c385a6f3fd461f76cbd903d5a252d798800"

	inputsBytes, err := hex.DecodeString(inputsHex)
	if err != nil {
		t.Fatalf("Hex decode error: %v", err)
	}

	var inputSet TransactionBody.TransactionInputSet
	err = inputSet.UnmarshalCBOR(inputsBytes)
	if err != nil {
		t.Fatalf("InputSet decode error: %v", err)
	}

	if len(inputSet.Items()) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(inputSet.Items()))
	}
}

func TestConwayTransactionBodyDeserialization(t *testing.T) {
	cborBytes, err := hex.DecodeString(conwayTxHex)
	if err != nil {
		t.Fatalf("Hex decode error: %v", err)
	}

	var txArray []cbor.RawMessage
	err = cbor.Unmarshal(cborBytes, &txArray)
	if err != nil {
		t.Fatalf("Transaction array decode error: %v", err)
	}

	var body TransactionBody.TransactionBody
	err = body.UnmarshalCBOR(txArray[0])
	if err != nil {
		t.Fatalf("Body decode error: %v", err)
	}

	if body.Fee != 682343 {
		t.Errorf("Expected fee 682343, got %d", body.Fee)
	}

	if len(body.Inputs) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(body.Inputs))
	}
}
