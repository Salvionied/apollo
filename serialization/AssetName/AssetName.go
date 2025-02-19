package AssetName

import (
	"encoding/hex"
	"errors"

	"github.com/fxamacker/cbor/v2"
)

type AssetName struct {
	value string
}

/*
internal use only
*/
func NewAssetNameFromHexString(value string) *AssetName {
	_, err := hex.DecodeString(value)

	if err != nil || len(value) > 64 {
		return nil
	}

	return &AssetName{value: value}
}

func NewAssetNameFromString(value string) AssetName {
	v := hex.EncodeToString([]byte(value))
	return AssetName{value: v}
}

func (an AssetName) String() string {
	decoded, _ := hex.DecodeString(an.value)
	return string(decoded)
}

func (an AssetName) HexString() string {
	return an.value
}

func (an *AssetName) MarshalCBOR() ([]byte, error) {
	if an.value == "[]" || an.value == "" {
		return cbor.Marshal(make([]byte, 0))
	}

	if len(an.value) > 64 {
		return nil, errors.New("invalid asset name length")
	}

	byteSlice, _ := hex.DecodeString(an.value)

	return cbor.Marshal(byteSlice)
}

func (an *AssetName) UnmarshalCBOR(value []byte) error {
	var res []byte
	err := cbor.Unmarshal(value, &res)
	if err != nil {
		return err
	}

	if len(res) > 32 {
		return errors.New("invalid asset name length")
	}

	an.value = hex.EncodeToString(res)

	return nil
}
