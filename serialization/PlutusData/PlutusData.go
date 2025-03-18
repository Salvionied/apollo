package PlutusData

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"

	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"

	"github.com/Salvionied/cbor/v2"

	"golang.org/x/crypto/blake2b"
)

type CostModel []int

type InnerScript struct {
	_      struct{} `cbor:",toarray"`
	Script []byte
}

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
	if cborDatumOption.DatumType == DatumTypeInline {
		var cborDatumInline PlutusData
		errInline := cbor.Unmarshal(cborDatumOption.Content, &cborDatumInline)
		if errInline != nil {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", errInline)
		}
		if cborDatumInline.TagNr != 24 {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: DatumTypeInline but Tag was not 24: %v", cborDatumInline.TagNr)
		}
		taggedBytes, valid := cborDatumInline.Value.([]byte)
		if !valid {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: found tag 24 but there wasn't a byte array")
		}
		var inline PlutusData
		err = cbor.Unmarshal(taggedBytes, &inline)
		if err != nil {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", err)
		}
		d.DatumType = DatumTypeInline
		d.Inline = &inline
		return nil
	} else if cborDatumOption.DatumType == DatumTypeHash {
		var cborDatumHash []byte
		errHash := cbor.Unmarshal(cborDatumOption.Content, &cborDatumHash)
		if errHash != nil {
			return fmt.Errorf("DatumOption: UnmarshalCBOR: %v", errHash)
		}
		d.DatumType = DatumTypeHash
		d.Hash = cborDatumHash
		return nil
	} else {
		return fmt.Errorf("DatumOption: UnmarshalCBOR: Unknown tag: %v", cborDatumOption.DatumType)
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
			return nil, fmt.Errorf("DatumOption: MarshalCBOR(): Failed to marshal inline datum: %v", err)
		}
		format.Content = &PlutusData{
			PlutusDataType: PlutusBytes,
			TagNr:          24,
			Value:          bytes,
		}
	default:
		return nil, fmt.Errorf("Invalid DatumOption: %v", d)
	}
	return cbor.Marshal(format)
}

type ScriptRef struct {
	Script InnerScript
}

type CM map[string]int

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

