package Policy_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/cbor/v2"
)

const SAMPLE_POLICY = "95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb00"
const INV_LEN_SAMPLE_POLICY = "95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb0095"

func TestPolicyMarshalingUnmarshaling(t *testing.T) {
	p := Policy.PolicyId{Value: SAMPLE_POLICY}
	marshaled, _ := cbor.Marshal(p)
	if hex.EncodeToString(marshaled) != "581c95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb00" {
		t.Error("Invalid marshaling", hex.EncodeToString(marshaled), "Expected", SAMPLE_POLICY)
	}
	var p2 Policy.PolicyId
	err := cbor.Unmarshal(marshaled, &p2)
	if err != nil {
		t.Error("Failed unmarshaling", err)
	}
	if p2.Value != SAMPLE_POLICY {
		t.Error("Invalid unmarshaling", p2.Value, "Expected", SAMPLE_POLICY)
	}
	if p.String() != SAMPLE_POLICY {
		t.Error("Invalid string representation", p.String(), "Expected", SAMPLE_POLICY)
	}
}

func TestHelperMethods(t *testing.T) {
	_, err := Policy.New(INV_LEN_SAMPLE_POLICY)
	if err == nil {
		t.Error("Invalid length not detected")
	}
	policy_bytes, _ := hex.DecodeString(INV_LEN_SAMPLE_POLICY)
	_, err = Policy.FromBytes(policy_bytes)
	if err == nil {
		t.Error("Invalid length not detected")
	}
}
