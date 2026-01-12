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

	"github.com/Salvionied/apollo/v2/constants"
	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/Address"

	"github.com/blinklabs-io/gouroboros/cbor"

	"golang.org/x/crypto/blake2b"
)

type DatumType byte

const (
	DatumTypeHash   DatumType = 0
	DatumTypeInline DatumType = 1
)

type DatumOption struct {
	DatumType DatumType
	Hash      []byte
	Inline    *PlutusData
}

func (d *DatumOption) UnmarshalCBOR(b []byte) error {
	var cborDatumOption struct {
		cbor.StructAsArray
		DatumType DatumType
		Content   cbor.RawMessage
	}
	_, err := cbor.Decode(b, &cborDatumOption)
	if err != nil {
		return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", err)

	}
	switch cborDatumOption.DatumType {
	case DatumTypeInline:
		var cborDatumInline PlutusData
		_, errInline := cbor.Decode(cborDatumOption.Content, &cborDatumInline)
		if errInline != nil {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", errInline)
		}
		if cborDatumInline.TagNr == 24 {
			taggedBytes, valid := cborDatumInline.Value.([]byte)
			if !valid {
				return errors.New(
					"DatumOption: UnmarshalCBOR: found tag 24 but there wasn't a byte array",
				)
			}
			var inline PlutusData
			_, err := cbor.Decode(taggedBytes, &inline)
			if err != nil {
				return err
			}
			d.DatumType = DatumTypeInline
			d.Inline = &inline
		} else {
			d.DatumType = DatumTypeInline
			d.Inline = &cborDatumInline
		}
		return nil
	case DatumTypeHash:
		var cborDatumHash []byte
		_, errHash := cbor.Decode(cborDatumOption.Content, &cborDatumHash)
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
		cbor.StructAsArray
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
		bytes, err := cbor.Encode(d.Inline)
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
	return cbor.Encode(format)
}

type ScriptRef []byte

func (sr ScriptRef) Len() int {
	return len(sr)
}

type CostModels map[serialization.CustomBytes]CM

type CM map[string]int

// sortedMapValues returns the values of a map[string]int sorted by key.
func sortedMapValues(m map[string]int) []int {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := make([]int, 0, len(keys))
	for _, k := range keys {
		values = append(values, m[k])
	}
	return values
}

/*
*

	MarshalCBOR encodes the CM into a CBOR-encoded byte slice, in which
	it serializes the map key alphabetically and encodes the respective values.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if marshaling fails.
*/
func (cm CM) MarshalCBOR() ([]byte, error) {
	res := sortedMapValues(cm)
	partial, err := cbor.Encode(res)
	if err != nil {
		return nil, err
	}
	if partial == nil {
		return nil, errors.New("cbor.Encode returned nil")
	}
	partial[1] = 0x9f
	partial = append(partial, 0xff)
	return cbor.Encode(partial[1:])
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
	return cbor.Encode(res)
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
	return cbor.Encode(sortedMapValues(cv))
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

var COST_MODELSV2 = map[int]any{1: PLUTUSV2COSTMODEL}

var COST_MODELSV1 = map[serialization.CustomBytes]any{
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
)

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
		bytes, err := cbor.Encode(el)
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
		error: An error if conversion fails.
*/
func (pd *Datum) ToPlutusData() (PlutusData, error) {
	var res PlutusData
	enc, err := cbor.Encode(pd)
	if err != nil {
		return res, err
	}
	_, err = cbor.Decode(enc, &res)
	if err != nil {
		return res, err
	}
	return res, nil
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
		return cbor.Encode(pd.Value)
	} else {
		return cbor.Encode(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
	}
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
	_, err := cbor.Decode(value, &x)
	if err != nil {
		return err
	}
	ok, valid := x.(cbor.Tag)
	if valid {
		switch ok.Content.(type) {
		case []any:
			pd.TagNr = ok.Number
			pd.PlutusDataType = PlutusArray
			// Avoid recursive decoding - manually parse the array
			content := ok.Content.([]any)
			datumArray := make([]Datum, len(content))
			for i, item := range content {
				// Create a simple datum for each item
				switch v := item.(type) {
				case uint64:
					datumArray[i] = Datum{PlutusDataType: PlutusInt, Value: v}
				case []byte:
					datumArray[i] = Datum{PlutusDataType: PlutusBytes, Value: v}
				default:
					// For complex types, encode and decode individually
					itemBytes, encErr := cbor.Encode(v)
					if encErr != nil {
						return encErr
					}
					err = datumArray[i].UnmarshalCBOR(itemBytes)
					if err != nil {
						return err
					}
				}
			}
			pd.Value = &datumArray

		default:
			//TODO SKIP
			return nil
		}
	} else {
		switch x.(type) {
		case []any:
			y := new([]Datum)
			_, err := cbor.Decode(value, y)
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
			_, err := cbor.Decode(value, &y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusMap
			pd.Value = y
			pd.TagNr = 0
		case map[uint64]any:
			y := map[serialization.CustomBytes]Datum{}
			_, err := cbor.Decode(value, &y)
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
	// Raw holds the original CBOR bytes for this node when available.
	// It is optional and used to preserve exact-wire encodings for
	// historical fixtures that rely on specific CBOR bytes.
	Raw []byte
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
				sb.WriteString(k.HexString() + ": ")
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
			sb.WriteString("intMap{\n")
			for k, v := range value {
				contentString := v.String()
				sb.WriteString(k.HexString() + ": ")
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
	case PlutusInt:
		sb.WriteString("Int(")
		sb.WriteString(strconv.FormatUint(pd.Value.(uint64), 10))
		sb.WriteByte(')')
	case PlutusBytes:
		sb.WriteString("Bytes(")
		sb.WriteString(hex.EncodeToString(pd.Value.([]uint8)))
		sb.WriteByte(')')
	default:
		sb.WriteString(fmt.Sprintf("%v", pd.Value))
	}
	return sb.String()
}

func (pd *PlutusData) UnmarshalCBOR(value []uint8) error {
	var x any
	_, err := cbor.Decode(value, &x)
	if err != nil {
		return err
	}
	result, err := AnyToPlutusData(x)
	if err != nil {
		return err
	}
	*pd = result
	return nil
}

// buildTaggedPlutusData creates a PlutusData with the given type, tag, and
// value, optionally capturing raw CBOR bytes from the original tagged value.
// This helper reduces duplication in AnyToPlutusData tag case handling.
func buildTaggedPlutusData(
	dataType PlutusType,
	tagNr uint64,
	value any,
	rawSource any,
) PlutusData {
	pd := PlutusData{
		PlutusDataType: dataType,
		TagNr:          tagNr,
		Value:          value,
	}
	if rawSource != nil {
		if b, err := cbor.Encode(rawSource); err == nil {
			pd.Raw = b
		}
	}
	return pd
}

// AnyToPlutusData converts a decoded CBOR value (any) into a PlutusData
// structure. It handles CBOR tags, integers, byte strings, arrays, maps,
// and nested CBOR data. Returns an error if the type is not supported.
func AnyToPlutusData(v any) (PlutusData, error) {
	switch val := v.(type) {
	case cbor.Tag:
		switch val.Number {
		case 121, 122:
			// Tags 121/122: PlutusBool (nil content) or PlutusList/tagged array
			// Tag 121 with nil = false (int 0), Tag 122 with nil = true (int 1)
			if val.Content == nil {
				nilValue := int64(val.Number - 121) // 121->0, 122->1
				return buildTaggedPlutusData(
					PlutusInt, val.Number, nilValue, nil,
				), nil
			}
			contentPd, err := AnyToPlutusData(val.Content)
			if err != nil {
				return PlutusData{}, err
			}
			return buildTaggedPlutusData(
				PlutusArray, val.Number, contentPd.Value, val,
			), nil
		case 123:
			// PlutusList def - but some historical encodings put a map under
			// this tag. If the tagged content decodes to a map, treat it as
			// a PlutusMap (preserve TagNr) so Unmarshal paths expecting maps
			// succeed.
			contentPd, err := AnyToPlutusData(val.Content)
			if err != nil {
				return PlutusData{}, err
			}
			// If the decoded content is an array, convert to DefArray
			if arr, ok := contentPd.Value.(PlutusIndefArray); ok {
				return buildTaggedPlutusData(
					PlutusArray, 123, PlutusDefArray(arr), val,
				), nil
			}
			// If the decoded content is a map (string-keyed), return PlutusMap
			switch m := contentPd.Value.(type) {
			case map[string]PlutusData:
				return buildTaggedPlutusData(PlutusMap, 123, m, val), nil
			case map[serialization.CustomBytes]PlutusData:
				return buildTaggedPlutusData(PlutusMap, 123, m, nil), nil
			}
			// Fallback: return as array with the decoded value
			return buildTaggedPlutusData(
				PlutusArray, 123, contentPd.Value, val,
			), nil
		case 124, 125:
			// Tags 124/125: PlutusBigInt (positive/negative)
			contentPd, err := AnyToPlutusData(val.Content)
			if err != nil {
				return PlutusData{}, err
			}
			var value big.Int
			if bigInt, ok := contentPd.Value.(big.Int); ok {
				value = bigInt
				if val.Number == 125 {
					value.Neg(&value)
				}
			}
			return buildTaggedPlutusData(
				PlutusBigInt, val.Number, value, val,
			), nil
		case 126, 127:
			// Tags 126/127: PlutusBytes (indef/def)
			contentPd, err := AnyToPlutusData(val.Content)
			if err != nil {
				return PlutusData{}, err
			}
			var value []byte
			if b, ok := contentPd.Value.([]byte); ok {
				value = b
			}
			return buildTaggedPlutusData(
				PlutusBytes, val.Number, value, val,
			), nil
		default:
			// For unknown tags, preserve the tag number and keep the
			// underlying content type so that tagged integers remain
			// integers, tagged bytes remain bytes, etc.
			contentPd, err := AnyToPlutusData(val.Content)
			if err != nil {
				return PlutusData{}, err
			}
			contentPd.TagNr = val.Number
			return contentPd, nil
		}
	case []any:
		arr := make(PlutusIndefArray, len(val))
		for i, item := range val {
			pd, err := AnyToPlutusData(item)
			if err != nil {
				return PlutusData{}, err
			}
			arr[i] = pd
		}
		pd := PlutusData{PlutusDataType: PlutusArray, Value: arr}
		if b, err := cbor.Encode(val); err == nil {
			pd.Raw = b
		}
		return pd, nil
	case map[any]any:
		m := make(map[serialization.CustomBytes]PlutusData)
		for k, v := range val {
			var keyBytes []byte
			switch kt := k.(type) {
			case string:
				keyBytes = []byte(kt)
			case []byte:
				keyBytes = kt
			default:
				// Try to encode then decode the key to extract a primitive
				kb, err := cbor.Encode(k)
				if err != nil {
					return PlutusData{}, err
				}
				var kv any
				_, err = cbor.Decode(kb, &kv)
				if err != nil {
					// Fallback to the encoded form if decode fails
					keyBytes = kb
				} else {
					switch kv2 := kv.(type) {
					case []byte:
						keyBytes = kv2
					case string:
						keyBytes = []byte(kv2)
					default:
						// Fallback to the encoded form
						keyBytes = kb
					}
				}
			}
			// Build CustomBytes key using the hex encoding of raw key bytes
			cb := serialization.CustomBytes{Value: hex.EncodeToString(keyBytes)}
			child, err := AnyToPlutusData(v)
			if err != nil {
				return PlutusData{}, err
			}
			// attempt to capture raw bytes for the child as well
			if b, err := cbor.Encode(v); err == nil {
				child.Raw = b
			}
			m[cb] = child
		}
		pd := PlutusData{PlutusDataType: PlutusMap, Value: m}
		if b, err := cbor.Encode(val); err == nil {
			pd.Raw = b
		}
		return pd, nil
	case int64:
		return PlutusData{PlutusDataType: PlutusInt, Value: val}, nil
	case uint64:
		return PlutusData{PlutusDataType: PlutusInt, Value: int64(val)}, nil
	case big.Int:
		return PlutusData{PlutusDataType: PlutusBigInt, Value: val}, nil
	case *big.Int:
		if val != nil {
			return PlutusData{PlutusDataType: PlutusBigInt, Value: *val}, nil
		}
		return PlutusData{PlutusDataType: PlutusBigInt, Value: big.Int{}}, nil
	case []byte:
		return PlutusData{PlutusDataType: PlutusBytes, Value: val}, nil
	case string:
		return PlutusData{PlutusDataType: PlutusBytes, Value: []byte(val)}, nil
	case cbor.WrappedCbor:
		// Tag 24: nested CBOR data - decode and recursively process
		// This handles transactions built by other tools (e.g., Lucid Evolution)
		var decoded any
		if _, err := cbor.Decode(val.Bytes(), &decoded); err != nil {
			// If decoding fails, fall back to treating as raw bytes
			return PlutusData{PlutusDataType: PlutusBytes, Value: val.Bytes()}, nil
		}
		return AnyToPlutusData(decoded)
	case interface{ Bytes() []byte }:
		if val == nil {
			return PlutusData{
				PlutusDataType: PlutusBytes,
				Value:          []byte{},
			}, nil
		}
		return PlutusData{PlutusDataType: PlutusBytes, Value: val.Bytes()}, nil
	case nil:
		return PlutusData{
			PlutusDataType: PlutusArray,
			Value:          PlutusIndefArray{},
		}, nil
	default:
		return PlutusData{}, fmt.Errorf("unsupported type: %T", v)
	}
}

// encodeTagBytes constructs the CBOR tag prefix bytes for a given tag number.
// CBOR tags use different encodings based on the tag value:
//   - Tags 0-23: single byte 0xc0 + tagNr
//   - Tags 24-255: two bytes 0xd8 + tagNr
//   - Tags 256-65535: three bytes 0xd9 + tagNr (big-endian)
//   - Tags 65536+: five bytes 0xda + tagNr (big-endian)
func encodeTagBytes(tagNr uint64) []byte {
	if tagNr < 24 {
		return []byte{0xc0 + byte(tagNr)}
	} else if tagNr < 256 {
		return []byte{0xd8, byte(tagNr)}
	} else if tagNr < 65536 {
		return []byte{0xd9, byte(tagNr >> 8), byte(tagNr & 0xff)}
	}
	return []byte{
		0xda,
		byte(tagNr >> 24),
		byte(tagNr >> 16),
		byte(tagNr >> 8),
		byte(tagNr & 0xff),
	}
}

// marshalPlutusMap encodes a PlutusData map into CBOR bytes. It sorts keys
// by hex string for deterministic output. If tagNr is non-zero, the result
// is prefixed with the appropriate CBOR tag bytes.
func marshalPlutusMap(
	m map[serialization.CustomBytes]PlutusData,
	tagNr uint64,
) ([]byte, error) {
	keys := make([]serialization.CustomBytes, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].HexString() < keys[j].HexString()
	})

	lenM := len(m)
	res := make([]byte, 0, 1024)

	// Build map header based on length
	if lenM <= 23 {
		res = append(res, 0xa0+byte(lenM))
	} else if lenM <= 255 {
		res = append(res, 0xb8, byte(lenM))
	} else {
		res = append(res, 0xb9, byte(lenM>>8), byte(lenM&0xff))
	}

	// Encode key/value pairs
	for _, k := range keys {
		keyBytes, err := cbor.Encode(k)
		if err != nil {
			return nil, err
		}
		res = append(res, keyBytes...)
		valBytes, err := cbor.Encode(m[k])
		if err != nil {
			return nil, err
		}
		res = append(res, valBytes...)
	}

	// Prepend tag bytes if needed
	if tagNr != 0 {
		tagBytes := encodeTagBytes(tagNr)
		res = append(tagBytes, res...)
	}

	return res, nil
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
	marshaledThis, err := cbor.Encode(pd)
	if err != nil {
		return false
	}
	marshaledOther, err := cbor.Encode(other)
	if err != nil {
		return false
	}
	return bytes.Equal(marshaledThis, marshaledOther)
}

/*
*

	ToDatum converts a PlutusData to a Datum, in which
	it encodes the PlutusData into CBOR format and later
	into a Datum.

	Returns:
		Datum: The converted Datum.
		error: An error if conversion fails.
*/
func (pd *PlutusData) ToDatum() (Datum, error) {
	var res Datum
	enc, err := cbor.Encode(pd)
	if err != nil {
		return res, err
	}
	_, err = cbor.Decode(enc, &res)
	if err != nil {
		return res, err
	}
	return res, nil
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
	if pd == nil {
		return nil, errors.New("cannot marshal nil PlutusData")
	}
	// If the original raw CBOR bytes were captured during decode, prefer
	// emitting them unchanged to preserve exact-wire encodings used by
	// historical fixtures and tests.
	if len(pd.Raw) > 0 {
		return pd.Raw, nil
	}
	//enc, _ := cbor.CanonicalEncOptions().EncMode()
	switch pd.PlutusDataType {
	case PlutusMap, PlutusIntMap:
		m := pd.Value.(map[serialization.CustomBytes]PlutusData)
		return marshalPlutusMap(m, pd.TagNr)
	case PlutusArray:
		arr := make([]PlutusData, 0)
		if pd.Value != nil {
			switch v := pd.Value.(type) {
			case PlutusDefArray:
				arr = []PlutusData(v)
			case PlutusIndefArray:
				arr = []PlutusData(v)
			default:
				return nil, fmt.Errorf("unexpected type %T", pd.Value)
			}
		}
		var res []byte
		isIndef := false
		if pd.Value != nil {
			_, isIndef = pd.Value.(PlutusIndefArray)
		}
		if isIndef {
			res = append(res, 0x9f)
			for _, v := range arr {
				valBytes, err := v.MarshalCBOR()
				if err != nil {
					return nil, err
				}
				res = append(res, valBytes...)
			}
			res = append(res, 0xff)
		} else {
			length := len(arr)
			if length <= 23 {
				res = append(res, 0x80+byte(length))
			} else if length <= 255 {
				res = append(res, 0x98, byte(length))
			} else {
				res = append(res, 0x99, byte(length>>8), byte(length&0xff))
			}
			for _, v := range arr {
				valBytes, err := v.MarshalCBOR()
				if err != nil {
					return nil, err
				}
				res = append(res, valBytes...)
			}
		}
		if pd.TagNr != 0 {
			tagBytes := encodeTagBytes(pd.TagNr)
			res = append(tagBytes, res...)
		}
		return res, nil
	case PlutusBigInt:
		if pd.TagNr == 0 {
			return cbor.Encode(pd.Value)
		}
		tagBytes := encodeTagBytes(pd.TagNr)
		valBytes, err := cbor.Encode(pd.Value)
		if err != nil {
			return nil, err
		}
		return append(tagBytes, valBytes...), nil
	default:
		//enc, _ := cbor.EncOptions{Sort: cbor.SortCTAP2}.EncMode()
		if pd.TagNr == 0 {
			return cbor.Encode(pd.Value)
		}
		tagBytes := encodeTagBytes(pd.TagNr)
		valBytes, err := cbor.Encode(pd.Value)
		if err != nil {
			return nil, err
		}
		return append(tagBytes, valBytes...), nil
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
			intMap := make(map[serialization.CustomBytes]PlutusData)
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
							cb := serialization.NewCustomBytesInt(int(parsedInt))
							intMap[cb] = pd
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
				pd.Value = intMap
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
		fmt.Println("DEFAULT", x)

	}
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
	bytes, err := cbor.Encode(x)
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
	return HashDatum(pd)
}

/*
*

	HashDatum computes the hash of a CBOR marshaler using the Blake2b algorithm.

	Params:
		d (interface{}): The value to be hashed

	Returns:
		serialization.DatumHash: The hash of the value.
		error: An error if the hash Datum fails.
*/
func HashDatum(d any) (serialization.DatumHash, error) {
	bytes, err := cbor.Encode(d)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return serialization.DatumHash{}, err
	}
	if _, err = hash.Write(bytes); err != nil {
		return serialization.DatumHash{}, err
	}
	r := serialization.DatumHash{Payload: hash.Sum(nil)}
	return r, nil
}

type ScriptHashable interface {
	Hash() (serialization.ScriptHash, error)
}

// hashScript computes a script hash by prepending a version prefix to the
// script bytes and hashing with Blake2b-224.
func hashScript(
	prefix string,
	script []byte,
) (serialization.ScriptHash, error) {
	prefixBytes, err := hex.DecodeString(prefix)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	finalbytes := append(prefixBytes, script...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	if _, err := hash.Write(finalbytes); err != nil {
		return serialization.ScriptHash{}, err
	}
	var r serialization.ScriptHash
	copy(r[:], hash.Sum(nil))
	return r, nil
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
	return hashScript("03", ps3)
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
	return hashScript("01", ps)
}

/*
*

	 	Hash computes the script hash for a PlutusV2Script.

	 	Returns:
	   		serialization.ScriptHash: The script hash of the PlutusV2Script.
			error: An error if the Hashing fails.
*/
func (ps PlutusV2Script) Hash() (serialization.ScriptHash, error) {
	return hashScript("02", ps)
}