var V1COSTMODELKEYS = []string{
	"addInteger-cpu-arguments-intercept",
	"addInteger-cpu-arguments-slope",
	"addInteger-memory-arguments-intercept",
	"addInteger-memory-arguments-slope",
	"appendByteString-cpu-arguments-intercept",
	"appendByteString-cpu-arguments-slope",
	"appendByteString-memory-arguments-intercept",
	"appendByteString-memory-arguments-slope",
	"appendString-cpu-arguments-intercept",
	"appendString-cpu-arguments-slope",
	"appendString-memory-arguments-intercept",
	"appendString-memory-arguments-slope",
	"bData-cpu-arguments",
	"bData-memory-arguments",
	"blake2b_256-cpu-arguments-intercept",
	"blake2b_256-cpu-arguments-slope",
	"blake2b_256-memory-arguments",
	"cekApplyCost-exBudgetCPU",
	"cekApplyCost-exBudgetMemory",
	"cekBuiltinCost-exBudgetCPU",
	"cekBuiltinCost-exBudgetMemory",
	"cekConstCost-exBudgetCPU",
	"cekConstCost-exBudgetMemory",
	"cekDelayCost-exBudgetCPU",
	"cekDelayCost-exBudgetMemory",
	"cekForceCost-exBudgetCPU",
	"cekForceCost-exBudgetMemory",
	"cekLamCost-exBudgetCPU",
	"cekLamCost-exBudgetMemory",
	"cekStartupCost-exBudgetCPU",
	"cekStartupCost-exBudgetMemory",
	"cekVarCost-exBudgetCPU",
	"cekVarCost-exBudgetMemory",
	"chooseData-cpu-arguments",
	"chooseData-memory-arguments",
	"chooseList-cpu-arguments",
	"chooseList-memory-arguments",
	"chooseUnit-cpu-arguments",
	"chooseUnit-memory-arguments",
	"consByteString-cpu-arguments-intercept",
	"consByteString-cpu-arguments-slope",
	"consByteString-memory-arguments-intercept",
	"consByteString-memory-arguments-slope",
	"constrData-cpu-arguments",
	"constrData-memory-arguments",
	"decodeUtf8-cpu-arguments-intercept",
	"decodeUtf8-cpu-arguments-slope",
	"decodeUtf8-memory-arguments-intercept",
	"decodeUtf8-memory-arguments-slope",
	"divideInteger-cpu-arguments-constant",
	"divideInteger-cpu-arguments-model-arguments-intercept",
	"divideInteger-cpu-arguments-model-arguments-slope",
	"divideInteger-memory-arguments-intercept",
	"divideInteger-memory-arguments-minimum",
	"divideInteger-memory-arguments-slope",
	"encodeUtf8-cpu-arguments-intercept",
	"encodeUtf8-cpu-arguments-slope",
	"encodeUtf8-memory-arguments-intercept",
	"encodeUtf8-memory-arguments-slope",
	"equalsByteString-cpu-arguments-constant",
	"equalsByteString-cpu-arguments-intercept",
	"equalsByteString-cpu-arguments-slope",
	"equalsByteString-memory-arguments",
	"equalsData-cpu-arguments-intercept",
	"equalsData-cpu-arguments-slope",
	"equalsData-memory-arguments",
	"equalsInteger-cpu-arguments-intercept",
	"equalsInteger-cpu-arguments-slope",
	"equalsInteger-memory-arguments",
	"equalsString-cpu-arguments-constant",
	"equalsString-cpu-arguments-intercept",
	"equalsString-cpu-arguments-slope",
	"equalsString-memory-arguments",
	"fstPair-cpu-arguments",
	"fstPair-memory-arguments",
	"headList-cpu-arguments",
	"headList-memory-arguments",
	"iData-cpu-arguments",
	"iData-memory-arguments",
	"ifThenElse-cpu-arguments",
	"ifThenElse-memory-arguments",
	"indexByteString-cpu-arguments",
	"indexByteString-memory-arguments",
	"lengthOfByteString-cpu-arguments",
	"lengthOfByteString-memory-arguments",
	"lessThanByteString-cpu-arguments-intercept",
	"lessThanByteString-cpu-arguments-slope",
	"lessThanByteString-memory-arguments",
	"lessThanEqualsByteString-cpu-arguments-intercept",
	"lessThanEqualsByteString-cpu-arguments-slope",
	"lessThanEqualsByteString-memory-arguments",
	"lessThanEqualsInteger-cpu-arguments-intercept",
	"lessThanEqualsInteger-cpu-arguments-slope",
	"lessThanEqualsInteger-memory-arguments",
	"lessThanInteger-cpu-arguments-intercept",
	"lessThanInteger-cpu-arguments-slope",
	"lessThanInteger-memory-arguments",
	"listData-cpu-arguments",
	"listData-memory-arguments",
	"mapData-cpu-arguments",
	"mapData-memory-arguments",
	"mkCons-cpu-arguments",
	"mkCons-memory-arguments",
	"mkNilData-cpu-arguments",
	"mkNilData-memory-arguments",
	"mkNilPairData-cpu-arguments",
	"mkNilPairData-memory-arguments",
	"mkPairData-cpu-arguments",
	"mkPairData-memory-arguments",
	"modInteger-cpu-arguments-constant",
	"modInteger-cpu-arguments-model-arguments-intercept",
	"modInteger-cpu-arguments-model-arguments-slope",
	"modInteger-memory-arguments-intercept",
	"modInteger-memory-arguments-minimum",
	"modInteger-memory-arguments-slope",
	"multiplyInteger-cpu-arguments-intercept",
	"multiplyInteger-cpu-arguments-slope",
	"multiplyInteger-memory-arguments-intercept",
	"multiplyInteger-memory-arguments-slope",
	"nullList-cpu-arguments",
	"nullList-memory-arguments",
	"quotientInteger-cpu-arguments-constant",
	"quotientInteger-cpu-arguments-model-arguments-intercept",
	"quotientInteger-cpu-arguments-model-arguments-slope",
	"quotientInteger-memory-arguments-intercept",
	"quotientInteger-memory-arguments-minimum",
	"quotientInteger-memory-arguments-slope",
	"remainderInteger-cpu-arguments-constant",
	"remainderInteger-cpu-arguments-model-arguments-intercept",
	"remainderInteger-cpu-arguments-model-arguments-slope",
	"remainderInteger-memory-arguments-intercept",
	"remainderInteger-memory-arguments-minimum",
	"remainderInteger-memory-arguments-slope",
	"sha2_256-cpu-arguments-intercept",
	"sha2_256-cpu-arguments-slope",
	"sha2_256-memory-arguments",
	"sha3_256-cpu-arguments-intercept",
	"sha3_256-cpu-arguments-slope",
	"sha3_256-memory-arguments",
	"sliceByteString-cpu-arguments-intercept",
	"sliceByteString-cpu-arguments-slope",
	"sliceByteString-memory-arguments-intercept",
	"sliceByteString-memory-arguments-slope",
	"sndPair-cpu-arguments",
	"sndPair-memory-arguments",
	"subtractInteger-cpu-arguments-intercept",
	"subtractInteger-cpu-arguments-slope",
	"subtractInteger-memory-arguments-intercept",
	"subtractInteger-memory-arguments-slope",
	"tailList-cpu-arguments",
	"tailList-memory-arguments",
	"trace-cpu-arguments",
	"trace-memory-arguments",
	"unBData-cpu-arguments",
	"unBData-memory-arguments",
	"unConstrData-cpu-arguments",
	"unConstrData-memory-arguments",
	"unIData-cpu-arguments",
	"unIData-memory-arguments",
	"unListData-cpu-arguments",
	"unListData-memory-arguments",
	"unMapData-cpu-arguments",
	"unMapData-memory-arguments",
	"verifyEd25519Signature-cpu-arguments-intercept",
	"verifyEd25519Signature-cpu-arguments-slope",
	"verifyEd25519Signature-memory-arguments",
}

