package PlutusData

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/Salvionied/apollo/constants"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	apolloCbor "github.com/Salvionied/apollo/serialization/cbor"

	"github.com/fxamacker/cbor/v2"

	"golang.org/x/crypto/blake2b"
)

type DatumType byte

const (
	DatumTypeHash   DatumType = 0
	DatumTypeInline DatumType = 1
)

type DatumOption struct {
	_         struct{} `cbor:",toarray"`
	DatumType DatumType
	Hash      []byte
	Inline    *PlutusData
}

func (d *DatumOption) UnmarshalCBOR(b []byte) error {
	var cborDatumOption struct {
		_         struct{} `cbor:",toarray"`
		DatumType DatumType
		Content   cbor.RawMessage
	}
	err := cbor.Unmarshal(b, &cborDatumOption)
	if err != nil {
		return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", err)

	}
	switch cborDatumOption.DatumType {
	case DatumTypeInline:
		var cborDatumInline PlutusData
		errInline := cbor.Unmarshal(cborDatumOption.Content, &cborDatumInline)
		if errInline != nil {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", errInline)
		}
		if cborDatumInline.TagNr != 24 {
			return fmt.Errorf(
				"found DatumTypeInline but Tag was not 24: %v",
				cborDatumInline.TagNr,
			)
		}
		taggedBytes, valid := cborDatumInline.Value.([]byte)
		if !valid {
			return errors.New(
				"DatumOption: UnmarshalCBOR: found tag 24 but there wasn't a byte array",
			)
		}
		var inline PlutusData
		err = cbor.Unmarshal(taggedBytes, &inline)
		if err != nil {
			return err
		}
		d.DatumType = DatumTypeInline
		d.Inline = &inline
		return nil
	case DatumTypeHash:
		var cborDatumHash []byte
		errHash := cbor.Unmarshal(cborDatumOption.Content, &cborDatumHash)
		if errHash != nil {
			return errHash
		}
		d.DatumType = DatumTypeHash
		d.Hash = cborDatumHash
		return nil
	default:
		return fmt.Errorf("unknown tag: %v", cborDatumOption.DatumType)
	}

}

func DatumOptionHash(hash []byte) DatumOption {
	return DatumOption{
		DatumType: DatumTypeHash,
		Hash:      hash,
	}
}

func DatumOptionInline(pd *PlutusData) DatumOption {
	return DatumOption{
		DatumType: DatumTypeInline,
		Inline:    pd,
	}
}

func (d DatumOption) MarshalCBOR() ([]byte, error) {
	var format struct {
		_       struct{} `cbor:",toarray"`
		Tag     DatumType
		Content *PlutusData
	}
	switch d.DatumType {
	case DatumTypeHash:
		format.Tag = DatumTypeHash
		format.Content = &PlutusData{
			PlutusDataType: PlutusBytes,
			TagNr:          0,
			Value:          d.Hash,
		}
	case DatumTypeInline:
		format.Tag = DatumTypeInline
		bytes, err := cbor.Marshal(d.Inline)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal inline datum: %w", err)
		}
		format.Content = &PlutusData{
			PlutusDataType: PlutusBytes,
			TagNr:          24,
			Value:          bytes,
		}
	default:
		return nil, fmt.Errorf("invalid DatumOption: %v", d)
	}
	return cbor.Marshal(format)
}

type ScriptRef []byte

func (sr ScriptRef) Len() int {
	return len(sr)
}

// MarshalCBOR encodes ScriptRef as CBOR tag 24 wrapping a byte string,
// per the Cardano CDDL: script_ref = #6.24(bytes .cbor script)
func (sr ScriptRef) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(cbor.Tag{Number: 24, Content: []byte(sr)})
}

// UnmarshalCBOR decodes ScriptRef from CBOR tag 24 wrapping a byte string.
func (sr *ScriptRef) UnmarshalCBOR(data []byte) error {
	var tag cbor.Tag
	if err := cbor.Unmarshal(data, &tag); err != nil {
		return err
	}
	if tag.Number != 24 {
		return fmt.Errorf("expected CBOR tag 24, got %d", tag.Number)
	}
	content, ok := tag.Content.([]byte)
	if !ok {
		return errors.New("expected byte string content in CBOR tag 24")
	}
	*sr = ScriptRef(content)
	return nil
}

// NewV1ScriptRef creates a ScriptRef for a Plutus V1 script.
// The inner CBOR encoding is [1, script_bytes] per the Cardano CDDL.
func NewV1ScriptRef(script PlutusV1Script) (ScriptRef, error) {
	inner, err := cbor.Marshal([]interface{}{uint64(1), []byte(script)})
	if err != nil {
		return nil, err
	}
	return ScriptRef(inner), nil
}

// NewV2ScriptRef creates a ScriptRef for a Plutus V2 script.
// The inner CBOR encoding is [2, script_bytes] per the Cardano CDDL.
func NewV2ScriptRef(script PlutusV2Script) (ScriptRef, error) {
	inner, err := cbor.Marshal([]interface{}{uint64(2), []byte(script)})
	if err != nil {
		return nil, err
	}
	return ScriptRef(inner), nil
}

// NewV3ScriptRef creates a ScriptRef for a Plutus V3 script.
// The inner CBOR encoding is [3, script_bytes] per the Cardano CDDL.
func NewV3ScriptRef(script PlutusV3Script) (ScriptRef, error) {
	inner, err := cbor.Marshal([]interface{}{uint64(3), []byte(script)})
	if err != nil {
		return nil, err
	}
	return ScriptRef(inner), nil
}

type CostModels map[serialization.CustomBytes]CM

type CM map[string]int

/*
*

	MarshalCBOR encodes the CM into a CBOR-encoded byte slice, in which
	it serializes the map key alphabetically and encodes the respective values.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if marshaling fails.
*/
func (cm CM) MarshalCBOR() ([]byte, error) {
	res := make([]int, 0, len(cm))
	mk := make([]string, 0, len(cm))
	for k := range cm {
		mk = append(mk, k)
	}
	sort.Strings(mk)
	for _, v := range mk {
		res = append(res, cm[v])
	}
	partial, _ := cbor.Marshal(res)
	partial[1] = 0x9f
	partial = append(partial, 0xff)
	return cbor.Marshal(partial[1:])
}

