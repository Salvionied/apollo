package PlutusData

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"

	"github.com/Salvionied/cbor/v2"

	"golang.org/x/crypto/blake2b"
)

type _Script struct {
	_      struct{} `cbor:",toarray"`
	Script []byte
}

type _DatumOption struct {
	_     struct{} `cbor:",toarray"`
	datum []byte
}

type ScriptRef struct {
	Script _Script
}

type CostModels map[serialization.CustomBytes]CM

type CM map[string]int

/**
	MarshalCBOR encodes the CM into a CBOR-encoded byte slice, in which
	it serializes the map key alphabetically and encodes the respective values.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if marshaling fails.
*/
func (cm CM) MarshalCBOR() ([]byte, error) {
	res := make([]int, 0)
	mk := make([]string, 0)
	for k, _ := range cm {
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
	"addInteger-cpu-arguments-intercept":                       205665,
	"addInteger-cpu-arguments-slope":                           812,
	"addInteger-memory-arguments-intercept":                    1,
	"addInteger-memory-arguments-slope":                        1,
	"appendByteString-cpu-arguments-intercept":                 1000,
	"appendByteString-cpu-arguments-slope":                     571,
	"appendByteString-memory-arguments-intercept":              0,
	"appendByteString-memory-arguments-slope":                  1,
	"appendString-cpu-arguments-intercept":                     1000,
	"appendString-cpu-arguments-slope":                         24177,
	"appendString-memory-arguments-intercept":                  4,
	"appendString-memory-arguments-slope":                      1,
	"bData-cpu-arguments":                                      1000,
	"bData-memory-arguments":                                   32,
	"blake2b_256-cpu-arguments-intercept":                      117366,
	"blake2b_256-cpu-arguments-slope":                          10475,
	"blake2b_256-memory-arguments":                             4,
	"cekApplyCost-exBudgetCPU":                                 23000,
	"cekApplyCost-exBudgetMemory":                              100,
	"cekBuiltinCost-exBudgetCPU":                               23000,
	"cekBuiltinCost-exBudgetMemory":                            100,
	"cekConstCost-exBudgetCPU":                                 23000,
	"cekConstCost-exBudgetMemory":                              100,
	"cekDelayCost-exBudgetCPU":                                 23000,
	"cekDelayCost-exBudgetMemory":                              100,
	"cekForceCost-exBudgetCPU":                                 23000,
	"cekForceCost-exBudgetMemory":                              100,
	"cekLamCost-exBudgetCPU":                                   23000,
	"cekLamCost-exBudgetMemory":                                100,
	"cekStartupCost-exBudgetCPU":                               100,
	"cekStartupCost-exBudgetMemory":                            100,
	"cekVarCost-exBudgetCPU":                                   23000,
	"cekVarCost-exBudgetMemory":                                100,
	"chooseData-cpu-arguments":                                 19537,
	"chooseData-memory-arguments":                              32,
	"chooseList-cpu-arguments":                                 175354,
	"chooseList-memory-arguments":                              32,
	"chooseUnit-cpu-arguments":                                 46417,
	"chooseUnit-memory-arguments":                              4,
	"consByteString-cpu-arguments-intercept":                   221973,
	"consByteString-cpu-arguments-slope":                       511,
	"consByteString-memory-arguments-intercept":                0,
	"consByteString-memory-arguments-slope":                    1,
	"constrData-cpu-arguments":                                 89141,
	"constrData-memory-arguments":                              32,
	"decodeUtf8-cpu-arguments-intercept":                       497525,
	"decodeUtf8-cpu-arguments-slope":                           14068,
	"decodeUtf8-memory-arguments-intercept":                    4,
	"decodeUtf8-memory-arguments-slope":                        2,
	"divideInteger-cpu-arguments-constant":                     196500,
	"divideInteger-cpu-arguments-model-arguments-intercept":    453240,
	"divideInteger-cpu-arguments-model-arguments-slope":        220,
	"divideInteger-memory-arguments-intercept":                 0,
	"divideInteger-memory-arguments-minimum":                   1,
	"divideInteger-memory-arguments-slope":                     1,
	"encodeUtf8-cpu-arguments-intercept":                       1000,
	"encodeUtf8-cpu-arguments-slope":                           28662,
	"encodeUtf8-memory-arguments-intercept":                    4,
	"encodeUtf8-memory-arguments-slope":                        2,
	"equalsByteString-cpu-arguments-constant":                  245000,
	"equalsByteString-cpu-arguments-intercept":                 216773,
	"equalsByteString-cpu-arguments-slope":                     62,
	"equalsByteString-memory-arguments":                        1,
	"equalsData-cpu-arguments-intercept":                       1060367,
	"equalsData-cpu-arguments-slope":                           12586,
	"equalsData-memory-arguments":                              1,
	"equalsInteger-cpu-arguments-intercept":                    208512,
	"equalsInteger-cpu-arguments-slope":                        421,
	"equalsInteger-memory-arguments":                           1,
	"equalsString-cpu-arguments-constant":                      187000,
	"equalsString-cpu-arguments-intercept":                     1000,
	"equalsString-cpu-arguments-slope":                         52998,
	"equalsString-memory-arguments":                            1,
	"fstPair-cpu-arguments":                                    80436,
	"fstPair-memory-arguments":                                 32,
	"headList-cpu-arguments":                                   43249,
	"headList-memory-arguments":                                32,
	"iData-cpu-arguments":                                      1000,
	"iData-memory-arguments":                                   32,
	"ifThenElse-cpu-arguments":                                 80556,
	"ifThenElse-memory-arguments":                              1,
	"indexByteString-cpu-arguments":                            57667,
	"indexByteString-memory-arguments":                         4,
	"lengthOfByteString-cpu-arguments":                         1000,
	"lengthOfByteString-memory-arguments":                      10,
	"lessThanByteString-cpu-arguments-intercept":               197145,
	"lessThanByteString-cpu-arguments-slope":                   156,
	"lessThanByteString-memory-arguments":                      1,
	"lessThanEqualsByteString-cpu-arguments-intercept":         197145,
	"lessThanEqualsByteString-cpu-arguments-slope":             156,
	"lessThanEqualsByteString-memory-arguments":                1,
	"lessThanEqualsInteger-cpu-arguments-intercept":            204924,
	"lessThanEqualsInteger-cpu-arguments-slope":                473,
	"lessThanEqualsInteger-memory-arguments":                   1,
	"lessThanInteger-cpu-arguments-intercept":                  208896,
	"lessThanInteger-cpu-arguments-slope":                      511,
	"lessThanInteger-memory-arguments":                         1,
	"listData-cpu-arguments":                                   52467,
	"listData-memory-arguments":                                32,
	"mapData-cpu-arguments":                                    64832,
	"mapData-memory-arguments":                                 32,
	"mkCons-cpu-arguments":                                     65493,
	"mkCons-memory-arguments":                                  32,
	"mkNilData-cpu-arguments":                                  22558,
	"mkNilData-memory-arguments":                               32,
	"mkNilPairData-cpu-arguments":                              16563,
	"mkNilPairData-memory-arguments":                           32,
	"mkPairData-cpu-arguments":                                 76511,
	"mkPairData-memory-arguments":                              32,
	"modInteger-cpu-arguments-constant":                        196500,
	"modInteger-cpu-arguments-model-arguments-intercept":       453240,
	"modInteger-cpu-arguments-model-arguments-slope":           220,
	"modInteger-memory-arguments-intercept":                    0,
	"modInteger-memory-arguments-minimum":                      1,
	"modInteger-memory-arguments-slope":                        1,
	"multiplyInteger-cpu-arguments-intercept":                  69522,
	"multiplyInteger-cpu-arguments-slope":                      11687,
	"multiplyInteger-memory-arguments-intercept":               0,
	"multiplyInteger-memory-arguments-slope":                   1,
	"nullList-cpu-arguments":                                   60091,
	"nullList-memory-arguments":                                32,
	"quotientInteger-cpu-arguments-constant":                   196500,
	"quotientInteger-cpu-arguments-model-arguments-intercept":  453240,
	"quotientInteger-cpu-arguments-model-arguments-slope":      220,
	"quotientInteger-memory-arguments-intercept":               0,
	"quotientInteger-memory-arguments-minimum":                 1,
	"quotientInteger-memory-arguments-slope":                   1,
	"remainderInteger-cpu-arguments-constant":                  196500,
	"remainderInteger-cpu-arguments-model-arguments-intercept": 453240,
	"remainderInteger-cpu-arguments-model-arguments-slope":     220,
	"remainderInteger-memory-arguments-intercept":              0,
	"remainderInteger-memory-arguments-minimum":                1,
	"remainderInteger-memory-arguments-slope":                  1,
	"sha2_256-cpu-arguments-intercept":                         806990,
	"sha2_256-cpu-arguments-slope":                             30482,
	"sha2_256-memory-arguments":                                4,
	"sha3_256-cpu-arguments-intercept":                         1927926,
	"sha3_256-cpu-arguments-slope":                             82523,
	"sha3_256-memory-arguments":                                4,
	"sliceByteString-cpu-arguments-intercept":                  265318,
	"sliceByteString-cpu-arguments-slope":                      0,
	"sliceByteString-memory-arguments-intercept":               4,
	"sliceByteString-memory-arguments-slope":                   0,
	"sndPair-cpu-arguments":                                    85931,
	"sndPair-memory-arguments":                                 32,
	"subtractInteger-cpu-arguments-intercept":                  205665,
	"subtractInteger-cpu-arguments-slope":                      812,
	"subtractInteger-memory-arguments-intercept":               1,
	"subtractInteger-memory-arguments-slope":                   1,
	"tailList-cpu-arguments":                                   41182,
	"tailList-memory-arguments":                                32,
	"trace-cpu-arguments":                                      212342,
	"trace-memory-arguments":                                   32,
	"unBData-cpu-arguments":                                    31220,
	"unBData-memory-arguments":                                 32,
	"unConstrData-cpu-arguments":                               32696,
	"unConstrData-memory-arguments":                            32,
	"unIData-cpu-arguments":                                    43357,
	"unIData-memory-arguments":                                 32,
	"unListData-cpu-arguments":                                 32247,
	"unListData-memory-arguments":                              32,
	"unMapData-cpu-arguments":                                  38314,
	"unMapData-memory-arguments":                               32,
	"verifyEd25519Signature-cpu-arguments-intercept":           57996947,
	"verifyEd25519Signature-cpu-arguments-slope":               18975,
	"verifyEd25519Signature-memory-arguments":                  10,
}

type CostView map[string]int

/**
	MarshalCBOR encodes the CostView into a CBOR-encoded byte slice, in which
	it serializes the map key alphabetically and encodes the respective values.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if marshaling fails.
*/
func (cm CostView) MarshalCBOR() ([]byte, error) {
	res := make([]int, 0)
	mk := make([]string, 0)
	for k, _ := range cm {
		mk = append(mk, k)
	}
	sort.Strings(mk)
	for _, v := range mk {
		res = append(res, cm[v])
	}
	return cbor.Marshal(res)

}

var PLUTUSV2COSTMODEL = CostView{
	"addInteger-cpu-arguments-intercept":                       205665,
	"addInteger-cpu-arguments-slope":                           812,
	"addInteger-memory-arguments-intercept":                    1,
	"addInteger-memory-arguments-slope":                        1,
	"appendByteString-cpu-arguments-intercept":                 1000,
	"appendByteString-cpu-arguments-slope":                     571,
	"appendByteString-memory-arguments-intercept":              0,
	"appendByteString-memory-arguments-slope":                  1,
	"appendString-cpu-arguments-intercept":                     1000,
	"appendString-cpu-arguments-slope":                         24177,
	"appendString-memory-arguments-intercept":                  4,
	"appendString-memory-arguments-slope":                      1,
	"bData-cpu-arguments":                                      1000,
	"bData-memory-arguments":                                   32,
	"blake2b_256-cpu-arguments-intercept":                      117366,
	"blake2b_256-cpu-arguments-slope":                          10475,
	"blake2b_256-memory-arguments":                             4,
	"cekApplyCost-exBudgetCPU":                                 23000,
	"cekApplyCost-exBudgetMemory":                              100,
	"cekBuiltinCost-exBudgetCPU":                               23000,
	"cekBuiltinCost-exBudgetMemory":                            100,
	"cekConstCost-exBudgetCPU":                                 23000,
	"cekConstCost-exBudgetMemory":                              100,
	"cekDelayCost-exBudgetCPU":                                 23000,
	"cekDelayCost-exBudgetMemory":                              100,
	"cekForceCost-exBudgetCPU":                                 23000,
	"cekForceCost-exBudgetMemory":                              100,
	"cekLamCost-exBudgetCPU":                                   23000,
	"cekLamCost-exBudgetMemory":                                100,
	"cekStartupCost-exBudgetCPU":                               100,
	"cekStartupCost-exBudgetMemory":                            100,
	"cekVarCost-exBudgetCPU":                                   23000,
	"cekVarCost-exBudgetMemory":                                100,
	"chooseData-cpu-arguments":                                 19537,
	"chooseData-memory-arguments":                              32,
	"chooseList-cpu-arguments":                                 175354,
	"chooseList-memory-arguments":                              32,
	"chooseUnit-cpu-arguments":                                 46417,
	"chooseUnit-memory-arguments":                              4,
	"consByteString-cpu-arguments-intercept":                   221973,
	"consByteString-cpu-arguments-slope":                       511,
	"consByteString-memory-arguments-intercept":                0,
	"consByteString-memory-arguments-slope":                    1,
	"constrData-cpu-arguments":                                 89141,
	"constrData-memory-arguments":                              32,
	"decodeUtf8-cpu-arguments-intercept":                       497525,
	"decodeUtf8-cpu-arguments-slope":                           14068,
	"decodeUtf8-memory-arguments-intercept":                    4,
	"decodeUtf8-memory-arguments-slope":                        2,
	"divideInteger-cpu-arguments-constant":                     196500,
	"divideInteger-cpu-arguments-model-arguments-intercept":    453240,
	"divideInteger-cpu-arguments-model-arguments-slope":        220,
	"divideInteger-memory-arguments-intercept":                 0,
	"divideInteger-memory-arguments-minimum":                   1,
	"divideInteger-memory-arguments-slope":                     1,
	"encodeUtf8-cpu-arguments-intercept":                       1000,
	"encodeUtf8-cpu-arguments-slope":                           28662,
	"encodeUtf8-memory-arguments-intercept":                    4,
	"encodeUtf8-memory-arguments-slope":                        2,
	"equalsByteString-cpu-arguments-constant":                  245000,
	"equalsByteString-cpu-arguments-intercept":                 216773,
	"equalsByteString-cpu-arguments-slope":                     62,
	"equalsByteString-memory-arguments":                        1,
	"equalsData-cpu-arguments-intercept":                       1060367,
	"equalsData-cpu-arguments-slope":                           12586,
	"equalsData-memory-arguments":                              1,
	"equalsInteger-cpu-arguments-intercept":                    208512,
	"equalsInteger-cpu-arguments-slope":                        421,
	"equalsInteger-memory-arguments":                           1,
	"equalsString-cpu-arguments-constant":                      187000,
	"equalsString-cpu-arguments-intercept":                     1000,
	"equalsString-cpu-arguments-slope":                         52998,
	"equalsString-memory-arguments":                            1,
	"fstPair-cpu-arguments":                                    80436,
	"fstPair-memory-arguments":                                 32,
	"headList-cpu-arguments":                                   43249,
	"headList-memory-arguments":                                32,
	"iData-cpu-arguments":                                      1000,
	"iData-memory-arguments":                                   32,
	"ifThenElse-cpu-arguments":                                 80556,
	"ifThenElse-memory-arguments":                              1,
	"indexByteString-cpu-arguments":                            57667,
	"indexByteString-memory-arguments":                         4,
	"lengthOfByteString-cpu-arguments":                         1000,
	"lengthOfByteString-memory-arguments":                      10,
	"lessThanByteString-cpu-arguments-intercept":               197145,
	"lessThanByteString-cpu-arguments-slope":                   156,
	"lessThanByteString-memory-arguments":                      1,
	"lessThanEqualsByteString-cpu-arguments-intercept":         197145,
	"lessThanEqualsByteString-cpu-arguments-slope":             156,
	"lessThanEqualsByteString-memory-arguments":                1,
	"lessThanEqualsInteger-cpu-arguments-intercept":            204924,
	"lessThanEqualsInteger-cpu-arguments-slope":                473,
	"lessThanEqualsInteger-memory-arguments":                   1,
	"lessThanInteger-cpu-arguments-intercept":                  208896,
	"lessThanInteger-cpu-arguments-slope":                      511,
	"lessThanInteger-memory-arguments":                         1,
	"listData-cpu-arguments":                                   52467,
	"listData-memory-arguments":                                32,
	"mapData-cpu-arguments":                                    64832,
	"mapData-memory-arguments":                                 32,
	"mkCons-cpu-arguments":                                     65493,
	"mkCons-memory-arguments":                                  32,
	"mkNilData-cpu-arguments":                                  22558,
	"mkNilData-memory-arguments":                               32,
	"mkNilPairData-cpu-arguments":                              16563,
	"mkNilPairData-memory-arguments":                           32,
	"mkPairData-cpu-arguments":                                 76511,
	"mkPairData-memory-arguments":                              32,
	"modInteger-cpu-arguments-constant":                        196500,
	"modInteger-cpu-arguments-model-arguments-intercept":       453240,
	"modInteger-cpu-arguments-model-arguments-slope":           220,
	"modInteger-memory-arguments-intercept":                    0,
	"modInteger-memory-arguments-minimum":                      1,
	"modInteger-memory-arguments-slope":                        1,
	"multiplyInteger-cpu-arguments-intercept":                  69522,
	"multiplyInteger-cpu-arguments-slope":                      11687,
	"multiplyInteger-memory-arguments-intercept":               0,
	"multiplyInteger-memory-arguments-slope":                   1,
	"nullList-cpu-arguments":                                   60091,
	"nullList-memory-arguments":                                32,
	"quotientInteger-cpu-arguments-constant":                   196500,
	"quotientInteger-cpu-arguments-model-arguments-intercept":  453240,
	"quotientInteger-cpu-arguments-model-arguments-slope":      220,
	"quotientInteger-memory-arguments-intercept":               0,
	"quotientInteger-memory-arguments-minimum":                 1,
	"quotientInteger-memory-arguments-slope":                   1,
	"remainderInteger-cpu-arguments-constant":                  196500,
	"remainderInteger-cpu-arguments-model-arguments-intercept": 453240,
	"remainderInteger-cpu-arguments-model-arguments-slope":     220,
	"remainderInteger-memory-arguments-intercept":              0,
	"remainderInteger-memory-arguments-minimum":                1,
	"remainderInteger-memory-arguments-slope":                  1,
	"serialiseData-cpu-arguments-intercept":                    1159724,
	"serialiseData-cpu-arguments-slope":                        392670,
	"serialiseData-memory-arguments-intercept":                 0,
	"serialiseData-memory-arguments-slope":                     2,
	"sha2_256-cpu-arguments-intercept":                         806990,
	"sha2_256-cpu-arguments-slope":                             30482,
	"sha2_256-memory-arguments":                                4,
	"sha3_256-cpu-arguments-intercept":                         1927926,
	"sha3_256-cpu-arguments-slope":                             82523,
	"sha3_256-memory-arguments":                                4,
	"sliceByteString-cpu-arguments-intercept":                  265318,
	"sliceByteString-cpu-arguments-slope":                      0,
	"sliceByteString-memory-arguments-intercept":               4,
	"sliceByteString-memory-arguments-slope":                   0,
	"sndPair-cpu-arguments":                                    85931,
	"sndPair-memory-arguments":                                 32,
	"subtractInteger-cpu-arguments-intercept":                  205665,
	"subtractInteger-cpu-arguments-slope":                      812,
	"subtractInteger-memory-arguments-intercept":               1,
	"subtractInteger-memory-arguments-slope":                   1,
	"tailList-cpu-arguments":                                   41182,
	"tailList-memory-arguments":                                32,
	"trace-cpu-arguments":                                      212342,
	"trace-memory-arguments":                                   32,
	"unBData-cpu-arguments":                                    31220,
	"unBData-memory-arguments":                                 32,
	"unConstrData-cpu-arguments":                               32696,
	"unConstrData-memory-arguments":                            32,
	"unIData-cpu-arguments":                                    43357,
	"unIData-memory-arguments":                                 32,
	"unListData-cpu-arguments":                                 32247,
	"unListData-memory-arguments":                              32,
	"unMapData-cpu-arguments":                                  38314,
	"unMapData-memory-arguments":                               32,
	"verifyEcdsaSecp256k1Signature-cpu-arguments":              35892428,
	"verifyEcdsaSecp256k1Signature-memory-arguments":           10,
	"verifyEd25519Signature-cpu-arguments-intercept":           57996947,
	"verifyEd25519Signature-cpu-arguments-slope":               18975,
	"verifyEd25519Signature-memory-arguments":                  10,
	"verifySchnorrSecp256k1Signature-cpu-arguments-intercept":  38887044,
	"verifySchnorrSecp256k1Signature-cpu-arguments-slope":      32947,
	"verifySchnorrSecp256k1Signature-memory-arguments":         10,
}

var COST_MODELSV2 = map[int]cbor.Marshaler{1: PLUTUSV2COSTMODEL}
var COST_MODELSV1 = map[serialization.CustomBytes]cbor.Marshaler{{Value: "00"}: PLUTUSV1COSTMODEL}

type PlutusType int

const (
	PlutusArray PlutusType = iota
	PlutusMap
	PlutusInt
	PlutusBytes
	PlutusShortArray
)

type PlutusList interface {
	Len() int
}

type PlutusIndefArray []PlutusData
type PlutusDefArray []PlutusData

/**
	Len returns the length of the PlutusIndefArray.

	Returns:
		int: The length of the PlutusIndefArray.
*/
func (pia PlutusIndefArray) Len() int {
	return len(pia)
}

/**
	Len returns the length of the PlutusDefArray.

	Returns:
		int: The length of the PlutusDefArray.
*/
func (pia PlutusDefArray) Len() int {
	return len(pia)
}

/**
	Clone creates a deep copy of the PlutusIndefArray.

	Returns:
		PlutusIndefArray: A deep copy of the PlutusIndefArray.
*/
func (pia *PlutusIndefArray) Clone() PlutusIndefArray {
	var ret PlutusIndefArray
	for _, v := range *pia {
		ret = append(ret, v.Clone())
	}
	return ret
}

/**
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
			log.Fatal(err)
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

/**
	ToPlutusData converts a datum to PlutusData, encoding
	the Datum into CBOR format and then decodes it into a
	PlutusData.

	Returns:
		PlutusData: The converted PlutusData.
*/
func (pd *Datum) ToPlutusData() PlutusData {
	var res PlutusData
	enc, _ := cbor.Marshal(pd)
	cbor.Unmarshal(enc, &res)
	return res
}

/**
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

/**
	MarshalCBOR encodes the Datum into a CBOR-encoded byte slice,
	it applies a CBOR tag, if TagNr is not 0, otherwise it marshals the Value
 	
	Returns:
   		[]uint8: The CBOR-encoded byte slice.
   		error: An error if marshaling fails.
*/
func (pd Datum) MarshalCBOR() ([]uint8, error) {
	if pd.TagNr == 0 {
		return cbor.Marshal(pd.Value)
	} else {
		return cbor.Marshal(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
	}
}

/**
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
		case []interface{}:
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
		case []interface{}:
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

		case map[interface{}]interface{}:
			y := new(map[serialization.CustomBytes]Datum)
			err = cbor.Unmarshal(value, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusMap
			pd.Value = y
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

/**
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

/**
	ToDatum converts a PlutusData to a Datum, in which
	it encodes the PlutusData into CBOR format and later
	into a Datum.

	Returns:
		Datum: The converted Datum.
*/
func (pd *PlutusData) ToDatum() Datum {

	var res Datum
	enc, _ := cbor.Marshal(pd)
	cbor.Unmarshal(enc, &res)
	return res
}

/**
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

/**
	MarshalCBOR encodes the PlutusData into a CBOR byte slice.

	Returns:
		[]uint8: The CBOR-encoded byte slice.
		error: An error, if any, during ecoding.
*/
func (pd *PlutusData) MarshalCBOR() ([]uint8, error) {
	if pd.TagNr == 0 {
		return cbor.Marshal(pd.Value)
	} else {
		return cbor.Marshal(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
	}
}

/**
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
	case []interface{}:
		y := new([]PlutusData)
		err = json.Unmarshal(value, y)
		if err != nil {
			return err
		}
		pd.PlutusDataType = PlutusArray
		pd.Value = PlutusIndefArray(*y)
		pd.TagNr = 0
	case map[string]interface{}:
		val := x.(map[string]interface{})
		_, ok := val["fields"]
		if ok {
			contents, _ := json.Marshal(val["fields"])
			var tag int
			constructor, ok := val["constructor"]
			if ok {
				tag = int(121 + constructor.(float64))
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
		} else if _, ok := val["bytes"]; ok {
			pd.PlutusDataType = PlutusBytes
			pd.Value, _ = hex.DecodeString(val["bytes"].(string))
		} else if _, ok := val["int"]; ok {
			pd.PlutusDataType = PlutusInt
			pd.Value = uint64(val["int"].(float64))
		} else {
			fmt.Println("Invalid Nested Struct in plutus data")
		}
	}
	return nil
}

/**
	UnmarshalCBOR unmarshals CBOR-encoded data into a PlutusData object.

	Params:
   		value ([]uint8): The CBOR-encoded data to unmarshal.

 	Returns:
   		error: An error, if any, during unmarshaling.
*/
func (pd *PlutusData) UnmarshalCBOR(value []uint8) error {
	var x any
	err := cbor.Unmarshal(value, &x)
	if err != nil {
		return err
	}
	ok, valid := x.(cbor.Tag)
	if valid {
		switch ok.Content.(type) {
		case []interface{}:
			pd.TagNr = ok.Number
			pd.PlutusDataType = PlutusArray
			if value[2] == 0x9f {
				y := PlutusIndefArray{}
				err = cbor.Unmarshal(value[2:], &y)
				if err != nil {
					return err
				}
				pd.Value = y
			} else {
				y := PlutusDefArray{}
				err = cbor.Unmarshal(value[2:], &y)
				if err != nil {
					return err
				}
				pd.Value = y
			}
		case []uint8:
			pd.TagNr = ok.Number
			pd.PlutusDataType = PlutusBytes
			pd.Value = ok.Content

		default:
			//TODO SKIP
			return nil
		}
	} else {
		switch x.(type) {
		case []interface{}:
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

		case map[interface{}]interface{}:
			y := new(map[serialization.CustomBytes]PlutusData)
			err = cbor.Unmarshal(value, y)
			if err != nil {
				return err
			}
			pd.PlutusDataType = PlutusMap
			pd.Value = y
			pd.TagNr = 0

		default:
			fmt.Println("Invalid Nested Struct in plutus data", reflect.TypeOf(x))
		}

	}

	return nil
}

type RawPlutusData struct {
	//TODO
}

/**
	ToCbor converts the given interface to a hexadecimal-encoded CBOR string.

	Params:
		x (interface{}): The input value to be encoded to CBOR to converted
						 to a hexadecimal string.
	
	Returns:
		string: The hexadecimal-encoded CBOR representation of the input value.
*/
func ToCbor(x interface{}) string {
	bytes, err := cbor.Marshal(x)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

/**
	PlutusDataHash computes the hash of a PlutusData structure using the Blake2b algorithm.

 	Params:
	   	pd (*PlutusData): A pointer to the PlutusData structure to be hashed.

	Returns:
  		serialization.DatumHash: The hash of the PlutusData.
*/
func PlutusDataHash(pd *PlutusData) serialization.DatumHash {
	finalbytes := []byte{}
	bytes, err := cbor.Marshal(pd)
	if err != nil {
		log.Fatal(err)
	}
	finalbytes = append(finalbytes, bytes...)
	hash, err := blake2b.New(32, nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		log.Fatal(err)
	}
	r := serialization.DatumHash{hash.Sum(nil)}
	return r
}

/**
	HashDatum computes the hash of a CBOR marshaler using the Blake2b algorithm.

	Params:
		d (cbor.Marshaler): The CBOR marshaler to be hashed

	Returns:
		serialization.DatumHash: The hash of the CBOR marshaler.
*/
func HashDatum(d cbor.Marshaler) serialization.DatumHash {
	finalbytes := []byte{}
	bytes, err := cbor.Marshal(d)
	if err != nil {
		log.Fatal(err)
	}
	finalbytes = append(finalbytes, bytes...)
	hash, err := blake2b.New(32, nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		log.Fatal(err)
	}
	r := serialization.DatumHash{hash.Sum(nil)}
	return r
}

type ScriptHashable interface {
	Hash() serialization.ScriptHash
}

/**
	PlutusScriptHash computes the script hash of a ScriptHashable object.

	Params:
		script (ScriptHashable): The ScriptHashable object to be hashed.

	Returns:
		serialization.ScriptHash: The script hash of the ScriptHashable object.
*/
func PlutusScriptHash(script ScriptHashable) serialization.ScriptHash {
	return script.Hash()
}

type PlutusV1Script []byte

/**
	ToAddress converts a PlutusV1Script to an Address with an optional staking credential.

 	Params:
   		stakingCredential ([]byte): The staking credential to include in the address.

 	Returns:
   		Address.Address: The generated address.
*/
func (ps *PlutusV1Script) ToAddress(stakingCredential []byte) Address.Address {
	hash := PlutusScriptHash(ps)
	if stakingCredential == nil {
		return Address.Address{hash.Bytes(), nil, Address.MAINNET, Address.SCRIPT_NONE, 0b01110001, "addr"}
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

/**
	ToAddress converts a PlutusV2Script to an Address with an optional staking credential.

 	Params:
   		stakingCredential ([]byte): The staking credential to include in the address.

 	Returns:
   		Address.Address: The generated address.
*/
func (ps *PlutusV2Script) ToAddress(stakingCredential []byte) Address.Address {
	hash := PlutusScriptHash(ps)
	if stakingCredential == nil {
		return Address.Address{hash.Bytes(), nil, Address.MAINNET, Address.SCRIPT_NONE, 0b01110001, "addr"}
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

/**
 	Hash computes the script hash for a PlutusV1Script.

 	Returns:
   		serialization.ScriptHash: The script hash of the PlutusV1Script.
*/
func (ps PlutusV1Script) Hash() serialization.ScriptHash {
	finalbytes, err := hex.DecodeString("01")
	if err != nil {
		log.Fatal(err)
	}
	finalbytes = append(finalbytes, ps...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		log.Fatal(err)
	}
	r := serialization.ScriptHash{}
	copy(r[:], hash.Sum(nil))
	return r
}

/**
 	Hash computes the script hash for a PlutusV2Script.

 	Returns:
   		serialization.ScriptHash: The script hash of the PlutusV2Script.
*/
func (ps PlutusV2Script) Hash() serialization.ScriptHash {
	finalbytes, err := hex.DecodeString("02")
	if err != nil {
		log.Fatal(err)
	}
	finalbytes = append(finalbytes, ps...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		log.Fatal(err)
	}
	r := serialization.ScriptHash{}
	copy(r[:], hash.Sum(nil))
	return r
}