type CostView map[string]int

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

var V2COSTMODELKEYS = []string{
	"addInteger-cpu-arguments-intercept",
	"addInteger-cpu-arguments-slope",
	"addInteger-memory-arguments-intercept",
	"addInteger-memory-arguments-slope",
	"appendByteString-cpu-arguments-intercept",
	"appendByteString-cpu-arguments-slope",
	"appendByteString-memory-arguments-intercept",
	"appendByteString-memory-arguments-slope",
	"appendString-cpu-arguments-intercept",
	"appendString-cpu-arguments-slope",
	"appendString-memory-arguments-intercept",
	"appendString-memory-arguments-slope",
	"bData-cpu-arguments",
	"bData-memory-arguments",
	"blake2b_256-cpu-arguments-intercept",
	"blake2b_256-cpu-arguments-slope",
	"blake2b_256-memory-arguments",
	"cekApplyCost-exBudgetCPU",
	"cekApplyCost-exBudgetMemory",
	"cekBuiltinCost-exBudgetCPU",
	"cekBuiltinCost-exBudgetMemory",
	"cekConstCost-exBudgetCPU",
	"cekConstCost-exBudgetMemory",
	"cekDelayCost-exBudgetCPU",
	"cekDelayCost-exBudgetMemory",
	"cekForceCost-exBudgetCPU",
	"cekForceCost-exBudgetMemory",
	"cekLamCost-exBudgetCPU",
	"cekLamCost-exBudgetMemory",
	"cekStartupCost-exBudgetCPU",
	"cekStartupCost-exBudgetMemory",
	"cekVarCost-exBudgetCPU",
	"cekVarCost-exBudgetMemory",
	"chooseData-cpu-arguments",
	"chooseData-memory-arguments",
	"chooseList-cpu-arguments",
	"chooseList-memory-arguments",
	"chooseUnit-cpu-arguments",
	"chooseUnit-memory-arguments",
	"consByteString-cpu-arguments-intercept",
	"consByteString-cpu-arguments-slope",
	"consByteString-memory-arguments-intercept",
	"consByteString-memory-arguments-slope",
	"constrData-cpu-arguments",
	"constrData-memory-arguments",
	"decodeUtf8-cpu-arguments-intercept",
	"decodeUtf8-cpu-arguments-slope",
	"decodeUtf8-memory-arguments-intercept",
	"decodeUtf8-memory-arguments-slope",
	"divideInteger-cpu-arguments-constant",
	"divideInteger-cpu-arguments-model-arguments-intercept",
	"divideInteger-cpu-arguments-model-arguments-slope",
	"divideInteger-memory-arguments-intercept",
	"divideInteger-memory-arguments-minimum",
	"divideInteger-memory-arguments-slope",
	"encodeUtf8-cpu-arguments-intercept",
	"encodeUtf8-cpu-arguments-slope",
	"encodeUtf8-memory-arguments-intercept",
	"encodeUtf8-memory-arguments-slope",
	"equalsByteString-cpu-arguments-constant",
	"equalsByteString-cpu-arguments-intercept",
	"equalsByteString-cpu-arguments-slope",
	"equalsByteString-memory-arguments",
	"equalsData-cpu-arguments-intercept",
	"equalsData-cpu-arguments-slope",
	"equalsData-memory-arguments",
	"equalsInteger-cpu-arguments-intercept",
	"equalsInteger-cpu-arguments-slope",
	"equalsInteger-memory-arguments",
	"equalsString-cpu-arguments-constant",
	"equalsString-cpu-arguments-intercept",
	"equalsString-cpu-arguments-slope",
	"equalsString-memory-arguments",
	"fstPair-cpu-arguments",
	"fstPair-memory-arguments",
	"headList-cpu-arguments",
	"headList-memory-arguments",
	"iData-cpu-arguments",
	"iData-memory-arguments",
	"ifThenElse-cpu-arguments",
	"ifThenElse-memory-arguments",
	"indexByteString-cpu-arguments",
	"indexByteString-memory-arguments",
	"lengthOfByteString-cpu-arguments",
	"lengthOfByteString-memory-arguments",
	"lessThanByteString-cpu-arguments-intercept",
	"lessThanByteString-cpu-arguments-slope",
	"lessThanByteString-memory-arguments",
	"lessThanEqualsByteString-cpu-arguments-intercept",
	"lessThanEqualsByteString-cpu-arguments-slope",
	"lessThanEqualsByteString-memory-arguments",
	"lessThanEqualsInteger-cpu-arguments-intercept",
	"lessThanEqualsInteger-cpu-arguments-slope",
	"lessThanEqualsInteger-memory-arguments",
	"lessThanInteger-cpu-arguments-intercept",
	"lessThanInteger-cpu-arguments-slope",
	"lessThanInteger-memory-arguments",
	"listData-cpu-arguments",
	"listData-memory-arguments",
	"mapData-cpu-arguments",
	"mapData-memory-arguments",
	"mkCons-cpu-arguments",
	"mkCons-memory-arguments",
	"mkNilData-cpu-arguments",
	"mkNilData-memory-arguments",
	"mkNilPairData-cpu-arguments",
	"mkNilPairData-memory-arguments",
	"mkPairData-cpu-arguments",
	"mkPairData-memory-arguments",
	"modInteger-cpu-arguments-constant",
	"modInteger-cpu-arguments-model-arguments-intercept",
	"modInteger-cpu-arguments-model-arguments-slope",
	"modInteger-memory-arguments-intercept",
	"modInteger-memory-arguments-minimum",
	"modInteger-memory-arguments-slope",
	"multiplyInteger-cpu-arguments-intercept",
	"multiplyInteger-cpu-arguments-slope",
	"multiplyInteger-memory-arguments-intercept",
	"multiplyInteger-memory-arguments-slope",
	"nullList-cpu-arguments",
	"nullList-memory-arguments",
	"quotientInteger-cpu-arguments-constant",
	"quotientInteger-cpu-arguments-model-arguments-intercept",
	"quotientInteger-cpu-arguments-model-arguments-slope",
	"quotientInteger-memory-arguments-intercept",
	"quotientInteger-memory-arguments-minimum",
	"quotientInteger-memory-arguments-slope",
	"remainderInteger-cpu-arguments-constant",
	"remainderInteger-cpu-arguments-model-arguments-intercept",
	"remainderInteger-cpu-arguments-model-arguments-slope",
	"remainderInteger-memory-arguments-intercept",
	"remainderInteger-memory-arguments-minimum",
	"remainderInteger-memory-arguments-slope",
	"serialiseData-cpu-arguments-intercept",
	"serialiseData-cpu-arguments-slope",
	"serialiseData-memory-arguments-intercept",
	"serialiseData-memory-arguments-slope",
	"sha2_256-cpu-arguments-intercept",
	"sha2_256-cpu-arguments-slope",
	"sha2_256-memory-arguments",
	"sha3_256-cpu-arguments-intercept",
	"sha3_256-cpu-arguments-slope",
	"sha3_256-memory-arguments",
	"sliceByteString-cpu-arguments-intercept",
	"sliceByteString-cpu-arguments-slope",
	"sliceByteString-memory-arguments-intercept",
	"sliceByteString-memory-arguments-slope",
	"sndPair-cpu-arguments",
	"sndPair-memory-arguments",
	"subtractInteger-cpu-arguments-intercept",
	"subtractInteger-cpu-arguments-slope",
	"subtractInteger-memory-arguments-intercept",
	"subtractInteger-memory-arguments-slope",
	"tailList-cpu-arguments",
	"tailList-memory-arguments",
	"trace-cpu-arguments",
	"trace-memory-arguments",
	"unBData-cpu-arguments",
	"unBData-memory-arguments",
	"unConstrData-cpu-arguments",
	"unConstrData-memory-arguments",
	"unIData-cpu-arguments",
	"unIData-memory-arguments",
	"unListData-cpu-arguments",
	"unListData-memory-arguments",
	"unMapData-cpu-arguments",
	"unMapData-memory-arguments",
	"verifyEcdsaSecp256k1Signature-cpu-arguments",
	"verifyEcdsaSecp256k1Signature-memory-arguments",
	"verifyEd25519Signature-cpu-arguments-intercept",
	"verifyEd25519Signature-cpu-arguments-slope",
	"verifyEd25519Signature-memory-arguments",
	"verifySchnorrSecp256k1Signature-cpu-arguments-intercept",
	"verifySchnorrSecp256k1Signature-cpu-arguments-slope",
	"verifySchnorrSecp256k1Signature-memory-arguments",
}