var PLUTUSV1COSTMODEL = CM{
	"addInteger-cpu-arguments-intercept":                       100788,
	"addInteger-cpu-arguments-slope":                           420,
	"addInteger-memory-arguments-intercept":                    1,
	"addInteger-memory-arguments-slope":                        1,
	"appendByteString-cpu-arguments-intercept":                 1000,
	"appendByteString-cpu-arguments-slope":                     173,
	"appendByteString-memory-arguments-intercept":              0,
	"appendByteString-memory-arguments-slope":                  1,
	"appendString-cpu-arguments-intercept":                     1000,
	"appendString-cpu-arguments-slope":                         59957,
	"appendString-memory-arguments-intercept":                  4,
	"appendString-memory-arguments-slope":                      1,
	"bData-cpu-arguments":                                      11183,
	"bData-memory-arguments":                                   32,
	"blake2b_256-cpu-arguments-intercept":                      201305,
	"blake2b_256-cpu-arguments-slope":                          8356,
	"blake2b_256-memory-arguments":                             4,
	"cekApplyCost-exBudgetCPU":                                 16000,
	"cekApplyCost-exBudgetMemory":                              100,
	"cekBuiltinCost-exBudgetCPU":                               16000,
	"cekBuiltinCost-exBudgetMemory":                            100,
	"cekConstCost-exBudgetCPU":                                 16000,
	"cekConstCost-exBudgetMemory":                              100,
	"cekDelayCost-exBudgetCPU":                                 16000,
	"cekDelayCost-exBudgetMemory":                              100,
	"cekForceCost-exBudgetCPU":                                 16000,
	"cekForceCost-exBudgetMemory":                              100,
	"cekLamCost-exBudgetCPU":                                   16000,
	"cekLamCost-exBudgetMemory":                                100,
	"cekStartupCost-exBudgetCPU":                               100,
	"cekStartupCost-exBudgetMemory":                            100,
	"cekVarCost-exBudgetCPU":                                   16000,
	"cekVarCost-exBudgetMemory":                                100,
	"chooseData-cpu-arguments":                                 94375,
	"chooseData-memory-arguments":                              32,
	"chooseList-cpu-arguments":                                 132994,
	"chooseList-memory-arguments":                              32,
	"chooseUnit-cpu-arguments":                                 61462,
	"chooseUnit-memory-arguments":                              4,
	"consByteString-cpu-arguments-intercept":                   72010,
	"consByteString-cpu-arguments-slope":                       178,
	"consByteString-memory-arguments-intercept":                0,
	"consByteString-memory-arguments-slope":                    1,
	"constrData-cpu-arguments":                                 22151,
	"constrData-memory-arguments":                              32,
	"decodeUtf8-cpu-arguments-intercept":                       91189,
	"decodeUtf8-cpu-arguments-slope":                           769,
	"decodeUtf8-memory-arguments-intercept":                    4,
	"decodeUtf8-memory-arguments-slope":                        2,
	"divideInteger-cpu-arguments-constant":                     85848,
	"divideInteger-cpu-arguments-model-arguments-intercept":    228465,
	"divideInteger-cpu-arguments-model-arguments-slope":        122,
	"divideInteger-memory-arguments-intercept":                 0,
	"divideInteger-memory-arguments-minimum":                   1,
	"divideInteger-memory-arguments-slope":                     1,
	"encodeUtf8-cpu-arguments-intercept":                       1000,
	"encodeUtf8-cpu-arguments-slope":                           42921,
	"encodeUtf8-memory-arguments-intercept":                    4,
	"encodeUtf8-memory-arguments-slope":                        2,
	"equalsByteString-cpu-arguments-constant":                  24548,
	"equalsByteString-cpu-arguments-intercept":                 29498,
	"equalsByteString-cpu-arguments-slope":                     38,
	"equalsByteString-memory-arguments":                        1,
	"equalsData-cpu-arguments-intercept":                       898148,
	"equalsData-cpu-arguments-slope":                           27279,
	"equalsData-memory-arguments":                              1,
	"equalsInteger-cpu-arguments-intercept":                    51775,
	"equalsInteger-cpu-arguments-slope":                        558,
	"equalsInteger-memory-arguments":                           1,
	"equalsString-cpu-arguments-constant":                      39184,
	"equalsString-cpu-arguments-intercept":                     1000,
	"equalsString-cpu-arguments-slope":                         60594,
	"equalsString-memory-arguments":                            1,
	"fstPair-cpu-arguments":                                    141895,
	"fstPair-memory-arguments":                                 32,
	"headList-cpu-arguments":                                   83150,
	"headList-memory-arguments":                                32,
	"iData-cpu-arguments":                                      15299,
	"iData-memory-arguments":                                   32,
	"ifThenElse-cpu-arguments":                                 76049,
	"ifThenElse-memory-arguments":                              1,
	"indexByteString-cpu-arguments":                            13169,
	"indexByteString-memory-arguments":                         4,
	"lengthOfByteString-cpu-arguments":                         22100,
	"lengthOfByteString-memory-arguments":                      10,
	"lessThanByteString-cpu-arguments-intercept":               28999,
	"lessThanByteString-cpu-arguments-slope":                   74,
	"lessThanByteString-memory-arguments":                      1,
	"lessThanEqualsByteString-cpu-arguments-intercept":         28999,
	"lessThanEqualsByteString-cpu-arguments-slope":             74,
	"lessThanEqualsByteString-memory-arguments":                1,
	"lessThanEqualsInteger-cpu-arguments-intercept":            43285,
	"lessThanEqualsInteger-cpu-arguments-slope":                552,
	"lessThanEqualsInteger-memory-arguments":                   1,
	"lessThanInteger-cpu-arguments-intercept":                  44749,
	"lessThanInteger-cpu-arguments-slope":                      541,
	"lessThanInteger-memory-arguments":                         1,
	"listData-cpu-arguments":                                   33852,
	"listData-memory-arguments":                                32,
	"mapData-cpu-arguments":                                    68246,
	"mapData-memory-arguments":                                 32,
	"mkCons-cpu-arguments":                                     72362,
	"mkCons-memory-arguments":                                  32,
	"mkNilData-cpu-arguments":                                  7243,
	"mkNilData-memory-arguments":                               32,
	"mkNilPairData-cpu-arguments":                              7391,
	"mkNilPairData-memory-arguments":                           32,
	"mkPairData-cpu-arguments":                                 11546,
	"mkPairData-memory-arguments":                              32,
	"modInteger-cpu-arguments-constant":                        85848,
	"modInteger-cpu-arguments-model-arguments-intercept":       228465,
	"modInteger-cpu-arguments-model-arguments-slope":           122,
	"modInteger-memory-arguments-intercept":                    0,
	"modInteger-memory-arguments-minimum":                      1,
	"modInteger-memory-arguments-slope":                        1,
	"multiplyInteger-cpu-arguments-intercept":                  90434,
	"multiplyInteger-cpu-arguments-slope":                      519,
	"multiplyInteger-memory-arguments-intercept":               0,
	"multiplyInteger-memory-arguments-slope":                   1,
	"nullList-cpu-arguments":                                   74433,
	"nullList-memory-arguments":                                32,
	"quotientInteger-cpu-arguments-constant":                   85848,
	"quotientInteger-cpu-arguments-model-arguments-intercept":  228465,
	"quotientInteger-cpu-arguments-model-arguments-slope":      122,
	"quotientInteger-memory-arguments-intercept":               0,
	"quotientInteger-memory-arguments-minimum":                 1,
	"quotientInteger-memory-arguments-slope":                   1,
	"remainderInteger-cpu-arguments-constant":                  85848,
	"remainderInteger-cpu-arguments-model-arguments-intercept": 228465,
	"remainderInteger-cpu-arguments-model-arguments-slope":     122,
	"remainderInteger-memory-arguments-intercept":              0,
	"remainderInteger-memory-arguments-minimum":                1,
	"remainderInteger-memory-arguments-slope":                  1,
	"sha2_256-cpu-arguments-intercept":                         270652,
	"sha2_256-cpu-arguments-slope":                             22588,
	"sha2_256-memory-arguments":                                4,
	"sha3_256-cpu-arguments-intercept":                         1457325,
	"sha3_256-cpu-arguments-slope":                             64566,
	"sha3_256-memory-arguments":                                4,
	"sliceByteString-cpu-arguments-intercept":                  20467,
	"sliceByteString-cpu-arguments-slope":                      1,
	"sliceByteString-memory-arguments-intercept":               4,
	"sliceByteString-memory-arguments-slope":                   0,
	"sndPair-cpu-arguments":                                    141992,
	"sndPair-memory-arguments":                                 32,
	"subtractInteger-cpu-arguments-intercept":                  100788,
	"subtractInteger-cpu-arguments-slope":                      420,
	"subtractInteger-memory-arguments-intercept":               1,
	"subtractInteger-memory-arguments-slope":                   1,
	"tailList-cpu-arguments":                                   81663,
	"tailList-memory-arguments":                                32,
	"trace-cpu-arguments":                                      59498,
	"trace-memory-arguments":                                   32,
	"unBData-cpu-arguments":                                    20142,
	"unBData-memory-arguments":                                 32,
	"unConstrData-cpu-arguments":                               24588,
	"unConstrData-memory-arguments":                            32,
	"unIData-cpu-arguments":                                    20744,
	"unIData-memory-arguments":                                 32,
	"unListData-cpu-arguments":                                 25933,
	"unListData-memory-arguments":                              32,
	"unMapData-cpu-arguments":                                  24623,
	"unMapData-memory-arguments":                               32,
	"verifyEd25519Signature-cpu-arguments-intercept":           53384111,
	"verifyEd25519Signature-cpu-arguments-slope":               14333,
	"verifyEd25519Signature-memory-arguments":                  10,
}

type CostView map[string]int

type CostModelArray []int32

func (cma CostModelArray) MarshalCBOR() ([]byte, error) {
	res := []int32(cma)
	return cbor.Marshal(res)
}

