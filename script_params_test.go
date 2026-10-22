package apollo

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/blinklabs-io/plutigo/syn"
)

// identityScriptHex is (program 1.1.0 (lam x x)) as a CBOR byte string
// wrapping its flat encoding, the form used by Aiken blueprints and by
// gouroboros Plutus script types.
const identityScriptHex = "46010100200101"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return b
}

// scriptFromUplc builds a CBOR-wrapped flat script from textual UPLC.
func scriptFromUplc(t *testing.T, src string) []byte {
	t.Helper()
	program, err := syn.Parse(src)
	if err != nil {
		t.Fatalf("parse UPLC: %v", err)
	}
	debruijn, err := syn.NameToDeBruijn(program)
	if err != nil {
		t.Fatalf("convert to De Bruijn: %v", err)
	}
	flat, err := syn.Encode(debruijn)
	if err != nil {
		t.Fatalf("flat encode: %v", err)
	}
	wrapped, err := cbor.Encode(flat)
	if err != nil {
		t.Fatalf("cbor wrap: %v", err)
	}
	return wrapped
}

func TestApplyParamsToScriptKnownVector(t *testing.T) {
	script := common.PlutusV3Script(mustHex(t, identityScriptHex))
	got, err := ApplyParamsToScript(script, data.NewInteger(big.NewInt(42)))
	if err != nil {
		t.Fatalf("ApplyParamsToScript: %v", err)
	}
	// [(lam x x) (con data (I 42))] at UPLC 1.1.0, flat encoded by hand.
	want := mustHex(t, "4c010100320014c102182a0001")
	if !bytes.Equal(got, want) {
		t.Fatalf("applied script = %x, want %x", []byte(got), want)
	}
}

// aikenParamsScriptHex is the compiledCode Aiken v1.1.24 generates for this
// parameterized PlutusV3 validator:
//
//	pub type Config {
//	  Config { owner: ByteArray, threshold: Int, tags: List<Int> }
//	  Disabled
//	}
//
//	validator params(owner: ByteArray, amount: Int, config: Config, extra: Data) {
//	  else(_ctx: Data) {
//	    when config is {
//	      Config { owner: o, threshold: t, tags } ->
//	        o == owner && t <= amount && tags != [] && extra != extra
//	      Disabled -> False
//	    }
//	  }
//	}
const aikenParamsScriptHex = "588b010100222229800aba2aba1aab9eaab9dab9a9bae0059bad0044888888966002b30013370e900018021baa00a899192cc004cdc79bae30080020048acc004cdc49bad30080010038acc006600266ebd30101800030083009001a50a5140151980099baf00b00ba50a51401514a0802a29410054528200a30080013005375401514a0801a29344d95900301"

// aikenAppliedScriptHex is the compiledCode produced by applying the
// parameters in TestApplyParamsToScriptMatchesAiken, one at a time, with
// `aiken blueprint apply` (Aiken v1.1.24).
const aikenAppliedScriptHex = "5901230101003333222229800aba2aba1aab9eaab9dab9a9bae0059bad0044888888966002b30013370e900018021baa00a899192cc004cdc79bae30080020048acc004cdc49bad30080010038acc006600266ebd30101800030083009001a50a5140151980099baf00b00ba50a51401514a0802a29410054528200a30080013005375401514a0801a29344d95900313010544deadbeef004c01053a000f423f004c0117d8799f42cafec2494000000000000000009f010203ffff004c0165a3018040d9050080d8668218c89f5f58400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000046000000000000ffffc3493fffffffffffffffff0001"

func TestApplyParamsToScriptMatchesAiken(t *testing.T) {
	big70 := new(big.Int).Lsh(big.NewInt(1), 70)
	params := []data.PlutusData{
		data.NewByteString(mustHex(t, "deadbeef")),
		data.NewInteger(big.NewInt(-1_000_000)),
		data.NewConstr(0,
			data.NewByteString(mustHex(t, "cafe")),
			data.NewInteger(big70),
			data.NewList(
				data.NewInteger(big.NewInt(1)),
				data.NewInteger(big.NewInt(2)),
				data.NewInteger(big.NewInt(3)),
			),
		),
		data.NewMap([][2]data.PlutusData{
			{data.NewInteger(big.NewInt(1)), data.NewList()},
			{data.NewByteString(nil), data.NewConstr(7)},
			{
				data.NewConstr(200, data.NewByteString(make([]byte, 70))),
				data.NewInteger(new(big.Int).Neg(big70)),
			},
		}),
	}
	script := common.PlutusV3Script(mustHex(t, aikenParamsScriptHex))
	got, err := ApplyParamsToScript(script, params...)
	if err != nil {
		t.Fatalf("ApplyParamsToScript: %v", err)
	}
	want := common.PlutusV3Script(mustHex(t, aikenAppliedScriptHex))
	if !bytes.Equal(got, want) {
		t.Fatalf("applied script = %x, want %x", []byte(got), []byte(want))
	}
	if got.Hash().String() != "326be20547d05261f3a22c051e55b69651dae73a4dbd57645936ca16" {
		t.Fatalf("applied script hash = %s", got.Hash().String())
	}
}

