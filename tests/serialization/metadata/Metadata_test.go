package metadata_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/cbor/v2"
	"github.com/SundaeSwap-finance/apollo/serialization/Metadata"
)

func TestAuxiliaryData(t *testing.T) {
	aux := Metadata.AuxiliaryData{}
	alonzoMeta := Metadata.ShelleyMaryMetadata{}

	meta := Metadata.Metadata{
		721: Metadata.TagMetadata{"{policy_id}": map[string]map[string]string{
			"{policy_name}": {
				"description": "<optional>",
				"name":        "<required>",
				"image":       "<required>",
				"type":        "<optional>",
			}},
		},
	}
	alonzoMeta.Metadata = meta
	aux.SetShelleyMetadata(alonzoMeta)
	marshaled, err := cbor.Marshal(aux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(marshaled) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf("InvalidReserialization got %s expected %s", hex.EncodeToString(marshaled), `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`)
	}
	if len(aux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(aux.Hash()) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf("Invalid Hashing Of AuxiliaryData expected %s got %s", "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def", hex.EncodeToString(aux.Hash()))
	}
}