var PLUTUSV3COSTMODEL = CostModelArray(
	[]int32{
		100788,
		420,
		1,
		1,
		1000,
		173,
		0,
		1,
		1000,
		59957,
		4,
		1,
		11183,
		32,
		201305,
		8356,
		4,
		16000,
		100,
		16000,
		100,
		16000,
		100,
		16000,
		100,
		16000,
		100,
		16000,
		100,
		100,
		100,
		16000,
		100,
		94375,
		32,
		132994,
		32,
		61462,
		4,
		72010,
		178,
		0,
		1,
		22151,
		32,
		91189,
		769,
		4,
		2,
		85848,
		123203,
		7305,
		-900,
		1716,
		549,
		57,
		85848,
		0,
		1,
		1,
		1000,
		42921,
		4,
		2,
		24548,
		29498,
		38,
		1,
		898148,
		27279,
		1,
		51775,
		558,
		1,
		39184,
		1000,
		60594,
		1,
		141895,
		32,
		83150,
		32,
		15299,
		32,
		76049,
		1,
		13169,
		4,
		22100,
		10,
		28999,
		74,
		1,
		28999,
		74,
		1,
		43285,
		552,
		1,
		44749,
		541,
		1,
		33852,
		32,
		68246,
		32,
		72362,
		32,
		7243,
		32,
		7391,
		32,
		11546,
		32,
		85848,
		123203,
		7305,
		-900,
		1716,
		549,
		57,
		85848,
		0,
		1,
		90434,
		519,
		0,
		1,
		74433,
		32,
		85848,
		123203,
		7305,
		-900,
		1716,
		549,
		57,
		85848,
		0,
		1,
		1,
		85848,
		123203,
		7305,
		-900,
		1716,
		549,
		57,
		85848,
		0,
		1,
		955506,
		213312,
		0,
		2,
		270652,
		22588,
		4,
		1457325,
		64566,
		4,
		20467,
		1,
		4,
		0,
		141992,
		32,
		100788,
		420,
		1,
		1,
		81663,
		32,
		59498,
		32,
		20142,
		32,
		24588,
		32,
		20744,
		32,
		25933,
		32,
		24623,
		32,
		43053543,
		10,
		53384111,
		14333,
		10,
		43574283,
		26308,
		10,
		16000,
		100,
		16000,
		100,
		962335,
		18,
		2780678,
		6,
		442008,
		1,
		52538055,
		3756,
		18,
		267929,
		18,
		76433006,
		8868,
		18,
		52948122,
		18,
		1995836,
		36,
		3227919,
		12,
		901022,
		1,
		166917843,
		4307,
		36,
		284546,
		36,
		158221314,
		26549,
		36,
		74698472,
		36,
		333849714,
		1,
		254006273,
		72,
		2174038,
		72,
		2261318,
		64571,
		4,
		207616,
		8310,
		4,
		1293828,
		28716,
		63,
		0,
		1,
		1006041,
		43623,
		251,
		0,
		1,
		100181,
		726,
		719,
		0,
		1,
		100181,
		726,
		719,
		0,
		1,
		100181,
		726,
		719,
		0,
		1,
		107878,
		680,
		0,
		1,
		95336,
		1,
		281145,
		18848,
		0,
		1,
		180194,
		159,
		1,
		1,
		158519,
		8942,
		0,
		1,
		159378,
		8813,
		0,
		1,
		107490,
		3298,
		1,
		106057,
		655,
		1,
		1964219,
		24520,
		3,
	},
)

/*
*

	MarshalCBOR encodes the CostView into a CBOR-encoded byte slice, in which
	it serializes the map key alphabetically and encodes the respective values.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if marshaling fails.
*/
func (cv CostView) MarshalCBOR() ([]byte, error) {
	res := make([]int, 0, len(cv))
	mk := make([]string, 0, len(cv))
	for k := range cv {
		mk = append(mk, k)
	}
	sort.Strings(mk)
	for _, v := range mk {
		res = append(res, cv[v])
	}
	return cbor.Marshal(res)

}

var PLUTUSV2COSTMODEL = CostView{
	"addInteger-cpu-arguments-intercept":                       100788,
	"addInteger-cpu-arguments-slope":                           420,
	"addInteger-memory-arguments-intercept":                    1,
	"addInteger-memory-arguments-slope":                        1,
	"appendByteString-cpu-arguments-intercept":                 1000,
	"appendByteString-cpu-arguments-slope":                     173,
	"appendByteString-memory-arguments-intercept":              0,
	"appendByteString-memory-arguments-slope":                  1,
	"appendString-cpu-arguments-intercept":                     1000,
	"appendString-cpu-arguments-slope":                         59957,
	"appendString-memory-arguments-intercept":                  4,
	"appendString-memory-arguments-slope":                      1,
	"bData-cpu-arguments":                                      11183,
	"bData-memory-arguments":                                   32,
	"blake2b_256-cpu-arguments-intercept":                      201305,
	"blake2b_256-cpu-arguments-slope":                          8356,
	"blake2b_256-memory-arguments":                             4,
	"cekApplyCost-exBudgetCPU":                                 16000,
	"cekApplyCost-exBudgetMemory":                              100,
	"cekBuiltinCost-exBudgetCPU":                               16000,
	"cekBuiltinCost-exBudgetMemory":                            100,
	"cekConstCost-exBudgetCPU":                                 16000,
	"cekConstCost-exBudgetMemory":                              100,
	"cekDelayCost-exBudgetCPU":                                 16000,
	"cekDelayCost-exBudgetMemory":                              100,
	"cekForceCost-exBudgetCPU":                                 16000,
	"cekForceCost-exBudgetMemory":                              100,
	"cekLamCost-exBudgetCPU":                                   16000,
	"cekLamCost-exBudgetMemory":                                100,
	"cekStartupCost-exBudgetCPU":                               100,
	"cekStartupCost-exBudgetMemory":                            100,
	"cekVarCost-exBudgetCPU":                                   16000,
	"cekVarCost-exBudgetMemory":                                100,
	"chooseData-cpu-arguments":                                 94375,
	"chooseData-memory-arguments":                              32,
	"chooseList-cpu-arguments":                                 132994,
	"chooseList-memory-arguments":                              32,
	"chooseUnit-cpu-arguments":                                 61462,
	"chooseUnit-memory-arguments":                              4,
	"consByteString-cpu-arguments-intercept":                   72010,
	"consByteString-cpu-arguments-slope":                       178,
	"consByteString-memory-arguments-intercept":                0,
	"consByteString-memory-arguments-slope":                    1,
	"constrData-cpu-arguments":                                 22151,
	"constrData-memory-arguments":                              32,
	"decodeUtf8-cpu-arguments-intercept":                       91189,
	"decodeUtf8-cpu-arguments-slope":                           769,
	"decodeUtf8-memory-arguments-intercept":                    4,
	"decodeUtf8-memory-arguments-slope":                        2,
	"divideInteger-cpu-arguments-constant":                     85848,
	"divideInteger-cpu-arguments-model-arguments-intercept":    228465,
	"divideInteger-cpu-arguments-model-arguments-slope":        122,
	"divideInteger-memory-arguments-intercept":                 0,
	"divideInteger-memory-arguments-minimum":                   1,
	"divideInteger-memory-arguments-slope":                     1,
	"encodeUtf8-cpu-arguments-intercept":                       1000,
	"encodeUtf8-cpu-arguments-slope":                           42921,
	"encodeUtf8-memory-arguments-intercept":                    4,
	"encodeUtf8-memory-arguments-slope":                        2,
	"equalsByteString-cpu-arguments-constant":                  24548,
	"equalsByteString-cpu-arguments-intercept":                 29498,
	"equalsByteString-cpu-arguments-slope":                     38,
	"equalsByteString-memory-arguments":                        1,
	"equalsData-cpu-arguments-intercept":                       898148,
	"equalsData-cpu-arguments-slope":                           27279,
	"equalsData-memory-arguments":                              1,
	"equalsInteger-cpu-arguments-intercept":                    51775,
	"equalsInteger-cpu-arguments-slope":                        558,
	"equalsInteger-memory-arguments":                           1,
	"equalsString-cpu-arguments-constant":                      39184,
	"equalsString-cpu-arguments-intercept":                     1000,
	"equalsString-cpu-arguments-slope":                         60594,
	"equalsString-memory-arguments":                            1,
	"fstPair-cpu-arguments":                                    141895,
	"fstPair-memory-arguments":                                 32,
	"headList-cpu-arguments":                                   83150,
	"headList-memory-arguments":                                32,
	"iData-cpu-arguments":                                      15299,
	"iData-memory-arguments":                                   32,
	"ifThenElse-cpu-arguments":                                 76049,
	"ifThenElse-memory-arguments":                              1,
	"indexByteString-cpu-arguments":                            13169,
	"indexByteString-memory-arguments":                         4,
	"lengthOfByteString-cpu-arguments":                         22100,
	"lengthOfByteString-memory-arguments":                      10,
	"lessThanByteString-cpu-arguments-intercept":               28999,
	"lessThanByteString-cpu-arguments-slope":                   74,
	"lessThanByteString-memory-arguments":                      1,
	"lessThanEqualsByteString-cpu-arguments-intercept":         28999,
	"lessThanEqualsByteString-cpu-arguments-slope":             74,
	"lessThanEqualsByteString-memory-arguments":                1,
	"lessThanEqualsInteger-cpu-arguments-intercept":            43285,
	"lessThanEqualsInteger-cpu-arguments-slope":                552,
	"lessThanEqualsInteger-memory-arguments":                   1,
	"lessThanInteger-cpu-arguments-intercept":                  44749,
	"lessThanInteger-cpu-arguments-slope":                      541,
	"lessThanInteger-memory-arguments":                         1,
	"listData-cpu-arguments":                                   33852,
	"listData-memory-arguments":                                32,
	"mapData-cpu-arguments":                                    68246,
	"mapData-memory-arguments":                                 32,
	"mkCons-cpu-arguments":                                     72362,
	"mkCons-memory-arguments":                                  32,
	"mkNilData-cpu-arguments":                                  7243,
	"mkNilData-memory-arguments":                               32,
	"mkNilPairData-cpu-arguments":                              7391,
	"mkNilPairData-memory-arguments":                           32,
	"mkPairData-cpu-arguments":                                 11546,
	"mkPairData-memory-arguments":                              32,
	"modInteger-cpu-arguments-constant":                        85848,
	"modInteger-cpu-arguments-model-arguments-intercept":       228465,
	"modInteger-cpu-arguments-model-arguments-slope":           122,
	"modInteger-memory-arguments-intercept":                    0,
	"modInteger-memory-arguments-minimum":                      1,
	"modInteger-memory-arguments-slope":                        1,
	"multiplyInteger-cpu-arguments-intercept":                  90434,
	"multiplyInteger-cpu-arguments-slope":                      519,
	"multiplyInteger-memory-arguments-intercept":               0,
	"multiplyInteger-memory-arguments-slope":                   1,
	"nullList-cpu-arguments":                                   74433,
	"nullList-memory-arguments":                                32,
	"quotientInteger-cpu-arguments-constant":                   85848,
	"quotientInteger-cpu-arguments-model-arguments-intercept":  228465,
	"quotientInteger-cpu-arguments-model-arguments-slope":      122,
	"quotientInteger-memory-arguments-intercept":               0,
	"quotientInteger-memory-arguments-minimum":                 1,
	"quotientInteger-memory-arguments-slope":                   1,
	"remainderInteger-cpu-arguments-constant":                  85848,
	"remainderInteger-cpu-arguments-model-arguments-intercept": 228465,
	"remainderInteger-cpu-arguments-model-arguments-slope":     122,
	"remainderInteger-memory-arguments-intercept":              0,
	"remainderInteger-memory-arguments-minimum":                1,
	"remainderInteger-memory-arguments-slope":                  1,
	"serialiseData-cpu-arguments-intercept":                    955506,
	"serialiseData-cpu-arguments-slope":                        213312,
	"serialiseData-memory-arguments-intercept":                 0,
	"serialiseData-memory-arguments-slope":                     2,
	"sha2_256-cpu-arguments-intercept":                         270652,
	"sha2_256-cpu-arguments-slope":                             22588,
	"sha2_256-memory-arguments":                                4,
	"sha3_256-cpu-arguments-intercept":                         1457325,
	"sha3_256-cpu-arguments-slope":                             64566,
	"sha3_256-memory-arguments":                                4,
	"sliceByteString-cpu-arguments-intercept":                  20467,
	"sliceByteString-cpu-arguments-slope":                      1,
	"sliceByteString-memory-arguments-intercept":               4,
	"sliceByteString-memory-arguments-slope":                   0,
	"sndPair-cpu-arguments":                                    141992,
	"sndPair-memory-arguments":                                 32,
	"subtractInteger-cpu-arguments-intercept":                  100788,
	"subtractInteger-cpu-arguments-slope":                      420,
	"subtractInteger-memory-arguments-intercept":               1,
	"subtractInteger-memory-arguments-slope":                   1,
	"tailList-cpu-arguments":                                   81663,
	"tailList-memory-arguments":                                32,
	"trace-cpu-arguments":                                      59498,
	"trace-memory-arguments":                                   32,
	"unBData-cpu-arguments":                                    20142,
	"unBData-memory-arguments":                                 32,
	"unConstrData-cpu-arguments":                               24588,
	"unConstrData-memory-arguments":                            32,
	"unIData-cpu-arguments":                                    20744,
	"unIData-memory-arguments":                                 32,
	"unListData-cpu-arguments":                                 25933,
	"unListData-memory-arguments":                              32,
	"unMapData-cpu-arguments":                                  24623,
	"unMapData-memory-arguments":                               32,
	"verifyEcdsaSecp256k1Signature-cpu-arguments":              43053543,
	"verifyEcdsaSecp256k1Signature-memory-arguments":           10,
	"verifyEd25519Signature-cpu-arguments-intercept":           53384111,
	"verifyEd25519Signature-cpu-arguments-slope":               14333,
	"verifyEd25519Signature-memory-arguments":                  10,
	"verifySchnorrSecp256k1Signature-cpu-arguments-intercept":  43574283,
	"verifySchnorrSecp256k1Signature-cpu-arguments-slope":      26308,
	"verifySchnorrSecp256k1Signature-memory-arguments":         10,
}