func TestApplyParamsToScriptMatchesParsedProgram(t *testing.T) {
	params := []data.PlutusData{
		data.NewByteString(mustHex(t, "deadbeef")),
		data.NewConstr(1, data.NewInteger(big.NewInt(-7)), data.NewList()),
		data.NewMap([][2]data.PlutusData{
			{data.NewInteger(big.NewInt(1)), data.NewByteString(nil)},
		}),
	}
	base := scriptFromUplc(t, "(program 1.0.0 (lam a (lam b (lam c a))))")
	want := common.PlutusV2Script(scriptFromUplc(t, `(program 1.0.0
		[[[(lam a (lam b (lam c a)))
			(con data (B #deadbeef))]
			(con data (Constr 1 [I -7, List []]))]
			(con data (Map [(I 1, B #)]))])`))

	got, err := ApplyParamsToScript(common.PlutusV2Script(base), params...)
	if err != nil {
		t.Fatalf("ApplyParamsToScript: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("applied script = %x, want %x", []byte(got), []byte(want))
	}
	if got.Hash() == common.PlutusV2Script(base).Hash() {
		t.Fatal("applying parameters did not change the script hash")
	}

	// Applying the parameters one at a time is equivalent to applying them
	// together.
	stepwise := common.PlutusV2Script(base)
	for _, p := range params {
		stepwise, err = ApplyParamsToScript(stepwise, p)
		if err != nil {
			t.Fatalf("stepwise ApplyParamsToScript: %v", err)
		}
	}
	if !bytes.Equal(stepwise, got) {
		t.Fatalf("stepwise result = %x, want %x", []byte(stepwise), []byte(got))
	}
}

func TestApplyParamsToScriptPlutusV1(t *testing.T) {
	base := scriptFromUplc(t, "(program 1.0.0 (lam a a))")
	want := scriptFromUplc(t, "(program 1.0.0 [(lam a a) (con data (I 0))])")
	got, err := ApplyParamsToScript(common.PlutusV1Script(base), data.NewInteger(big.NewInt(0)))
	if err != nil {
		t.Fatalf("ApplyParamsToScript: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("applied script = %x, want %x", []byte(got), want)
	}
}

func TestApplyParamsToScriptNoParamsReturnsCopy(t *testing.T) {
	script := common.PlutusV3Script(mustHex(t, identityScriptHex))
	got, err := ApplyParamsToScript(script)
	if err != nil {
		t.Fatalf("ApplyParamsToScript: %v", err)
	}
	if !bytes.Equal(got, script) {
		t.Fatalf("result = %x, want %x", []byte(got), []byte(script))
	}
	got[0] ^= 0xff
	if script[0] != mustHex(t, identityScriptHex)[0] {
		t.Fatal("result aliases the input script")
	}
}

func TestApplyParamsToScriptDoesNotMutateInput(t *testing.T) {
	original := mustHex(t, identityScriptHex)
	script := common.PlutusV3Script(bytes.Clone(original))
	if _, err := ApplyParamsToScript(script, data.NewInteger(big.NewInt(1))); err != nil {
		t.Fatalf("ApplyParamsToScript: %v", err)
	}
	if !bytes.Equal(script, original) {
		t.Fatalf("input mutated to %x", []byte(script))
	}
}

func TestApplyParamsToScriptRejectsInvalidInput(t *testing.T) {
	valid := mustHex(t, identityScriptHex)
	one := data.NewInteger(big.NewInt(1))
	var nilInteger *data.Integer
	tests := []struct {
		name    string
		script  []byte
		params  []data.PlutusData
		wantErr string
	}{
		{name: "empty script", script: nil, params: []data.PlutusData{one}, wantErr: "decode script CBOR"},
		{name: "not CBOR bytes", script: mustHex(t, "01"), params: []data.PlutusData{one}, wantErr: "decode script CBOR"},
		{name: "truncated CBOR", script: valid[:len(valid)-1], params: []data.PlutusData{one}, wantErr: "decode script CBOR"},
		{name: "trailing bytes", script: append(bytes.Clone(valid), 0x00), params: []data.PlutusData{one}, wantErr: "trailing bytes"},
		{name: "raw flat without CBOR wrapper", script: valid[1:], params: []data.PlutusData{one}, wantErr: "decode script CBOR"},
		{name: "malformed flat", script: mustHex(t, "43010100"), params: []data.PlutusData{one}, wantErr: "decode script program"},
		{name: "nil parameter", script: valid, params: []data.PlutusData{one, nil}, wantErr: "parameter 1 is nil"},
		{name: "typed nil parameter", script: valid, params: []data.PlutusData{nilInteger}, wantErr: "parameter 0 is nil"},
		{name: "nested nil parameter", script: valid, params: []data.PlutusData{data.NewList(one, nilInteger)}, wantErr: "parameter 0"},
		{name: "malformed script without parameters", script: mustHex(t, "01"), params: nil, wantErr: "decode script CBOR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ApplyParamsToScript(common.PlutusV3Script(tc.script), tc.params...)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