func CostModelV2(cm CostModel) cbor.Marshaler {
	cost := make(map[string]int)
	for ix, s := range V2COSTMODELKEYS {
		cost[s] = cm[ix]
	}
	return CostView(cost)
}

func CostModelV1(cm CostModel) cbor.Marshaler {
	cost := make(map[string]int)
	for ix, s := range V1COSTMODELKEYS {
		cost[s] = cm[ix]
	}
	return CM(cost)
}

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

func (pia PlutusIndefArray) Len() int {
	return len(pia)
}

func (pia PlutusDefArray) Len() int {
	return len(pia)
}

func (pia *PlutusIndefArray) Clone() PlutusIndefArray {
	var ret PlutusIndefArray
	for _, v := range *pia {
		ret = append(ret, v.Clone())
	}
	return ret
}

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

func (pd *Datum) ToPlutusData() PlutusData {
	var res PlutusData
	enc, _ := cbor.Marshal(pd)
	cbor.Unmarshal(enc, &res)
	return res
}

func (pd *Datum) Clone() Datum {
	return Datum{
		PlutusDataType: pd.PlutusDataType,
		TagNr:          pd.TagNr,
		Value:          pd.Value,
	}
}

func (pd Datum) MarshalCBOR() ([]uint8, error) {
	if pd.TagNr == 0 {
		return cbor.Marshal(pd.Value)
	} else {
		return cbor.Marshal(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
	}
}
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

func (pd *PlutusData) Equal(other PlutusData) bool {
	marshaledThis, _ := cbor.Marshal(pd)
	marshaledOther, _ := cbor.Marshal(other)
	return bytes.Equal(marshaledThis, marshaledOther)
}

func (pd *PlutusData) ToDatum() Datum {

	var res Datum
	enc, _ := cbor.Marshal(pd)
	cbor.Unmarshal(enc, &res)
	return res
}

func (pd *PlutusData) Clone() PlutusData {
	return PlutusData{
		PlutusDataType: pd.PlutusDataType,
		TagNr:          pd.TagNr,
		Value:          pd.Value,
	}
}

type CborMap struct {
	Contents *map[serialization.CustomBytes]PlutusData
}

func (cm *CborMap) MarshalCBOR() ([]uint8, error) {
	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return em.Marshal(cm.Contents)
}

func (cm *CborMap) UnmarshalCBOR(value []uint8) error {
	return cbor.Unmarshal(value, cm.Contents)
}

func (pd *PlutusData) MarshalCBOR() ([]uint8, error) {
	if pd.TagNr == 0 {
		return cbor.Marshal(pd.Value)
	} else {
		return cbor.Marshal(cbor.Tag{Number: pd.TagNr, Content: pd.Value})
	}
}
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
			y := CborMap{
				Contents: new(map[serialization.CustomBytes]PlutusData),
			}
			err = cbor.Unmarshal(value, &y)
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

func ToCbor(x interface{}) string {
	bytes, err := cbor.Marshal(x)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

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

func PlutusScriptHash(script ScriptHashable) serialization.ScriptHash {
	return script.Hash()
}

type PlutusV1Script []byte

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