var COST_MODELSV2 = map[int]cbor.Marshaler{1: PLUTUSV2COSTMODEL}

var COST_MODELSV1 = map[serialization.CustomBytes]cbor.Marshaler{
	{Value: "00"}: PLUTUSV1COSTMODEL,
}

type PlutusType int

const (
	PlutusArray PlutusType = iota
	PlutusMap
	PlutusIntMap
	PlutusInt
	PlutusBigInt
	PlutusBytes
	PlutusShortArray
	PlutusGenericMap // For maps with complex keys (like constructor-tagged values)
)

// PlutusMapPair represents a key-value pair in a Plutus data map
// where both key and value can be any PlutusData type.
// This is needed for maps where keys are constructor-tagged values.
type PlutusMapPair struct {
	Key   PlutusData
	Value PlutusData
}

// PlutusMapPairs represents a Plutus data map as a slice of key-value pairs.
// This representation is used instead of a Go map when keys are complex
// types like constructor-tagged values.
type PlutusMapPairs []PlutusMapPair

// decodeMapValue attempts to decode CBOR map data using multiple strategies.
// It tries simpler map types first, falling back to PlutusMapPairs for complex keys.
// Returns the decoded PlutusType, value, and any error.
func decodeMapValue(data []byte) (PlutusType, any, error) {
	// Strategy 1: Try map[CustomBytes]PlutusData (handles most common cases)
	simpleMap := map[serialization.CustomBytes]PlutusData{}
	if err := cbor.Unmarshal(data, &simpleMap); err == nil {
		// Check if all keys are integers (PlutusIntMap) or all are bytes (PlutusMap)
		allInt := len(simpleMap) > 0
		for k := range simpleMap {
			if !k.IsInt() {
				allInt = false
				break
			}
		}
		if allInt {
			return PlutusIntMap, simpleMap, nil
		}
		return PlutusMap, simpleMap, nil
	}

	// Strategy 2: Try map[PlutusDataKey]PlutusData
	keyMap := map[PlutusDataKey]PlutusData{}
	if err := cbor.Unmarshal(data, &keyMap); err == nil {
		return PlutusMap, keyMap, nil
	}

	// Strategy 3: Use DecodeMapPairs for complex keys (like constructor tags)
	pairs, pairErr := apolloCbor.DecodeMapPairs(data)
	if pairErr != nil {
		return 0, nil, fmt.Errorf("map decode failed: %w", pairErr)
	}

	mapPairs := make(PlutusMapPairs, len(pairs))
	for i, pair := range pairs {
		if err := cbor.Unmarshal(pair.KeyRaw, &mapPairs[i].Key); err != nil {
			return 0, nil, fmt.Errorf("map key decode: %w", err)
		}
		if err := cbor.Unmarshal(pair.ValueRaw, &mapPairs[i].Value); err != nil {
			return 0, nil, fmt.Errorf("map value decode: %w", err)
		}
	}
	return PlutusGenericMap, mapPairs, nil
}

type PlutusList interface {
	Len() int
}

type PlutusIndefArray []PlutusData
type PlutusDefArray []PlutusData

/*
*

	Len returns the length of the PlutusIndefArray.

	Returns:
		int: The length of the PlutusIndefArray.
*/
func (pia PlutusIndefArray) Len() int {
	return len(pia)
}

/*
*

	Len returns the length of the PlutusDefArray.

	Returns:
		int: The length of the PlutusDefArray.
*/
func (pia PlutusDefArray) Len() int {
	return len(pia)
}

/*
*

	Clone creates a deep copy of the PlutusIndefArray.

	Returns:
		PlutusIndefArray: A deep copy of the PlutusIndefArray.
*/
func (pia *PlutusIndefArray) Clone() PlutusIndefArray {
	ret := make(PlutusIndefArray, 0, len(*pia))
	for _, v := range *pia {
		ret = append(ret, v.Clone())
	}
	return ret
}

/*
*

	MarshalCBOR encodes the PlutusIndefArray into a CBOR-encoded byte


		slice, in which it serializes the elements in indefinite-length array format.

		Returns:
	   		[]uint8: The CBOR-encoded byte slice.
	   		error: An error if marshaling fails.
*/
func (pia PlutusIndefArray) MarshalCBOR() ([]uint8, error) {
	res := make([]byte, 0)
	res = append(res, 0x9f)
	for _, el := range pia {
		bytes, err := cbor.Marshal(el)
		if err != nil {
			return nil, err
		}
		res = append(res, bytes...)
	}
	res = append(res, 0xff)
	return res, nil
}

