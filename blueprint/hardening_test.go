package blueprint

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/Salvionied/apollo/v2/constants"
	"github.com/blinklabs-io/plutigo/data"
)

func TestParseRejectsTrailingTokensAndOversizedInput(t *testing.T) {
	raw := []byte(`{
		"preamble": {"title": "t", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{"title": "v", "redeemer": {"schema": {"dataType": "integer"}}}]
	}`)
	if _, err := Parse(append(append([]byte{}, raw...), []byte(" {}")...)); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing tokens: got %v", err)
	}
	if _, err := Parse(append(raw, '\n', ' ', '\t')); err != nil {
		t.Fatalf("trailing whitespace should be accepted: %v", err)
	}
	oversized := bytes.Repeat([]byte{' '}, MaxBlueprintBytes+1)
	if _, err := Parse(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized parse: got %v", err)
	}
}

func TestGuardedRecursiveSchemaIsAccepted(t *testing.T) {
	const raw = `{
		"preamble": {"title": "tree", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "v",
			"redeemer": {"schema": {"$ref": "#/definitions/Node"}}
		}],
		"definitions": {
			"Node": {
				"title": "Node",
				"dataType": "constructor",
				"index": 0,
				"fields": [
					{"title": "value", "dataType": "integer"},
					{"title": "children", "dataType": "list", "items": {"$ref": "#/definitions/Node"}}
				]
			}
		}
	}`
	bp, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	type node struct {
		Value    int64
		Children []node
	}
	pd, err := bp.EncodeRedeemer("v", "", node{Value: 1, Children: []node{{Value: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	var out node
	if err := bp.DecodeRedeemer("v", "", pd, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value != 1 || len(out.Children) != 1 || out.Children[0].Value != 2 {
		t.Fatalf("round trip = %+v", out)
	}

	cloned := bp.Clone()
	cloned.Validators[0].Title = "changed"
	if bp.Validators[0].Title != "v" {
		t.Fatal("Clone shared validator state with the original")
	}
}

func TestPlutusDataDepthLimit(t *testing.T) {
	const raw = `{
		"preamble": {"title": "deep", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "v",
			"redeemer": {"schema": {"$ref": "#/definitions/Wrap"}}
		}],
		"definitions": {
			"Wrap": {
				"dataType": "constructor",
				"index": 0,
				"fields": [{"title": "inner", "$ref": "#/definitions/Wrap"}]
			}
		}
	}`
	bp, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	pd := data.NewInteger(big.NewInt(1))
	for range constants.MaxPlutusDataDepth + 5 {
		pd = data.NewConstr(0, pd)
	}
	val, err := bp.Validator("v")
	if err != nil {
		t.Fatal(err)
	}
	if err := bp.ValidateData(&val.Redeemer.Single.Schema, pd); err == nil || !strings.Contains(err.Error(), "maximum depth") {
		t.Fatalf("depth limit: got %v", err)
	}
}

func TestOpaqueDataAndValueAreCloned(t *testing.T) {
	const raw = `{
		"preamble": {"title": "opaque", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{
			"title": "v",
			"datum": {"schema": {"$ref": "#/definitions/Data"}},
			"redeemer": {"schema": {"dataType": "value"}}
		}],
		"definitions": {"Data": {"title": "Data"}}
	}`
	bp, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	original := data.NewInteger(big.NewInt(4)).(*data.Integer)
	encoded, err := bp.EncodeDatum("v", original)
	if err != nil {
		t.Fatal(err)
	}
	original.Inner.SetInt64(9)
	got, ok := encoded.(*data.Integer)
	if !ok || got.Inner.Int64() != 4 {
		t.Fatalf("encoded integer changed with the caller value: %#v", encoded)
	}

	var decoded data.PlutusData
	if err := bp.DecodeDatum("v", encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	decodedInt := decoded.(*data.Integer)
	decodedInt.Inner.SetInt64(3)
	if got.Inner.Int64() != 4 {
		t.Fatal("decoded opaque data aliases the encoded value")
	}
}

func TestDatumHashMatchesDatumHashMethod(t *testing.T) {
	bp := loadFixture(t, "spend_record.json")
	value := escrowDatum{
		Owner:  bytes.Repeat([]byte{0x11}, 28),
		Amount: 5,
		Memo:   optionNote{None: &struct{}{}},
		Steps:  []escrowNote{{Tag: []byte("s"), Weight: 1}},
	}
	datum, err := bp.EncodeDatumValue("escrow.spend", value)
	if err != nil {
		t.Fatal(err)
	}
	if len(datum.Cbor()) == 0 {
		t.Fatal("EncodeDatumValue did not record canonical datum CBOR")
	}
	want, err := bp.DatumHash("escrow.spend", value)
	if err != nil {
		t.Fatal(err)
	}
	if datum.Hash() != want {
		t.Fatalf("datum.Hash() = %s, DatumHash = %s", datum.Hash(), want)
	}
}