type Datum struct {
	PlutusDataType PlutusType
	TagNr          uint64
	Value          any
}

/*
*

	ToPlutusData converts a datum to PlutusData, encoding
	the Datum into CBOR format and then decodes it into a
	PlutusData.

	Returns:
		PlutusData: The converted PlutusData.
*/
func (pd *Datum) ToPlutusData() PlutusData {
	var res PlutusData
	enc, _ := cbor.Marshal(pd)
	err := cbor.Unmarshal(enc, &res)
	if err != nil {
		// TODO: return errors
		return res
	}
	return res
}

/*
*

	Clone creates a deep copy of the Datum>

	Returns:
		Datum: A deep copy of the Datum
*/
func (pd *Datum) Clone() Datum {
	return Datum{
		PlutusDataType: pd.PlutusDataType,
		TagNr:          pd.TagNr,
		Value:          pd.Value,
	}
}

/*
*

	MarshalCBOR encodes the Datum into a CBOR-encoded byte slice,


		it applies a CBOR tag, if TagNr is not 0, otherwise it marshals the Value

		Returns:
	   		[]uint8: The CBOR-encoded byte slice.
	   		error: An error if marshaling fails.
*/
func (pd Datum) MarshalCBOR() ([]uint8, error) {
	if pd.TagNr == 0 {
		return cbor.Marshal(pd.Value)
	}
	return cbor.Marshal(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
}

/*
*

	UnmarshalCBOR decodes a CBOR-encoded byte slice into a Datum.


		It handles different Plutus data types and applies appropriate decoding logic.

		Returns:
	   		error: An error if unmarshaling fails.
*/
func (pd *Datum) UnmarshalCBOR(value []uint8) error {
	var x any
	err := cbor.Unmarshal(value, &x)
	if err != nil {
		return err
	}
	ok, valid := x.(cbor.Tag)
	if valid {
		switch ok.Content.(type) {
		case []any:
			pd.TagNr = ok.Number
			pd.PlutusDataType = PlutusArray
			res, err := cbor.Marshal(ok.Content)
			if err != nil {
				return err
			}
			y := new([]Datum)
			err = cbor.Unmarshal(res, y)
			if err != nil {
				return err
			}
			pd.Value = y

		default:
			//TODO SKIP
			return nil
		}
	} else {
		switch x.(type) {
		case []any:
			y := new([]Datum)
			err = cbor.Unmarshal(value, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusArray
			pd.Value = y
			pd.TagNr = 0
		case uint64:
			pd.PlutusDataType = PlutusInt
			pd.Value = x
			pd.TagNr = 0

		case []uint8:
			pd.PlutusDataType = PlutusBytes
			pd.Value = x
			pd.TagNr = 0

		case map[any]any:
			y := map[serialization.CustomBytes]Datum{}
			err = cbor.Unmarshal(value, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusMap
			pd.Value = y
			pd.TagNr = 0
		case map[uint64]any:
			y := map[serialization.CustomBytes]Datum{}
			err = cbor.Unmarshal(value, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusIntMap
			pd.Value = &y
			pd.TagNr = 0
		default:
		}

	}

	return nil
}

type PlutusData struct {
	PlutusDataType PlutusType
	TagNr          uint64
	Value          any
}

func (pd *PlutusData) String() string {
	var sb strings.Builder
	if pd.TagNr != 0 {
		sb.WriteString("Constr: ")
		sb.WriteString(strconv.FormatUint(pd.TagNr, 10))
		sb.WriteByte('\n')
	}
	switch pd.PlutusDataType {
	case PlutusArray:
		sb.WriteString("Array[\n")
		value, ok := pd.Value.(PlutusIndefArray)
		if ok {
			for _, v := range value {
				contentString := v.String()
				for idx, line := range strings.Split(contentString, "\n") {
					if idx == len(strings.Split(contentString, "\n"))-1 {
						sb.WriteString("    " + line)
					} else {
						sb.WriteString("    " + line + "\n")
					}
				}
				sb.WriteString(",\n")
			}
			sb.WriteByte(']')
		}
		value2, ok := pd.Value.(PlutusDefArray)
		if ok {
			for _, v := range value2 {
				contentString := v.String()
				for line := range strings.SplitSeq(contentString, "\n") {
					sb.WriteString("    " + line + "\n")
				}
				sb.WriteString(",\n")
			}
			sb.WriteByte(']')
		}
	case PlutusMap:
		value, ok := pd.Value.(map[serialization.CustomBytes]PlutusData)
		if ok {
			sb.WriteString("Map{\n")
			for k, v := range value {
				contentString := v.String()
				sb.WriteString(k.String() + ": ")
				for idx, line := range strings.Split(contentString, "\n") {
					if idx == 0 {
						sb.WriteString(line + "\n")
						continue
					}
					sb.WriteString("    " + line + "\n")
				}
				sb.WriteString(",\n")
			}
			sb.WriteByte('}')
		}

	case PlutusIntMap:
		value, ok := pd.Value.(map[serialization.CustomBytes]PlutusData)
		if ok {
			sb.WriteString("IntMap{\n")
			for k, v := range value {
				contentString := v.String()
				sb.WriteString(k.String() + ": ")
				for idx, line := range strings.Split(contentString, "\n") {
					if idx == 0 {
						sb.WriteString(line + "\n")
						continue
					}
					sb.WriteString("    " + line + "\n")
				}
				sb.WriteString(",\n")
			}
			sb.WriteByte('}')
		}
	case PlutusGenericMap:
		pairs, ok := pd.Value.(PlutusMapPairs)
		if ok {
			sb.WriteString("GenericMap{\n")
			for _, pair := range pairs {
				keyString := pair.Key.String()
				valueString := pair.Value.String()
				sb.WriteString(
					"    " + keyString + " => " + valueString + ",\n",
				)
			}
			sb.WriteByte('}')
		}
	case PlutusInt:
		sb.WriteString("Int(")
		if v, ok := pd.Value.(uint64); ok {
			sb.WriteString(strconv.FormatUint(v, 10))
		} else {
			sb.WriteString(fmt.Sprintf("%v", pd.Value))
		}
		sb.WriteByte(')')
	case PlutusBytes:
		sb.WriteString("Bytes(")
		if v, ok := pd.Value.([]uint8); ok {
			sb.WriteString(hex.EncodeToString(v))
		} else {
			sb.WriteString(fmt.Sprintf("%v", pd.Value))
		}
		sb.WriteByte(')')
	default:
		sb.WriteString(fmt.Sprintf("%v", pd.Value))
	}
	return sb.String()
}

/*
*

	Equal check if two PlutusData values are equal
	using their CBOR representations.

	Params:
		other (PlutusData): The other PlutusData to compare to.

	Returns:
		bool: True if the PlutusData are equal, false otherwise.
*/
func (pd *PlutusData) Equal(other PlutusData) bool {
	marshaledThis, _ := cbor.Marshal(pd)
	marshaledOther, _ := cbor.Marshal(other)
	return bytes.Equal(marshaledThis, marshaledOther)
}

/*
*

	ToDatum converts a PlutusData to a Datum, in which
	it encodes the PlutusData into CBOR format and later
	into a Datum.

	Returns:
		Datum: The converted Datum.
*/
func (pd *PlutusData) ToDatum() Datum {

	var res Datum
	enc, _ := cbor.Marshal(pd)
	err := cbor.Unmarshal(enc, &res)
	if err != nil {
		// TODO: return errors
		return res
	}
	return res
}

/*
*

	Clone creates a deep copy of a PlutusData object.

	Returns:
		PlutusData: A cloned PlutusData object.
*/
func (pd *PlutusData) Clone() PlutusData {
	return PlutusData{
		PlutusDataType: pd.PlutusDataType,
		TagNr:          pd.TagNr,
		Value:          pd.Value,
	}
}

/*
*

	MarshalCBOR encodes the PlutusData into a CBOR byte slice.

	Returns:
		[]uint8: The CBOR-encoded byte slice.
		error: An error, if any, during ecoding.
*/
func (pd *PlutusData) MarshalCBOR() ([]uint8, error) {
	//enc, _ := cbor.CanonicalEncOptions().EncMode()
	switch pd.PlutusDataType {
	case PlutusMap:
		customEnc, _ := cbor.EncOptions{
			Sort: cbor.SortBytewiseLexical,
		}.EncMode()
		if pd.TagNr != 0 {
			return customEnc.Marshal(
				cbor.Tag{Number: pd.TagNr, Content: pd.Value},
			)
		}
		return customEnc.Marshal(pd.Value)
	case PlutusIntMap:
		canonicalenc, _ := cbor.CanonicalEncOptions().EncMode()
		if pd.TagNr != 0 {
			return canonicalenc.Marshal(
				cbor.Tag{Number: pd.TagNr, Content: pd.Value},
			)
		}
		return canonicalenc.Marshal(pd.Value)
	case PlutusBigInt:
		return cbor.Marshal(pd.Value)
	case PlutusGenericMap:
		// Convert PlutusMapPairs back to a CBOR map
		pairs, ok := pd.Value.(PlutusMapPairs)
		if !ok {
			return nil, errors.New(
				"PlutusGenericMap value is not PlutusMapPairs",
			)
		}
		// Encode as raw key-value pairs with definite-length map header
		result := make([]byte, 0)
		mapLen := len(pairs)
		switch {
		case mapLen < 24:
			result = append(result, byte(apolloCbor.CborMapBase+mapLen))
		case mapLen <= 0xff:
			result = append(result, apolloCbor.CborMap1ByteLen, byte(mapLen))
		case mapLen <= 0xffff:
			result = append(result, apolloCbor.CborMap2ByteLen,
				byte(mapLen>>8), byte(mapLen))
		case mapLen <= 0xffffffff:
			result = append(
				result,
				apolloCbor.CborMap4ByteLen,
				byte(
					mapLen>>24,
				),
				byte(mapLen>>16),
				byte(mapLen>>8),
				byte(mapLen),
			)
		default:
			ml := uint64(mapLen)
			result = append(result, apolloCbor.CborMap8ByteLen,
				byte(ml>>56), byte(ml>>48), byte(ml>>40), byte(ml>>32),
				byte(ml>>24), byte(ml>>16), byte(ml>>8), byte(ml))
		}
		for _, pair := range pairs {
			keyBytes, err := cbor.Marshal(pair.Key)
			if err != nil {
				return nil, err
			}
			valueBytes, err := cbor.Marshal(pair.Value)
			if err != nil {
				return nil, err
			}
			result = append(result, keyBytes...)
			result = append(result, valueBytes...)
		}
		return result, nil
	default:
		if pd.TagNr == 0 {
			return cbor.Marshal(pd.Value)
		}
		return cbor.Marshal(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
	}

}

/*
*

		UnmarshalJSON unmarshals JSON-encoded PlutusData into a PlutusData object.

		Params:
	   		value ([]byte): The JSON-encoded data to unmarshal.

	 	Returns:
	   		error: An error, if any, during unmarshaling.
*/
func (pd *PlutusData) UnmarshalJSON(value []byte) error {
	var x any
	err := json.Unmarshal(value, &x)
	if err != nil {
		return err
	}
	switch x.(type) {
	case []any:
		y := new([]PlutusData)
		err = json.Unmarshal(value, y)
		if err != nil {
			return err
		}
		pd.PlutusDataType = PlutusArray
		pd.Value = PlutusIndefArray(*y)
		pd.TagNr = 0
	case map[string]any:
		val := x.(map[string]any)
		_, ok := val["fields"]
		if ok {
			contents, _ := json.Marshal(val["fields"])
			var tag int
			constructor, ok := val["constructor"]
			if ok {
				constrfloat := constructor.(float64)
				if constrfloat < 7 {
					tag = int(121 + constrfloat)
				} else if 7 <= constrfloat && constrfloat < 1400 {
					tag = int(1280 + constrfloat - 7)
				} else {
					return errors.New("constructor out of range ")
				}
			} else {
				tag = 0
			}
			y := new(PlutusData)
			err = json.Unmarshal(contents, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusMap
			pd.Value = y
			pd.TagNr = uint64(tag)
		} else if _, ok := val["biguint"]; ok {
			vl := big.NewInt(0)
			vl.SetBytes([]byte(val["biguint"].(string)))
			pd.PlutusDataType = PlutusInt
			pd.Value = vl.Uint64()
		} else if _, ok := val["bignint"]; ok {
			vl := big.NewInt(0)
			vl.SetBytes([]byte(val["bignint"].(string)))
			pd.PlutusDataType = PlutusInt
			pd.Value = vl.Uint64()
		} else if _, ok := val["bytes"]; ok {
			pd.PlutusDataType = PlutusBytes
			pd.Value, _ = hex.DecodeString(val["bytes"].(string))
		} else if _, ok := val["int"]; ok {
			pd.PlutusDataType = PlutusInt
			pd.Value = uint64(val["int"].(float64))
		} else if valu, ok := val["map"]; ok {
			var tag int
			constructor, ok := val["constructor"]
			if ok {
				constrfloat := constructor.(float64)
				if constrfloat < 7 {
					tag = int(121 + constrfloat)
				} else if 7 <= constrfloat && constrfloat < 1400 {
					tag = int(1280 + constrfloat - 7)
				} else {
					return errors.New("constructor out of range ")
				}
			} else {
				tag = 0
			}
			isInt := false
			normalMap := make(map[serialization.CustomBytes]PlutusData)
			IntMap := make(map[uint64]PlutusData)
			for _, element := range valu.([]any) {
				dictionary, ok := element.(map[string]any)
				if ok {
					kval, okk := dictionary["k"].(map[string]any)
					vval, okv := dictionary["v"].(map[string]any)
					if okk && okv {
						if kvalue, okk := kval["int"]; okk {
							isInt = true
							pd := PlutusData{}
							var marshaled []byte
							marshaled, err = json.Marshal(vval)
							if err != nil {
								return err
							}

							err = json.Unmarshal(marshaled, &pd)
							if err != nil {
								return err
							}
							parsedInt := kvalue.(float64)
							IntMap[uint64(parsedInt)] = pd
						} else {
							pd := PlutusData{}
							var marshaled []byte
							marshaled, err = json.Marshal(vval)
							if err != nil {
								return err
							}

							err = json.Unmarshal(marshaled, &pd)
							if err != nil {
								return err
							}
							bytes, _ := hex.DecodeString(kval["bytes"].(string))
							cb := serialization.NewCustomBytes(string(bytes))
							normalMap[cb] = pd
						}
					}
				}
			}
			if isInt {
				pd.PlutusDataType = PlutusIntMap
				pd.Value = IntMap
				pd.TagNr = uint64(tag)
			} else {
				pd.PlutusDataType = PlutusMap
				pd.Value = normalMap
				pd.TagNr = uint64(tag)
			}
			return nil

		} else if valu, ok := val["list"]; ok {
			y := new([]PlutusData)
			var tag int
			constructor, ok := val["constructor"]
			if ok {
				constrfloat := constructor.(float64)
				if constrfloat < 7 {
					tag = int(121 + constrfloat)
				} else if 7 <= constrfloat && constrfloat < 1400 {
					tag = int(1280 + constrfloat - 7)
				} else {
					return errors.New("constructor out of range ")
				}
			} else {
				tag = 0
			}

			marshaled, _ := json.Marshal(valu)
			err = json.Unmarshal(marshaled, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusArray
			pd.Value = PlutusIndefArray(*y)
			pd.TagNr = uint64(tag)

		}
	default:
		return fmt.Errorf("unsupported JSON structure for PlutusData: %T", x)
	}
	return nil
}

// unmarshalMap handles CBOR map decoding for PlutusData.
// This is separate because CBOR maps with tag-based keys (like constructor keys)
// cannot be decoded into Go maps directly due to cbor.Tag not being comparable.
func (pd *PlutusData) unmarshalMap(value []uint8) error {
	dataType, val, err := decodeMapValue(value)
	if err != nil {
		return err
	}
	pd.PlutusDataType = dataType
	pd.Value = val
	pd.TagNr = 0
	return nil
}

// unmarshalTaggedMap handles CBOR tags whose content is a map that might have tag-based keys.
// This is needed because cbor.Tag with map content containing tag keys fails during generic decode.
func (pd *PlutusData) unmarshalTaggedMap(
	tagNumber uint64,
	mapContent []byte,
) error {
	dataType, val, err := decodeMapValue(mapContent)
	if err != nil {
		return err
	}
	pd.PlutusDataType = dataType
	pd.Value = val
	pd.TagNr = tagNumber
	return nil
}

type PlutusDataKey struct {
	CborHexValue string
}

func (pdk *PlutusDataKey) String() string {
	return pdk.CborHexValue

}

func (pdk *PlutusDataKey) UnmarshalCBOR(value []uint8) error {
	pdk.CborHexValue = hex.EncodeToString(value)
	return nil
}

func (pdk *PlutusDataKey) MarshalCBOR() ([]uint8, error) {
	decodedHex, _ := hex.DecodeString(pdk.CborHexValue)
	return decodedHex, nil
}

/*
*

		UnmarshalCBOR unmarshals CBOR-encoded data into a PlutusData object.

		Params:
	   		value ([]uint8): The CBOR-encoded data to unmarshal.

	 	Returns:
	   		error: An error, if any, during unmarshaling.
*/
func (pd *PlutusData) UnmarshalCBOR(value []uint8) error {
	if len(value) == 0 {
		return errors.New("empty CBOR data")
	}

	// Determine CBOR major type from the first byte
	// This allows us to route to type-specific handlers without using generic any decode,
	// which fails when there are maps with tag-based keys (cbor.Tag is not comparable)
	majorType := value[0] >> 5

	switch majorType {
	case 0, 1: // Unsigned or negative integer
		var intVal uint64
		if err := cbor.Unmarshal(value, &intVal); err != nil {
			// Try big int
			var bigVal big.Int
			if err2 := cbor.Unmarshal(value, &bigVal); err2 != nil {
				return err
			}
			pd.PlutusDataType = PlutusBigInt
			pd.Value = bigVal
			pd.TagNr = 0
			return nil
		}
		pd.PlutusDataType = PlutusInt
		pd.Value = intVal
		pd.TagNr = 0
		return nil

	case 2: // Byte string
		var byteVal []byte
		if err := cbor.Unmarshal(value, &byteVal); err != nil {
			return err
		}
		pd.PlutusDataType = PlutusBytes
		pd.Value = byteVal
		pd.TagNr = 0
		return nil

	case 4: // Array
		// Decode array elements as PlutusData to handle nested maps with tag keys
		if value[0] == 0x9f { // Indefinite length array
			y := PlutusIndefArray{}
			if err := cbor.Unmarshal(value, &y); err != nil {
				return err
			}
			pd.PlutusDataType = PlutusArray
			pd.Value = y
			pd.TagNr = 0
		} else { // Definite length array
			y := PlutusDefArray{}
			if err := cbor.Unmarshal(value, &y); err != nil {
				return err
			}
			pd.PlutusDataType = PlutusArray
			pd.Value = y
			pd.TagNr = 0
		}
		return nil

	case 5: // Map
		return pd.unmarshalMap(value)

	case 6: // Tag
		return pd.unmarshalTag(value)

	case 7: // Simple values (false, true, null, undefined, floats)
		// Handle CBOR null (0xf6) which represents an empty/uninitialized PlutusData
		if value[0] == 0xf6 {
			pd.PlutusDataType = PlutusBytes
			pd.Value = []byte{}
			pd.TagNr = 0
			return nil
		}
		// For other simple values, fall through to generic decode
		fallthrough

	default:
		// Fallback to generic decode for unknown types
		var x any
		err := cbor.Unmarshal(value, &x)
		if err != nil {
			return err
		}
		// Handle the result
		return pd.handleGenericValue(x, value)
	}
}

// unmarshalTag handles CBOR tags (major type 6) for PlutusData.
// This is separate to avoid the cbor.Unmarshal(value, &any) issue where nested
// maps with tag keys cause decode failures.
func (pd *PlutusData) unmarshalTag(value []uint8) error {
	// Decode as RawTag to get tag number and preserve raw content bytes
	var rawTag apolloCbor.RawTag
	if err := apolloCbor.Decode(value, &rawTag); err != nil {
		return errors.New("tag decode: " + err.Error())
	}

	tagNum := rawTag.Number
	content := []byte(rawTag.Content)

	// Handle special CBOR tags first
	switch tagNum {
	case 2, 3: // Big integer tags (2 = positive, 3 = negative)
		// Big integers are encoded as byte strings
		var bigInt big.Int
		if err := cbor.Unmarshal(value, &bigInt); err != nil {
			return errors.New("big int decode: " + err.Error())
		}
		pd.PlutusDataType = PlutusBigInt
		pd.Value = bigInt
		pd.TagNr = 0
		return nil
	}

	if len(content) == 0 {
		// Empty tag content
		pd.PlutusDataType = PlutusArray
		pd.Value = PlutusIndefArray{}
		pd.TagNr = tagNum
		return nil
	}

	// Determine content type and handle accordingly
	contentMajorType := content[0] >> 5

	switch contentMajorType {
	case 2: // Byte string - common for tag 24 (CBOR-encoded data)
		var byteContent []byte
		if err := cbor.Unmarshal(content, &byteContent); err != nil {
			return errors.New("tag byte content: " + err.Error())
		}
		pd.PlutusDataType = PlutusBytes
		pd.Value = byteContent
		pd.TagNr = tagNum
		return nil

	case 4: // Array - constructor with fields
		if content[0] == 0x9f { // Indefinite array
			y := PlutusIndefArray{}
			if err := cbor.Unmarshal(content, &y); err != nil {
				return errors.New("tag indef array content: " + err.Error())
			}
			pd.PlutusDataType = PlutusArray
			pd.Value = y
			pd.TagNr = tagNum
		} else {
			y := PlutusDefArray{}
			if err := cbor.Unmarshal(content, &y); err != nil {
				return errors.New("tag def array content: " + err.Error())
			}
			pd.PlutusDataType = PlutusArray
			pd.Value = y
			pd.TagNr = tagNum
		}
		return nil

	case 5: // Map - handle with our special map decoder
		return pd.unmarshalTaggedMap(tagNum, content)

	default:
		// For other content types, try generic decode on the content
		var contentPd PlutusData
		if err := contentPd.UnmarshalCBOR(content); err != nil {
			return errors.New("tag content decode: " + err.Error())
		}
		// Wrap the decoded content
		pd.PlutusDataType = contentPd.PlutusDataType
		pd.Value = contentPd.Value
		pd.TagNr = tagNum
		return nil
	}
}

// handleGenericValue processes an already-decoded interface{} value.
// This is only used as a fallback for unusual CBOR types.
func (pd *PlutusData) handleGenericValue(x any, value []uint8) error {
	var err error
	//fmt.Println(hex.EncodeToString(value))
	ok, valid := x.(cbor.Tag)
	if valid {
		switch content := ok.Content.(type) {
		case big.Int:
			pd.PlutusDataType = PlutusBigInt
			pd.Value = content
			pd.TagNr = 0
		case *big.Int:
			pd.PlutusDataType = PlutusBigInt
			if content != nil {
				pd.Value = *content
			} else {
				pd.Value = big.Int{}
			}
			pd.TagNr = 0
		case []any:
			pd.TagNr = ok.Number
			pd.PlutusDataType = PlutusArray
			lenTag := len([]byte(strconv.FormatUint(ok.Number, 10)))
			if value[lenTag-1] == 0x9f {
				y := PlutusIndefArray{}
				err = cbor.Unmarshal(value[lenTag-1:], &y)
				if err != nil {
					return err
				}
				pd.Value = y
			} else {
				y := PlutusDefArray{}
				err = cbor.Unmarshal(value[lenTag-1:], &y)
				if err != nil {

					return err
				}
				pd.Value = y
			}
		case []uint8:
			pd.TagNr = ok.Number
			pd.PlutusDataType = PlutusBytes
			pd.Value = content
		case map[any]any:
			dataType, val, decErr := decodeMapValue(value)
			if decErr != nil {
				return decErr
			}
			pd.PlutusDataType = dataType
			pd.Value = val
			pd.TagNr = 0

		default:
			//TODO SKIP
			return nil
		}
	} else {
		switch x := x.(type) {
		case big.Int:
			pd.PlutusDataType = PlutusBigInt
			pd.Value = x
			pd.TagNr = 0
		case *big.Int:
			pd.PlutusDataType = PlutusBigInt
			if x != nil {
				pd.Value = *x
			} else {
				pd.Value = big.Int{}
			}
			pd.TagNr = 0
		case []any:
			if value[0] == 0x9f {
				y := PlutusIndefArray{}
				err = cbor.Unmarshal(value, &y)
				if err != nil {
					return err
				}
				pd.PlutusDataType = PlutusArray
				pd.Value = y
				pd.TagNr = 0
			} else {
				y := PlutusDefArray{}
				err = cbor.Unmarshal(value, &y)
				if err != nil {
					return err
				}
				pd.PlutusDataType = PlutusArray
				pd.Value = y
				pd.TagNr = 0
			}
		case uint64:
			pd.PlutusDataType = PlutusInt
			pd.Value = x
			pd.TagNr = 0

		case []uint8:
			pd.PlutusDataType = PlutusBytes
			pd.Value = x
			pd.TagNr = 0

		case map[any]any:
			dataType, val, decErr := decodeMapValue(value)
			if decErr != nil {
				return decErr
			}
			pd.PlutusDataType = dataType
			pd.Value = val
			pd.TagNr = 0
		default:
			// Fallback for nil and unrecognized types from generic interface{} decode.
			// Return an error so callers are aware that an unsupported type was encountered.
			pd.PlutusDataType = PlutusBytes
			pd.Value = []byte{}
			pd.TagNr = 0
			return fmt.Errorf("handleGenericValue: unsupported generic value type %T", x)
		}

	}

	return nil
}

type RawPlutusData struct {
	//TODO
}

/*
*

	ToCbor converts the given interface to a hexadecimal-encoded CBOR string.

	Params:
		x (interface{}): The input value to be encoded to CBOR to converted
						 to a hexadecimal string.

	Returns:
		string: The hexadecimal-encoded CBOR representation of the input value.
		error: An error if the conversion fails.
*/
func ToCbor(x any) (string, error) {
	bytes, err := cbor.Marshal(x)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

/*
*

		PlutusDataHash computes the hash of a PlutusData structure using the Blake2b algorithm.

	 	Params:


		   	pd (*PlutusData): A pointer to the PlutusData structure to be hashed.

		Returns:
	  		serialization.DatumHash: The hash of the PlutusData.
			error: An error if the PlutusDataHash fails.
*/
func PlutusDataHash(pd *PlutusData) (serialization.DatumHash, error) {
	bytes, err := cbor.Marshal(pd)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	finalbytes := make([]byte, 0, len(bytes))
	finalbytes = append(finalbytes, bytes...)
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	r := serialization.DatumHash{Payload: hash.Sum(nil)}
	return r, nil
}

/*
*

	HashDatum computes the hash of a CBOR marshaler using the Blake2b algorithm.

	Params:
		d (cbor.Marshaler): The CBOR marshaler to be hashed

	Returns:
		serialization.DatumHash: The hash of the CBOR marshaler.
		error: An error if the hash Datum fails.
*/
func HashDatum(d cbor.Marshaler) (serialization.DatumHash, error) {
	bytes, err := cbor.Marshal(d)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	finalbytes := make([]byte, 0, len(bytes))
	finalbytes = append(finalbytes, bytes...)
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	r := serialization.DatumHash{Payload: hash.Sum(nil)}
	return r, nil
}

type ScriptHashable interface {
	Hash() (serialization.ScriptHash, error)
}

/*
*

	PlutusScriptHash computes the script hash of a ScriptHashable object.

	Params:
		script (ScriptHashable): The ScriptHashable object to be hashed.

	Returns:
		serialization.ScriptHash: The script hash of the ScriptHashable object.
*/
func PlutusScriptHash(script ScriptHashable) serialization.ScriptHash {
	hash, _ := script.Hash()
	return hash
}

type PlutusV1Script []byte

/*
*

		ToAddress converts a PlutusV1Script to an Address with an optional staking credential.

	 	Params:


	  		stakingCredential ([]byte): The staking credential to include in the address.

		Returns:
	  		Address.Address: The generated address.
*/
func (ps *PlutusV1Script) ToAddress(stakingCredential []byte) Address.Address {
	hash := PlutusScriptHash(ps)
	if stakingCredential == nil {
		return Address.Address{
			PaymentPart: hash.Bytes(),
			StakingPart: nil,
			Network:     Address.MAINNET,
			AddressType: Address.SCRIPT_NONE,
			HeaderByte:  0b01110001,
			Hrp:         "addr",
		}
	} else {
		return Address.Address{
			PaymentPart: hash.Bytes(),
			StakingPart: stakingCredential,
			Network:     Address.MAINNET,
			AddressType: Address.SCRIPT_KEY,
			HeaderByte:  0b00010001,
			Hrp:         "addr",
		}
	}
}

type PlutusV2Script []byte

type PlutusV3Script []byte

func (ps3 PlutusV3Script) Hash() (serialization.ScriptHash, error) {
	finalbytes, err := hex.DecodeString("03")
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	finalbytes = append(finalbytes, ps3...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	r := serialization.ScriptHash{}
	copy(r[:], hash.Sum(nil))
	return r, nil
}

func (ps3 *PlutusV3Script) ToAddress(
	stakingCredential []byte,
	isStakingScript bool,
	network constants.Network,
) Address.Address {
	hash := PlutusScriptHash(ps3)
	if stakingCredential == nil {
		if network == constants.MAINNET {
			return Address.Address{
				PaymentPart: hash.Bytes(),
				StakingPart: nil,
				Network:     Address.MAINNET,
				AddressType: Address.SCRIPT_NONE,
				HeaderByte:  0b01110001,
				Hrp:         "addr",
			}
		} else {
			return Address.Address{
				PaymentPart: hash.Bytes(),
				StakingPart: nil,
				Network:     Address.TESTNET,
				AddressType: Address.SCRIPT_KEY,
				HeaderByte:  0b01110001,
				Hrp:         "addr_test",
			}
		}
	} else {
		if isStakingScript {
			if network == constants.MAINNET {
				return Address.Address{
					PaymentPart: hash.Bytes(),
					StakingPart: stakingCredential,
					Network:     Address.MAINNET,
					AddressType: Address.SCRIPT_SCRIPT,
					HeaderByte:  0b00110001,
					Hrp:         "addr",
				}
			} else {
				return Address.Address{
					PaymentPart: hash.Bytes(),
					StakingPart: stakingCredential,
					Network:     Address.TESTNET,
					AddressType: Address.SCRIPT_SCRIPT,
					HeaderByte:  0b00110000,
					Hrp:         "addr_test",
				}
			}
		} else {
			if network == constants.MAINNET {
				return Address.Address{
					PaymentPart: hash.Bytes(),
					StakingPart: stakingCredential,
					Network:     Address.MAINNET,
					AddressType: Address.SCRIPT_KEY,
					HeaderByte:  0b00010001,
					Hrp:         "addr",
				}
			} else {
				return Address.Address{
					PaymentPart: hash.Bytes(),
					StakingPart: stakingCredential,
					Network:     Address.TESTNET,
					AddressType: Address.SCRIPT_KEY,
					HeaderByte:  0b00010000,
					Hrp:         "addr_test",
				}
			}
		}
	}
}

/*
*

		ToAddress converts a PlutusV2Script to an Address with an optional staking credential.

	 	Params:


	  		stakingCredential ([]byte): The staking credential to include in the address.

		Returns:
	  		Address.Address: The generated address.
*/
func (ps *PlutusV2Script) ToAddress(
	stakingCredential []byte,
	network constants.Network,
) Address.Address {
	hash := PlutusScriptHash(ps)
	if stakingCredential == nil {
		if network == constants.MAINNET {
			return Address.Address{
				PaymentPart: hash.Bytes(),
				StakingPart: nil,
				Network:     Address.MAINNET,
				AddressType: Address.SCRIPT_NONE,
				HeaderByte:  0b01110001,
				Hrp:         "addr",
			}
		} else {
			return Address.Address{
				PaymentPart: hash.Bytes(),
				StakingPart: nil,
				Network:     Address.TESTNET,
				AddressType: Address.SCRIPT_KEY,
				HeaderByte:  0b01110001,
				Hrp:         "addr_test",
			}
		}
	} else {
		if network == constants.MAINNET {
			return Address.Address{
				PaymentPart: hash.Bytes(),
				StakingPart: stakingCredential,
				Network:     Address.MAINNET,
				AddressType: Address.SCRIPT_KEY,
				HeaderByte:  0b00010001,
				Hrp:         "addr",
			}
		} else {
			return Address.Address{
				PaymentPart: hash.Bytes(),
				StakingPart: stakingCredential,
				Network:     Address.TESTNET,
				AddressType: Address.SCRIPT_KEY,
				HeaderByte:  0b00010000,
				Hrp:         "addr_test",
			}
		}
	}
}

/*
*

	 	Hash computes the script hash for a PlutusV1Script.

	 	Returns:
	   		serialization.ScriptHash: The script hash of the PlutusV1Script.
			error: An error if the hashing fails.
*/
func (ps PlutusV1Script) Hash() (serialization.ScriptHash, error) {
	finalbytes, err := hex.DecodeString("01")
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	finalbytes = append(finalbytes, ps...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	r := serialization.ScriptHash{}
	copy(r[:], hash.Sum(nil))
	return r, nil
}

/*
*

	 	Hash computes the script hash for a PlutusV2Script.

	 	Returns:
	   		serialization.ScriptHash: The script hash of the PlutusV2Script.
			error: An error if the Hashing fails.
*/
func (ps PlutusV2Script) Hash() (serialization.ScriptHash, error) {
	finalbytes, err := hex.DecodeString("02")
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	finalbytes = append(finalbytes, ps...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	r := serialization.ScriptHash{}
	copy(r[:], hash.Sum(nil))
	return r, nil
}
